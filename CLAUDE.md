# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run locally (from repo root)
go run cmd/server/main.go

# Build binary
go build -o main cmd/server/main.go

# Run all tests
go test ./...

# Run tests for a specific package
go test ./internal/service/...

# Run a single test
go test ./internal/service/... -run TestFunctionName

# Start infrastructure (Postgres + MinIO)
docker compose up postgres minio -d

# Start full stack (includes backend container)
docker compose up --build
```

## Required Environment Variables

Copy `.env` and populate before running locally. Required vars (no defaults — server will fatal without them):

| Variable | Purpose |
|---|---|
| `DATABASE_URL` | Postgres connection string (or use `DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/DB_NAME`) |
| `JWT_ACCESS_SECRET` | Signs 15-minute access tokens |
| `JWT_REFRESH_SECRET` | Signs 7-day refresh tokens |
| `MINIO_ENDPOINT` | MinIO host (e.g. `localhost:9000`) |
| `MINIO_ACCESS_KEY` | MinIO access key |
| `MINIO_SECRET_KEY` | MinIO secret key |

Optional flags (default `false`):
- `RUN_MIGRATION=true` — runs `migrations/schema.sql` on startup
- `RUN_SEED=true` — seeds admin user, default tags, root folder (requires `ADMIN_EMAIL`, `ADMIN_PASSWORD`, `ADMIN_NAME`)

Other defaults: `PORT=4000`, `API_VERSION=v1`, `MINIO_USE_SSL=false`.

The docker-compose Postgres listens on port **5430** (not 5432) to avoid conflicts.

## Architecture

The server follows a strict four-layer pattern wired together in `cmd/server/main.go`:

```
HTTP Request
    ↓
middleware (auth_middleware, role_middleware, rate_limiter)
    ↓
handler (internal/handler/)       — parses request, calls service, writes response
    ↓
service (internal/service/)       — business logic, orchestrates repo + storage
    ↓
repository (internal/repository/) — raw SQL against Postgres via pgx/v5 pool
```

**Storage (MinIO)** is called directly from the service layer, not the repository. Each user gets their own MinIO bucket named `user-<lowercase-username>`. File content lives in MinIO; only metadata (`file_key`, `file_size`, `file_type`, etc.) is stored in Postgres.

**Routes** (`internal/routes/`) register Gin route groups on the versioned base path `/dbms/v1`. All routes except `/auth/register`, `/auth/login`, `/auth/refresh`, `/auth/forgot-password`, and `/auth/reset-password` require a `Bearer` JWT in the `Authorization` header.

**Middleware chain:**
- `AuthMiddleware` — validates JWT and injects `userId`, `email`, `role` into Gin context
- `AdminOnly` — must follow `AuthMiddleware`; gates admin-only routes
- `RateLimitMiddleware` — in-memory 20 req/min per IP (not currently wired into main.go by default)

## Key Conventions

**Response shape** — always use helpers in `internal/utils/response.go`:
```go
utils.Success(c, data)    // 200 { "data": ..., "error": null }
utils.Created(c, data)    // 201 { "data": ..., "error": null }
utils.Error(c, status, msg) // { "data": null, "error": "..." }
```

**Error propagation** — services wrap errors with `fmt.Errorf("service name: %w", err)`. Handlers check for sentinel errors (`utils.ErrNotFound`) to map them to specific HTTP status codes.

**Database transactions** — use `db.WithTransaction(ctx, func(tx pgx.Tx) error { ... })` for multi-step writes. The pool is exposed as `db.Pool` for direct use in the audit utility.

**Audit logging** — `utils.LogAudit(ctx, db.Pool, entry)` inserts directly into `audit_logs`. There is also a higher-level `audit_service` / `audit_handler` for querying logs via the API.

**Migrations** — a single idempotent `migrations/schema.sql` runs via `RUN_MIGRATION=true`. All tables use `CREATE TABLE IF NOT EXISTS` and `CREATE INDEX IF NOT EXISTS`. The schema auto-sets `updated_at` via a Postgres trigger (`set_updated_at`).

**Roles** — three user roles enforced in the DB check constraint: `admin`, `editor`, `viewer`.

**Document soft-delete** — `documents.deleted_at` column; records are not physically removed until `Delete` is called (which also deletes the file from MinIO).

**Document sharing** — `document_shares` table stores a `share_token` (UUID), optional password hash, expiry, and permission level (`view`/`edit`/`download`).

**2FA** — TOTP-based via `pquerna/otp`. State tracked in `user_two_factor` table with `enabled`/`verified` flags and a DB-level check constraint enforcing valid state transitions.

## RAG / AI Chat System

The backend includes a Retrieval-Augmented Generation (RAG) pipeline that lets users ask questions answered from their own documents.

### Required env vars (add to `.env`)
```
OPENAI_API_KEY=sk-...          # embeddings (text-embedding-3-small, ~$0.02/1M tokens)
ANTHROPIC_API_KEY=sk-ant-...   # LLM answers (claude-haiku-4-5-20251001)
```

### How it works
```
Upload → ExtractText → Chunk (500 words, 50 overlap) → Embed (OpenAI) → pgvector

POST /chat/ask
  → Embed question → pgvector cosine search (top 5 chunks)
  → Build prompt (system + retrieved context + question history)
  → Claude Haiku → answer + source citations
```

### New packages
| Path | Purpose |
|---|---|
| `internal/rag/extractor.go` | Text extraction: PDF (`ledongthuc/pdf`), DOCX (ZIP+XML), plain text |
| `internal/rag/chunker.go` | Word-based chunking, 500 words / 50 overlap |
| `internal/rag/embedder.go` | OpenAI `text-embedding-3-small` — 1536-dim vectors |
| `internal/service/rag_service.go` | Orchestrates index + Ask flow, manages chat sessions |
| `internal/service/anthropic_client.go` | Direct HTTP client to Anthropic Messages API |
| `internal/repository/rag_repository.go` | pgvector similarity search, chunk CRUD, session/message CRUD |
| `internal/handler/rag_handler.go` | HTTP handlers for chat routes |
| `internal/routes/rag_routes.go` | Route registration under `/chat` |

### Database tables (added to migrations/schema.sql)
- `document_chunks` — stores text chunks with `vector(1536)` embedding column
- `chat_sessions` — one per conversation thread per user
- `chat_messages` — individual turns; `sources` JSONB column holds cited chunks

### Indexing
`IndexDocument` is called as a goroutine inside `documents_handler.go` on every upload. It is safe to call concurrently. Unsupported file types (e.g. images, video) are skipped with a log line. Re-uploading a document re-indexes it (old chunks deleted first).

### Chat API endpoints
| Method | Path | Description |
|---|---|---|
| `POST` | `/dbms/v1/chat/ask` | Ask a question — main RAG endpoint |
| `POST` | `/dbms/v1/chat/sessions` | Create a new chat session |
| `GET` | `/dbms/v1/chat/sessions` | List user's sessions |
| `GET` | `/dbms/v1/chat/sessions/:id` | Get session + full message history |
| `DELETE` | `/dbms/v1/chat/sessions/:id` | Delete session and all messages |

### pgvector setup
Run `CREATE EXTENSION IF NOT EXISTS vector;` once on the database before running migrations (or set `RUN_MIGRATION=true` — the schema handles it). The IVFFlat index on `document_chunks.embedding` requires at least ~100 rows to be effective; on a fresh DB with few documents cosine search still works via sequential scan.
