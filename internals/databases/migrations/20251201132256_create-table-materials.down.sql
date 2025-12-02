-- +migrate Down
/* =====================================================================
   MATERIALS
   - Drop:
       * student_class_material_progresses
       * class_materials
       * school_materials
   - Drop enums:
       * material_progress_status_enum
       * material_importance_enum
       * material_type_enum
   ===================================================================== */

BEGIN;

-- =====================================================================
-- DROP TABLES (reverse dependency order)
-- =====================================================================

-- 1) Progress murid per materi (punya FK ke class_materials, scsst, students)
DROP TABLE IF EXISTS student_class_material_progresses;

-- 2) Materi di level CSST (punya FK ke schools, csst, sessions, school_materials)
DROP TABLE IF EXISTS class_materials;

-- 3) Master materi di level sekolah
DROP TABLE IF EXISTS school_materials;

-- =====================================================================
-- DROP ENUMS (jika sudah tidak dipakai oleh objek lain)
-- =====================================================================

DROP TYPE IF EXISTS material_progress_status_enum;
DROP TYPE IF EXISTS material_importance_enum;
DROP TYPE IF EXISTS material_type_enum;

COMMIT;
