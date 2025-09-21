-- Putus SEMUA FK yang menunjuk ke class_sections, apapun namanya
DO $$
DECLARE
  r record;
BEGIN
  FOR r IN
    SELECT conname,
           quote_ident(nsp.nspname) || '.' || quote_ident(cls.relname) AS relname
    FROM pg_constraint c
    JOIN pg_class     cls ON cls.oid = c.conrelid
    JOIN pg_namespace nsp ON nsp.oid = cls.relnamespace
    WHERE c.contype = 'f'
      AND c.confrelid = 'class_sections'::regclass
  LOOP
    EXECUTE format('ALTER TABLE %s DROP CONSTRAINT IF EXISTS %I', r.relname, r.conname);
  END LOOP;
END$$;

-- (Opsional) Drop index-index; sebenarnya DROP TABLE akan otomatis menghapus index milik tabel itu.
DROP INDEX IF EXISTS uq_sections_slug_per_masjid_alive;
DROP INDEX IF EXISTS uq_sections_code_per_class_alive;
DROP INDEX IF EXISTS idx_sections_class_masjid_alive;
DROP INDEX IF EXISTS idx_sections_teacher_masjid_alive;
DROP INDEX IF EXISTS idx_sections_room_masjid_alive;
DROP INDEX IF EXISTS idx_sections_leader_student_masjid_alive;
DROP INDEX IF EXISTS idx_sections_masjid;
DROP INDEX IF EXISTS ix_sections_masjid_active_created;
DROP INDEX IF EXISTS ix_sections_class_active_created;
DROP INDEX IF EXISTS ix_sections_teacher_active_created;
DROP INDEX IF EXISTS ix_sections_room_active_created;
DROP INDEX IF EXISTS gin_sections_name_trgm_alive;
DROP INDEX IF EXISTS gin_sections_slug_trgm_alive;
DROP INDEX IF EXISTS idx_sections_group_url_alive;
DROP INDEX IF EXISTS brin_sections_created_at;
DROP INDEX IF EXISTS idx_sections_image_purge_due;

-- Terakhir: drop table
DROP TABLE IF EXISTS class_sections;
