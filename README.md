# go-dbms

The Go backend for **PetroData** — a document management system with mandatory two-factor authentication, per-user and link-based document sharing, activity reporting, and an AI-powered RAG chat over your own documents.

Built with [Gin](https://github.com/gin-gonic/gin), [pgx](https://github.com/jackc/pgx) (Postgres), and [MinIO](https://min.io) for file storage.

## Features

- **Auth & security** — two-step login (password, then a TOTP code from an authenticator app), mandatory 2FA setup for every account, one-time recovery codes, JWT access/refresh tokens, password reset via TOTP/recovery code (no email required), rate-limited login/reset endpoints, full audit logging.
- **Documents** — upload, versioning, folders, tags, starring, soft-delete to Trash (restore or empty permanently), optional ClamAV virus scanning and file-type/MIME blocking on upload.
- **Sharing** — public token-based share links (optional password + expiry) *and* direct per-user sharing (grant a specific account view/download access, with an in-app notification and a "Shared with Me" list).
- **Collaboration** — comments, document watchers, a review/approval queue, workspace-wide activity notifications.
- **Reports & audit** — every user can generate a report of their own activity (or, for admins, a system-wide view) across today / yesterday / last 7 days / last 30 days, plus a searchable audit log.
- **AI chat (RAG)** — ask questions answered from your own uploaded documents (PDF/DOCX/text), backed by pgvector similarity search.
- **Admin** — user management, department stats, license-free role model (admin / editor / viewer).

## Tech stack

| Layer | Choice |
|---|---|
| Language | Go 1.25 |
| HTTP framework | [Gin](https://github.com/gin-gonic/gin) |
| Database | PostgreSQL + [pgvector](https://github.com/pgvector/pgvector) (via [pgx/v5](https://github.com/jackc/pgx)) |
| Object storage | MinIO (S3-compatible) |
| Auth | JWT (access + refresh) with TOTP 2FA ([pquerna/otp](https://github.com/pquerna/otp)) |
| Virus scanning | ClamAV (optional) |
| Embeddings / LLM | OpenAI or Ollama (embeddings), Claude / Gemini / Groq / Ollama (chat) |

## Architecture

Requests flow through a strict four-layer pattern, wired together in `cmd/server/main.go`:

```text
HTTP request
    → middleware   (internal/middleware)  — auth, admin-only, rate limiting
    → handler      (internal/handler)     — parses the request, calls the service, writes the response
    → service      (internal/service)     — business logic, orchestrates repos + storage
    → repository   (internal/repository)  — raw SQL against Postgres via pgx
```

Object storage (MinIO) is called directly from the service layer. Each user gets their own bucket; file *content* lives in MinIO, only metadata lives in Postgres.

## Getting started

### Prerequisites

- Go 1.25+
- PostgreSQL (with the [`pgvector`](https://github.com/pgvector/pgvector) extension available)
- MinIO (or any S3-compatible store)
- Docker + Docker Compose (optional, but the easiest way to get Postgres/MinIO/ClamAV running locally)

### 1. Clone and configure

```bash
git clone <repo-url>
cd go-dbms
cp .env .env.local   # or create your own .env — see Environment variables below
```

### 2. Start infrastructure

```bash
docker compose up postgres minio -d
```

### 3. Run the server

```bash
go run cmd/server/main.go
```

On first run, set `RUN_MIGRATION=true` (applies `migrations/schema.sql`, idempotent) and `RUN_SEED=true` (creates the first admin account from `ADMIN_EMAIL` / `ADMIN_PASSWORD` / `ADMIN_NAME`) in your `.env`.

The server listens on `PORT` (default `4000`) under the base path `/dbms/v1`.

### 4. Build a binary

```bash
go build -o main cmd/server/main.go
```

### Run the full stack with Docker

```bash
docker compose up --build
```

Brings up Postgres, MinIO, ClamAV, and the backend together. The compose Postgres listens on host port **5430** (mapped to container `5432`) to avoid clashing with a local Postgres install.

## Environment variables

Copy `.env` and fill in your own values. Variables with no default are **required** — the server will fail fast at startup if they're missing.

| Variable | Required | Purpose |
|---|---|---|
| `DATABASE_URL` *(or `DB_HOST`/`DB_PORT`/`DB_USER`/`DB_PASSWORD`/`DB_NAME`)* | ✅ | Postgres connection |
| `JWT_ACCESS_SECRET` | ✅ | Signs short-lived access tokens |
| `JWT_REFRESH_SECRET` | ✅ | Signs refresh tokens |
| `JWT_2FA_CHALLENGE_SECRET` | ✅ | Signs the short-lived, non-API "password verified, enter your 2FA code" challenge issued between login steps |
| `MINIO_ENDPOINT` / `MINIO_ACCESS_KEY` / `MINIO_SECRET_KEY` | ✅ | Object storage |
| `PORT` | – | HTTP port (default `4000`) |
| `API_VERSION` | – | Base path version segment (default `v1`, i.e. `/dbms/v1`) |
| `MINIO_USE_SSL` | – | `true`/`false` (default `false`) |
| `RUN_MIGRATION` | – | Apply `migrations/schema.sql` on startup |
| `RUN_SEED` | – | Seed the first admin user (`ADMIN_EMAIL`, `ADMIN_PASSWORD`, `ADMIN_NAME`, `ADMIN_DEPARTMENT`) |
| `CLAMAV_URL` | – | e.g. `clamd:3310` — enables virus scanning on upload if set |
| `CLAMAV_REQUIRED` | – | `true` to reject uploads on a scan failure/timeout; otherwise scanning fails open |
| `SMTP_HOST` / `SMTP_PORT` / `SMTP_USER` / `SMTP_PASS` / `SMTP_FROM` | – | Only used for optional comment/share email notifications — **not** required for password reset, which is TOTP/recovery-code based |
| `RAG_EMBED_PROVIDER` / `RAG_CHAT_PROVIDER` | – | `ollama` (default, free/local), or `openai`/`gemini`/`groq`/`anthropic` — see `.env` for the matching API key per provider |

## API overview

All routes are under `/dbms/v1`. Every route requires `Authorization: Bearer <access token>` **except**:

- `POST /auth/register`, `POST /auth/login`, `POST /auth/login/verify`, `POST /auth/refresh`, `POST /auth/reset-password`
- `GET /shares/public/:token`, `POST /shares/public/:token/download` (public share links)
- `GET /share/:token` equivalents used by the public share viewer
- `GET /health`

### Auth flow

Login is two steps:

1. `POST /auth/login` `{username, password}` → on success, returns a short-lived challenge (never an access token) and a status of `2fa_required` (existing, verified account) or `2fa_setup_required` (new account, or an existing one that hasn't set up 2FA yet — walks them through scanning a QR code).
2. `POST /auth/login/verify` `{login_challenge, code}` → `code` is a 6-digit TOTP code or a one-time recovery code. Issues the real access/refresh tokens. The first time an account completes this, it also returns a fresh set of recovery codes (shown once).

Route groups (see `internal/routes/`): `auth`, `users`, `documents`, `folders`, `tags`, `shares` (link-based), `documents/:id/user-shares` + `shared-with-me` (direct per-user sharing), `versions`, `trash`, `stats`, `bulk`, `audit`, `comments`, `notifications`, `reviews`, `watchers`, `reports`, `chat` (RAG).

## Project structure

```text
cmd/server/main.go       — composition root: config, DB, storage, wiring, route registration
internal/
  config/                — env var loading
  db/                    — connection pool, migration runner, seed data
  middleware/            — auth, admin-only, rate limiting
  handler/                — HTTP request/response per route group
  service/                — business logic
  repository/             — SQL access
  models/                — shared structs
  routes/                 — route registration per group
  storage/                — MinIO client
  rag/                    — text extraction, chunking, embeddings for the chat feature
  utils/                  — JWT, password hashing, audit logging, file-type/virus scanning, response helpers
migrations/schema.sql    — single idempotent schema file
```

## Testing

```bash
go test ./...                                  # all tests
go test ./internal/service/...                 # a specific package
go test ./internal/service/... -run TestName    # a single test
```
