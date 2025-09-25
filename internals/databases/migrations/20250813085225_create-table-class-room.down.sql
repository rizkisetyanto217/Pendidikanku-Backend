-- +migrate Down
-- =========================================================
-- DOWN — Class Rooms (+ Virtual Links, URLs)
-- =========================================================

-- ----------------------------
-- CHILD 1: class_room_urls
-- ----------------------------
-- (opsional) explicit drop indexes
DROP INDEX IF EXISTS idx_room_urls_tenant_alive;
DROP INDEX IF EXISTS idx_room_urls_href_trgm;
DROP INDEX IF EXISTS idx_room_urls_label_trgm;
DROP INDEX IF EXISTS idx_room_urls_room_kind_alive;
DROP INDEX IF EXISTS uq_room_urls_href_ci;
DROP INDEX IF EXISTS uq_room_urls_object_key_ci;
DROP INDEX IF EXISTS uq_room_urls_label_ci;
DROP INDEX IF EXISTS uq_room_urls_primary_per_kind;

DROP TABLE IF EXISTS class_room_urls;

-- ----------------------------
-- CHILD 2: class_room_virtual_links
-- ----------------------------
-- (opsional) explicit drop indexes
DROP INDEX IF EXISTS idx_room_vlinks_tenant_platform;
DROP INDEX IF EXISTS idx_room_vlinks_platform_active;
DROP INDEX IF EXISTS idx_room_vlinks_active;
DROP INDEX IF EXISTS uq_room_vlinks_url_ci;
DROP INDEX IF EXISTS uq_room_vlinks_label_ci;

DROP TABLE IF EXISTS class_room_virtual_links;

-- ----------------------------
-- ENUM: virtual_platform_enum
-- (hapus hanya jika sudah tidak dipakai tabel lain)
-- ----------------------------
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'virtual_platform_enum') THEN
    EXECUTE 'DROP TYPE virtual_platform_enum';
  END IF;
END$$;

-- ----------------------------
-- PARENT: class_rooms
-- ----------------------------
-- (opsional) explicit drop indexes
DROP INDEX IF EXISTS brin_class_rooms_created_at;
DROP INDEX IF EXISTS idx_class_rooms_location_trgm_alive;
DROP INDEX IF EXISTS idx_class_rooms_name_trgm_alive;
DROP INDEX IF EXISTS idx_class_rooms_features_gin_alive;
DROP INDEX IF EXISTS idx_class_rooms_tenant_active_alive;
DROP INDEX IF EXISTS idx_class_rooms_masjid_alive;
DROP INDEX IF EXISTS uq_class_rooms_tenant_slug_ci_alive;
DROP INDEX IF EXISTS uq_class_rooms_tenant_code_ci_alive;
DROP INDEX IF EXISTS uq_class_rooms_tenant_name_ci_alive;

DROP TABLE IF EXISTS class_rooms;

-- Catatan:
-- - EXTENSIONS (pgcrypto, pg_trgm) sengaja tidak di-drop di Down.
-- - Jika ada objek lain yang bergantung pada class_rooms (mis. schedules),
--   pastikan di-drop lebih dulu dalam migration terpisah sebelum blok ini.
