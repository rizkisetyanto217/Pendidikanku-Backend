-- +migrate Down
-- =========================================================
-- DOWN — User Notes (+ URLs + Types)
-- =========================================================

-- -----------------------------
-- 1) CHILD: user_note_urls
-- (DROP INDEX opsional; tabel drop otomatis menghapus index)
-- -----------------------------
DROP INDEX IF EXISTS brin_user_note_urls_created_at;
DROP INDEX IF EXISTS idx_user_note_urls_masjid_kind;
DROP INDEX IF EXISTS gin_user_note_urls_title_trgm;
DROP INDEX IF EXISTS gin_user_note_urls_url_trgm;
DROP INDEX IF EXISTS gin_user_note_urls_tags;
DROP INDEX IF EXISTS uq_user_note_urls_one_primary;
DROP INDEX IF EXISTS idx_user_note_urls_note_sort;

DROP TABLE IF EXISTS user_note_urls;

-- -----------------------------
-- 2) ENUM lokal: user_note_url_kind_enum
--    (hapus hanya jika tidak ada dependensi tersisa)
-- -----------------------------
DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM pg_type t
    WHERE t.typname = 'user_note_url_kind_enum'
      AND NOT EXISTS (
        SELECT 1
        FROM pg_depend d
        WHERE d.refobjid = t.oid
          AND d.deptype IN ('n','a','i','e','p')
      )
  ) THEN
    EXECUTE 'DROP TYPE user_note_url_kind_enum';
  END IF;
END$$;

-- -----------------------------
-- 3) PARENT: user_notes
-- -----------------------------
DROP INDEX IF EXISTS gin_user_notes_content_trgm;
DROP INDEX IF EXISTS gin_user_notes_title_trgm;
DROP INDEX IF EXISTS idx_user_notes_due_date;
DROP INDEX IF EXISTS idx_user_notes_pinned_partial;
DROP INDEX IF EXISTS idx_user_notes_labels_gin;
DROP INDEX IF EXISTS idx_user_notes_type_id;
DROP INDEX IF EXISTS idx_user_notes_class_scope_created;
DROP INDEX IF EXISTS idx_user_notes_student_created;
DROP INDEX IF EXISTS idx_user_notes_scope;
DROP INDEX IF EXISTS idx_user_notes_author_teacher;
DROP INDEX IF EXISTS idx_user_notes_author;
DROP INDEX IF EXISTS idx_user_notes_masjid;

DROP TABLE IF EXISTS user_notes;

-- -----------------------------
-- 4) MASTER: user_note_types
-- -----------------------------
DROP INDEX IF EXISTS idx_note_types_owner_active_sort;
DROP INDEX IF EXISTS idx_note_types_masjid;

DROP TABLE IF EXISTS user_note_types;
