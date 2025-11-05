-- +migrate Down
BEGIN;

DROP INDEX IF EXISTS uq_student_class_sections_active_per_student_per_class;

DROP INDEX IF EXISTS gin_student_class_enrollments_prefs;
DROP INDEX IF EXISTS ix_student_class_enrollments_status_created;
DROP INDEX IF EXISTS ix_student_class_enrollments_tenant_class_created;
DROP INDEX IF EXISTS ix_student_class_enrollments_tenant_student_created;
DROP INDEX IF EXISTS uq_student_class_enrollments_active_per_student_class;

DROP TABLE IF EXISTS student_class_enrollments;

-- (ENUM dibiarkan agar aman untuk migrasi lain)
-- DROP TYPE IF EXISTS class_enrollment_status;

COMMIT;
