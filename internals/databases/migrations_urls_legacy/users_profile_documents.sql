
-- =========================================================
-- 2) USERS_PROFILE_DOCUMENTS (fresh)
-- =========================================================
CREATE TABLE IF NOT EXISTS users_profile_documents (
  id                        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id                   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  doc_type                  VARCHAR(50) NOT NULL,
  file_url                  TEXT NOT NULL,
  file_trash_url            TEXT,
  file_delete_pending_until TIMESTAMPTZ,
  uploaded_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at                TIMESTAMPTZ,
  deleted_at                TIMESTAMPTZ
);

-- Partial UNIQUE (hanya row alive) → friendly ke soft-delete
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_doc_type_alive
  ON users_profile_documents(user_id, doc_type)
  WHERE deleted_at IS NULL;

-- Index umum (alive rows)
CREATE INDEX IF NOT EXISTS idx_users_profile_documents_user_alive
  ON users_profile_documents(user_id) WHERE deleted_at IS NULL;

-- Query umum: (user_id, doc_type) alive
CREATE INDEX IF NOT EXISTS idx_users_profile_documents_user_type_alive
  ON users_profile_documents(user_id, doc_type)
  WHERE deleted_at IS NULL;

-- GC: due cleanup
CREATE INDEX IF NOT EXISTS idx_users_profile_documents_gc_due
  ON users_profile_documents(file_delete_pending_until)
  WHERE file_trash_url IS NOT NULL;

-- (opsional) urut terbaru per user
CREATE INDEX IF NOT EXISTS idx_users_profile_documents_user_uploaded_alive
  ON users_profile_documents(user_id, uploaded_at DESC)
  WHERE deleted_at IS NULL;


COMMIT;
