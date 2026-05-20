-- ============================================================
-- EXTENSIONS
-- ============================================================
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================
-- USERS
-- ============================================================
CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  email TEXT NOT NULL,
  password_hash TEXT NOT NULL,
  name TEXT NOT NULL,

  role TEXT NOT NULL DEFAULT 'viewer'
    CHECK (role IN ('admin', 'editor', 'viewer')),

  department TEXT,
  avatar_url TEXT,

  status TEXT NOT NULL DEFAULT 'active'
    CHECK (status IN ('active', 'inactive', 'suspended')),

  last_login TIMESTAMPTZ,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ✅ case-insensitive unique email
CREATE UNIQUE INDEX idx_users_email_ci ON users (LOWER(email));

CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_department ON users(department);

-- ============================================================
-- REFRESH TOKENS
-- ============================================================
CREATE TABLE IF NOT EXISTS refresh_tokens (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL UNIQUE,

  expires_at TIMESTAMPTZ NOT NULL,
  revoked BOOLEAN NOT NULL DEFAULT false,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_refresh_tokens_user ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_expiry ON refresh_tokens(expires_at);

-- ============================================================
-- FOLDERS (HIERARCHY)
-- ============================================================
CREATE TABLE IF NOT EXISTS folders (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  name TEXT NOT NULL,
  parent_id UUID REFERENCES folders(id) ON DELETE CASCADE,

  -- ✅ enforce ownership
  owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

  department TEXT,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ✅ avoid duplicate folders under same parent
ALTER TABLE folders
ADD CONSTRAINT uniq_folder UNIQUE (name, parent_id, owner_id);

CREATE INDEX idx_folders_parent ON folders(parent_id);
CREATE INDEX idx_folders_owner ON folders(owner_id);

-- ============================================================
-- TAGS
-- ============================================================
CREATE TABLE IF NOT EXISTS tags (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  name TEXT NOT NULL,
  color TEXT NOT NULL DEFAULT '#3B82F6',

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ✅ case-insensitive tag names
CREATE UNIQUE INDEX idx_tags_name_ci ON tags (LOWER(name));

-- ============================================================
-- DOCUMENTS (CORE TABLE)
-- ============================================================
CREATE TABLE IF NOT EXISTS documents (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  title TEXT NOT NULL,
  description TEXT,

  file_name TEXT NOT NULL,

  -- ✅ IMPORTANT: this is MinIO object key
  file_key TEXT NOT NULL UNIQUE,

  file_type TEXT NOT NULL,
  file_size BIGINT NOT NULL,

  folder_id UUID REFERENCES folders(id) ON DELETE SET NULL,
  owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

  department TEXT,

  status TEXT NOT NULL DEFAULT 'draft'
    CHECK (status IN ('draft', 'published', 'archived')),

  is_starred BOOLEAN NOT NULL DEFAULT false,

  version INTEGER NOT NULL DEFAULT 1,

  last_accessed TIMESTAMPTZ,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_documents_owner ON documents(owner_id);
CREATE INDEX idx_documents_folder ON documents(folder_id);
CREATE INDEX idx_documents_status ON documents(status);
CREATE INDEX idx_documents_updated ON documents(updated_at DESC);

-- ✅ FULL-TEXT SEARCH (important)
CREATE INDEX idx_documents_search
ON documents USING gin(
  to_tsvector('english', title || ' ' || COALESCE(description, ''))
);

-- ============================================================
-- DOCUMENT TAG RELATION
-- ============================================================
CREATE TABLE IF NOT EXISTS document_tags (
  document_id UUID REFERENCES documents(id) ON DELETE CASCADE,
  tag_id UUID REFERENCES tags(id) ON DELETE CASCADE,
  PRIMARY KEY (document_id, tag_id)
);

-- ============================================================
-- DOCUMENT VERSIONS
-- ============================================================
CREATE TABLE IF NOT EXISTS document_versions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  version INTEGER NOT NULL,

  file_key TEXT NOT NULL, -- ✅ version stored in MinIO
  file_size BIGINT NOT NULL,

  uploaded_by UUID REFERENCES users(id) ON DELETE SET NULL,
  change_note TEXT,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  UNIQUE(document_id, version)
);

CREATE INDEX idx_versions_doc ON document_versions(document_id);

-- ============================================================
-- DOCUMENT SHARING
-- ============================================================
CREATE TABLE IF NOT EXISTS document_shares (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  share_token TEXT NOT NULL UNIQUE,

  shared_by UUID REFERENCES users(id) ON DELETE SET NULL,

  permission TEXT NOT NULL DEFAULT 'view'
    CHECK (permission IN ('view', 'edit', 'download')),

  password_hash TEXT,
  expires_at TIMESTAMPTZ,

  access_count INTEGER NOT NULL DEFAULT 0,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_shares_doc ON document_shares(document_id);
CREATE INDEX idx_shares_expiry ON document_shares(expires_at);

-- ============================================================
-- AUDIT LOGS
-- ============================================================
CREATE TABLE IF NOT EXISTS audit_logs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  user_id UUID REFERENCES users(id) ON DELETE SET NULL,

  action TEXT NOT NULL,
  resource_type TEXT NOT NULL,
  resource_id UUID,

  details JSONB,

  ip_address TEXT,
  user_agent TEXT,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_user ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX idx_audit_logs_created ON audit_logs(created_at DESC);

-- ============================================================
-- AUTO UPDATE TIMESTAMP TRIGGERS
-- ============================================================
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_users_updated
BEFORE UPDATE ON users
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_folders_updated
BEFORE UPDATE ON folders
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_documents_updated
BEFORE UPDATE ON documents
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
