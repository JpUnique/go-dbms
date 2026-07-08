-- ===============================
-- EXTENSIONS
-- ===============================
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ===============================
-- USERS
-- ===============================
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

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_name_ci ON users (LOWER(name));
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_ci ON users (LOWER(email));

CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
CREATE INDEX IF NOT EXISTS idx_users_department ON users(department);

-- ===============================
-- USER TWO FACTOR AUTHENTICATION
-- ===============================
CREATE TABLE IF NOT EXISTS user_two_factor (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  secret TEXT,
  enabled BOOLEAN NOT NULL DEFAULT false,
  verified BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (enabled = false OR secret IS NOT NULL),
  CHECK (verified = false OR enabled = true)
);

-- ===============================
-- USER RECOVERY CODES (2FA backup)
-- ===============================
CREATE TABLE IF NOT EXISTS user_recovery_codes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  code_hash TEXT NOT NULL,
  used_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_recovery_codes_user ON user_recovery_codes(user_id);

-- ===============================
-- USER PREFERENCES
-- ===============================
CREATE TABLE IF NOT EXISTS user_preferences (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  dark_mode BOOLEAN DEFAULT false,
  email_notifications BOOLEAN DEFAULT true,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- ===============================
-- REFRESH TOKENS
-- ===============================
CREATE TABLE IF NOT EXISTS refresh_tokens (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL UNIQUE,

  expires_at TIMESTAMPTZ NOT NULL,
  revoked BOOLEAN NOT NULL DEFAULT false,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user ON refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expiry ON refresh_tokens(expires_at);

-- ===============================
-- FOLDERS
-- ===============================
CREATE TABLE IF NOT EXISTS folders (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  name TEXT NOT NULL,
  parent_id UUID REFERENCES folders(id) ON DELETE CASCADE,

  owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  department TEXT,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ✅ SAFE UNIQUE CONSTRAINT (CORRECT PLACEMENT)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'uniq_folder'
    ) THEN
        ALTER TABLE folders
        ADD CONSTRAINT uniq_folder UNIQUE (name, parent_id, owner_id);
    END IF;
END$$;

-- indexes for folders
CREATE INDEX IF NOT EXISTS idx_folders_parent ON folders(parent_id);
CREATE INDEX IF NOT EXISTS idx_folders_owner ON folders(owner_id);

-- ===============================
-- TAGS
-- ===============================
CREATE TABLE IF NOT EXISTS tags (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  name TEXT NOT NULL,
  color TEXT NOT NULL DEFAULT '#3B82F6',

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_tags_name_ci ON tags (LOWER(name));

-- ===============================
-- DOCUMENTS
-- ===============================
CREATE TABLE IF NOT EXISTS documents (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  title TEXT NOT NULL,
  description TEXT,

  file_name TEXT NOT NULL,
  file_key TEXT NOT NULL UNIQUE,

  file_type TEXT NOT NULL,
  file_size BIGINT NOT NULL,

  folder_id UUID REFERENCES folders(id) ON DELETE SET NULL,
  owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

  department TEXT,

  status TEXT NOT NULL DEFAULT 'draft'
    CHECK (status IN ('draft', 'published', 'archived', 'pending_review')),

  is_starred BOOLEAN NOT NULL DEFAULT false,

  version INTEGER NOT NULL DEFAULT 1,

  last_accessed TIMESTAMPTZ,
  deleted_at TIMESTAMPTZ,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_documents_owner ON documents(owner_id);
CREATE INDEX IF NOT EXISTS idx_documents_folder ON documents(folder_id);
CREATE INDEX IF NOT EXISTS idx_documents_status ON documents(status);
CREATE INDEX IF NOT EXISTS idx_documents_owner_status ON documents(owner_id, status);
CREATE INDEX IF NOT EXISTS idx_documents_updated ON documents(updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_documents_department ON documents(department);

CREATE INDEX IF NOT EXISTS idx_documents_search
ON documents USING gin (
  to_tsvector('english', title || ' ' || COALESCE(description, ''))
);

-- ===============================
-- DOCUMENT TAG RELATION
-- ===============================
CREATE TABLE IF NOT EXISTS document_tags (
  document_id UUID REFERENCES documents(id) ON DELETE CASCADE,
  tag_id UUID REFERENCES tags(id) ON DELETE CASCADE,
  PRIMARY KEY (document_id, tag_id)
);

-- ===============================
-- DOCUMENT VERSIONS
-- ===============================
CREATE TABLE IF NOT EXISTS document_versions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  version INTEGER NOT NULL,

  file_key TEXT NOT NULL,
  file_size BIGINT NOT NULL,

  uploaded_by UUID REFERENCES users(id) ON DELETE SET NULL,
  change_note TEXT,

  access_count INTEGER NOT NULL DEFAULT 0,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  UNIQUE(document_id, version)
);

CREATE INDEX IF NOT EXISTS idx_versions_doc ON document_versions(document_id);

-- ===============================
-- DOCUMENT SHARING
-- ===============================
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

CREATE INDEX IF NOT EXISTS idx_shares_doc ON document_shares(document_id);
CREATE INDEX IF NOT EXISTS idx_shares_token ON document_shares(share_token);
CREATE INDEX IF NOT EXISTS idx_shares_expiry ON document_shares(expires_at);

-- ===============================
-- AUDIT LOGS
-- ===============================
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

CREATE INDEX IF NOT EXISTS idx_audit_logs_user ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created ON audit_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);

-- ===============================
-- PASSWORD RESETS (removed — replaced by TOTP/recovery-code reset, see user_recovery_codes)
-- ===============================
DROP TABLE IF EXISTS password_resets;

-- ===============================
-- TRIGGER FUNCTION
-- ===============================
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER trg_users_updated
BEFORE UPDATE ON users
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE OR REPLACE TRIGGER trg_folders_updated
BEFORE UPDATE ON folders
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE OR REPLACE TRIGGER trg_documents_updated
BEFORE UPDATE ON documents
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE OR REPLACE TRIGGER trg_user_two_factor_updated
BEFORE UPDATE ON user_two_factor
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ===============================
-- DOCUMENT COMMENTS
-- ===============================
CREATE TABLE IF NOT EXISTS document_comments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  user_id     UUID NOT NULL REFERENCES users(id)     ON DELETE CASCADE,

  content TEXT NOT NULL,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_comments_document ON document_comments(document_id);
CREATE INDEX IF NOT EXISTS idx_comments_user     ON document_comments(user_id);

CREATE OR REPLACE TRIGGER trg_comments_updated
BEFORE UPDATE ON document_comments
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ===============================
-- IN-APP NOTIFICATIONS
-- ===============================
CREATE TABLE IF NOT EXISTS notifications (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

  type        TEXT NOT NULL,   -- comment_added | doc_shared | review_submitted | review_approved | review_rejected | mentioned | doc_updated
  title       TEXT NOT NULL,
  body        TEXT NOT NULL DEFAULT '',
  resource_type TEXT NOT NULL DEFAULT 'document',
  resource_id UUID,

  is_read BOOLEAN NOT NULL DEFAULT false,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notifications_user    ON notifications(user_id);
CREATE INDEX IF NOT EXISTS idx_notifications_unread  ON notifications(user_id, is_read) WHERE is_read = false;
CREATE INDEX IF NOT EXISTS idx_notifications_created ON notifications(created_at DESC);

-- ===============================
-- DOCUMENT REVIEWS (APPROVAL WORKFLOW)
-- ===============================
CREATE TABLE IF NOT EXISTS document_reviews (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  document_id  UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  submitter_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  reviewer_id  UUID REFERENCES users(id) ON DELETE SET NULL,

  decision TEXT NOT NULL DEFAULT 'pending'
    CHECK (decision IN ('pending', 'approved', 'rejected')),

  note TEXT,

  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  reviewed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_reviews_document  ON document_reviews(document_id);
CREATE INDEX IF NOT EXISTS idx_reviews_submitter ON document_reviews(submitter_id);
CREATE INDEX IF NOT EXISTS idx_reviews_pending   ON document_reviews(decision) WHERE decision = 'pending';

-- ===============================
-- DOCUMENT WATCHERS (COLLABORATION)
-- ===============================
CREATE TABLE IF NOT EXISTS document_watchers (
  document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  user_id     UUID NOT NULL REFERENCES users(id)     ON DELETE CASCADE,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (document_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_watchers_document ON document_watchers(document_id);
CREATE INDEX IF NOT EXISTS idx_watchers_user     ON document_watchers(user_id);

-- ===============================
-- DOCUMENT USER SHARES (direct per-user sharing — distinct from the
-- token-based document_shares links above; grants a specific account
-- view/download access to a document)
-- ===============================
CREATE TABLE IF NOT EXISTS document_user_shares (
  document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  user_id     UUID NOT NULL REFERENCES users(id)     ON DELETE CASCADE,
  permission  TEXT NOT NULL DEFAULT 'view' CHECK (permission IN ('view', 'download')),
  shared_by   UUID REFERENCES users(id) ON DELETE SET NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (document_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_user_shares_document ON document_user_shares(document_id);
CREATE INDEX IF NOT EXISTS idx_user_shares_user     ON document_user_shares(user_id);

-- Update status constraint to include pending_review on existing databases
DO $$ BEGIN
  ALTER TABLE documents DROP CONSTRAINT IF EXISTS documents_status_check;
  ALTER TABLE documents ADD CONSTRAINT documents_status_check
    CHECK (status IN ('draft', 'published', 'archived', 'pending_review'));
EXCEPTION WHEN others THEN NULL;
END $$;

-- Add columns that may be missing from older database schemas
ALTER TABLE documents ADD COLUMN IF NOT EXISTS is_starred   BOOLEAN     NOT NULL DEFAULT false;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS version      INTEGER     NOT NULL DEFAULT 1;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS description  TEXT;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS department   TEXT;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS last_accessed TIMESTAMPTZ;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS deleted_at   TIMESTAMPTZ;

ALTER TABLE users ADD COLUMN IF NOT EXISTS department TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_url TEXT;
