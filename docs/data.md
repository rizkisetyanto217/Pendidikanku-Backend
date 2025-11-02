-- =========================================================
-- Extensions (idempotent)
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS btree_gin;

-- =========================================================
-- Utility: generic touch updated_at
-- =========================================================
CREATE OR REPLACE FUNCTION fn_touch_updated_at_generic()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at := CURRENT_TIMESTAMP;
  RETURN NEW;
END$$ LANGUAGE plpgsql;

-- =========================================================
-- (Opsional kuat) ACADEMIC TERMS / SEMESTERS
-- =========================================================
CREATE TABLE IF NOT EXISTS academic_terms (
academic_terms_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
academic_terms_school_id UUID NOT NULL REFERENCES schools(school_id) ON DELETE CASCADE,
academic_terms_academic_year TEXT NOT NULL, -- "2025/2026"
academic_terms_name TEXT NOT NULL, -- "Ganjil" | "Genap" | dst.
academic_terms_start_date DATE NOT NULL,
academic_terms_end_date DATE NOT NULL,
academic_terms_is_active BOOLEAN NOT NULL DEFAULT TRUE,

created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
updated_at TIMESTAMP,
deleted_at TIMESTAMP,

CHECK (academic_terms_end_date >= academic_terms_start_date)
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_academic_terms_tenant_year_name_live
ON academic_terms(academic_terms_school_id, academic_terms_academic_year, academic_terms_name)
WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_academic_terms_tenant_active_live
ON academic_terms(academic_terms_school_id, academic_terms_is_active)
WHERE deleted_at IS NULL;

DROP TRIGGER IF EXISTS trg_touch_academic_terms ON academic_terms;
CREATE TRIGGER trg_touch_academic_terms
BEFORE UPDATE ON academic_terms
FOR EACH ROW EXECUTE FUNCTION fn_touch_updated_at_generic();

-- =========================================================
-- 1) CLASS SUBJECT GRADING POLICIES
-- =========================================================
CREATE TABLE IF NOT EXISTS class_subject_grading_policies (
class_subject_grading_policies_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

class_subject_grading_policies_school_id UUID NOT NULL REFERENCES schools(school_id) ON DELETE CASCADE,
class_subject_grading_policies_class_subject_id UUID NOT NULL REFERENCES class_subjects(class_subjects_id) ON DELETE CASCADE,

-- null = berlaku lintas tahun (jika lembaga tidak membedakan)
class_subject_grading_policies_academic_year TEXT,

-- JSONB bobot & aturan (sesuai desain)
class_subject_grading_policies_weights JSONB NOT NULL, -- {daily, uts, uas, behavior}
class_subject_grading_policies_daily_rules JSONB,
class_subject_grading_policies_uts_rules JSONB,
class_subject_grading_policies_uas_rules JSONB,
class_subject_grading_policies_behavior_rules JSONB,

class_subject_grading_policies_is_active BOOLEAN NOT NULL DEFAULT TRUE,

created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
updated_at TIMESTAMP,
deleted_at TIMESTAMP,

-- Sum bobot harus 100 (0..100 integer)
CONSTRAINT ck_csgp_weights_sum_100 CHECK (
COALESCE((class_subject_grading_policies_weights->>'daily')::int, 0) +
COALESCE((class_subject_grading_policies_weights->>'uts')::int, 0) +
COALESCE((class_subject_grading_policies_weights->>'uas')::int, 0) +
COALESCE((class_subject_grading_policies_weights->>'behavior')::int, 0)
= 100
)
);

-- Satu policy AKTIF per (school, class_subject, academic_year) yang masih live
CREATE UNIQUE INDEX IF NOT EXISTS uq_csgp_active_per_cs_year_live
ON class_subject_grading_policies (
class_subject_grading_policies_school_id,
class_subject_grading_policies_class_subject_id,
COALESCE(class_subject_grading_policies_academic_year,'')
)
WHERE class_subject_grading_policies_is_active = TRUE
AND deleted_at IS NULL;

-- Bantu query
CREATE INDEX IF NOT EXISTS ix_csgp_tenant_cs_live
ON class_subject_grading_policies (class_subject_grading_policies_school_id, class_subject_grading_policies_class_subject_id)
WHERE deleted_at IS NULL;

DROP TRIGGER IF EXISTS trg_touch_csgp ON class_subject_grading_policies;
CREATE TRIGGER trg_touch_csgp
BEFORE UPDATE ON class_subject_grading_policies
FOR EACH ROW EXECUTE FUNCTION fn_touch_updated_at_generic();

-- =========================================================
-- 2) CLASS SUBJECT ASSESSMENTS (DAILY / UTS / UAS)
-- =========================================================
CREATE TABLE IF NOT EXISTS class_subject_assessments (
class_subject_assessments_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

class_subject_assessments_school_id UUID NOT NULL REFERENCES schools(school_id) ON DELETE CASCADE,
class_subject_assessments_class_subject_id UUID NOT NULL REFERENCES class_subjects(class_subjects_id) ON DELETE CASCADE,

class_subject_assessments_type TEXT NOT NULL
CHECK (class_subject_assessments_type IN ('daily','uts','uas')),

class_subject_assessments_title TEXT,
class_subject_assessments_date DATE NOT NULL DEFAULT CURRENT_DATE,

-- Max skor & (opsional) bobot per-komponen (untuk weighted average level harian)
class_subject_assessments_max_score NUMERIC(6,2) NOT NULL CHECK (class_subject_assessments_max_score > 0),
class_subject_assessments_weight NUMERIC(6,2) CHECK (class_subject_assessments_weight IS NULL OR class_subject_assessments_weight >= 0),

class_subject_assessments_is_active BOOLEAN NOT NULL DEFAULT TRUE,

created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
updated_at TIMESTAMP,
deleted_at TIMESTAMP
);

-- Index umum
CREATE INDEX IF NOT EXISTS ix_csa_tenant_cs_date_live
ON class_subject_assessments (class_subject_assessments_school_id, class_subject_assessments_class_subject_id, class_subject_assessments_date DESC)
WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_csa_type_live
ON class_subject_assessments (class_subject_assessments_type)
WHERE deleted_at IS NULL;

-- (opsional) search judul
CREATE INDEX IF NOT EXISTS ix_csa_title_trgm_live
ON class_subject_assessments USING GIN (class_subject_assessments_title gin_trgm_ops)
WHERE deleted_at IS NULL;

DROP TRIGGER IF EXISTS trg_touch_csa ON class_subject_assessments;
CREATE TRIGGER trg_touch_csa
BEFORE UPDATE ON class_subject_assessments
FOR EACH ROW EXECUTE FUNCTION fn_touch_updated_at_generic();

-- =========================================================
-- 3) CLASS SUBJECT ASSESSMENT SCORES (per siswa)
-- =========================================================
CREATE TABLE IF NOT EXISTS class_subject_assessment_scores (
class_subject_assessment_scores_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

class_subject_assessment_scores_school_id UUID NOT NULL REFERENCES schools(school_id) ON DELETE CASCADE,
class_subject_assessment_scores_assessment_id UUID NOT NULL REFERENCES class_subject_assessments(class_subject_assessments_id) ON DELETE CASCADE,
class_subject_assessment_scores_user_class_id UUID NOT NULL REFERENCES user_classes(user_classes_id) ON DELETE CASCADE,

class_subject_assessment_scores_score NUMERIC(6,2) NOT NULL CHECK (class_subject_assessment_scores_score >= 0),
class_subject_assessment_scores_note TEXT,

created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
updated_at TIMESTAMP,
deleted_at TIMESTAMP
);

-- Satu nilai per (assessment, user_class) yang live
CREATE UNIQUE INDEX IF NOT EXISTS uq_csas_assessment_user_class_live
ON class_subject_assessment_scores (class_subject_assessment_scores_assessment_id, class_subject_assessment_scores_user_class_id)
WHERE deleted_at IS NULL;

-- Bantu query
CREATE INDEX IF NOT EXISTS ix_csas_school_assessment_live
ON class_subject_assessment_scores (class_subject_assessment_scores_school_id, class_subject_assessment_scores_assessment_id)
WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_csas_user_class_live
ON class_subject_assessment_scores (class_subject_assessment_scores_user_class_id)
WHERE deleted_at IS NULL;

DROP TRIGGER IF EXISTS trg_touch_csas ON class_subject_assessment_scores;
CREATE TRIGGER trg_touch_csas
BEFORE UPDATE ON class_subject_assessment_scores
FOR EACH ROW EXECUTE FUNCTION fn_touch_updated_at_generic();

-- Validasi tenant: school score harus sama dengan school assessment (DEFERRABLE)
CREATE OR REPLACE FUNCTION fn_csas_validate_tenant()
RETURNS TRIGGER AS $$
DECLARE v_assess_school UUID;
BEGIN
SELECT class_subject_assessments_school_id
INTO v_assess_school
FROM class_subject_assessments
WHERE class_subject_assessments_id = NEW.class_subject_assessment_scores_assessment_id
AND deleted_at IS NULL;

IF v_assess_school IS NULL THEN
RAISE EXCEPTION 'Assessment invalid/terhapus';
END IF;
IF v_assess_school <> NEW.class_subject_assessment_scores_school_id THEN
RAISE EXCEPTION 'School mismatch: score(%) vs assessment(%)',
NEW.class_subject_assessment_scores_school_id, v_assess_school;
END IF;
RETURN NEW;
END$$ LANGUAGE plpgsql;

DO $$
BEGIN
IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_csas_validate_tenant') THEN
DROP TRIGGER trg_csas_validate_tenant ON class_subject_assessment_scores;
END IF;

CREATE CONSTRAINT TRIGGER trg_csas_validate_tenant
AFTER INSERT OR UPDATE OF class_subject_assessment_scores_school_id, class_subject_assessment_scores_assessment_id
ON class_subject_assessment_scores
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
EXECUTE FUNCTION fn_csas_validate_tenant();
END$$;

-- =========================================================
-- 4) USER CLASS SUBJECT FINAL GRADES (nilai akhir per siswa & mapel)
-- =========================================================
CREATE TABLE IF NOT EXISTS user_class_subject_final_grades (
user_class_subject_final_grades_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

user_class_subject_final_grades_school_id UUID NOT NULL REFERENCES schools(school_id) ON DELETE CASCADE,
user_class_subject_final_grades_user_class_id UUID NOT NULL REFERENCES user_classes(user_classes_id) ON DELETE CASCADE,
user_class_subject_final_grades_class_subject_id UUID NOT NULL REFERENCES class_subjects(class_subjects_id) ON DELETE CASCADE,

user_class_subject_final_grades_academic_year TEXT,

-- Semua terskala 0..100
user_class_subject_final_grades_daily_100 NUMERIC(5,2) CHECK (user_class_subject_final_grades_daily_100 IS NULL OR (user_class_subject_final_grades_daily_100 BETWEEN 0 AND 100)),
user_class_subject_final_grades_uts_100 NUMERIC(5,2) CHECK (user_class_subject_final_grades_uts_100 IS NULL OR (user_class_subject_final_grades_uts_100 BETWEEN 0 AND 100)),
user_class_subject_final_grades_uas_100 NUMERIC(5,2) CHECK (user_class_subject_final_grades_uas_100 IS NULL OR (user_class_subject_final_grades_uas_100 BETWEEN 0 AND 100)),
user_class_subject_final_grades_behavior_100 NUMERIC(5,2) CHECK (user_class_subject_final_grades_behavior_100 IS NULL OR (user_class_subject_final_grades_behavior_100 BETWEEN 0 AND 100)),
user_class_subject_final_grades_final_100 NUMERIC(5,2) CHECK (user_class_subject_final_grades_final_100 IS NULL OR (user_class_subject_final_grades_final_100 BETWEEN 0 AND 100)),

user_class_subject_final_grades_letter_grade TEXT,
user_class_subject_final_grades_notes TEXT,

user_class_subject_final_grades_locked_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
user_class_subject_final_grades_locked_at TIMESTAMP,

created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
updated_at TIMESTAMP,
deleted_at TIMESTAMP
);

-- Unik per (tenant, user_class, class_subject, academic_year) yang live
CREATE UNIQUE INDEX IF NOT EXISTS uq_ucsf_tenant_uc_cs_year_live
ON user_class_subject_final_grades (
user_class_subject_final_grades_school_id,
user_class_subject_final_grades_user_class_id,
user_class_subject_final_grades_class_subject_id,
COALESCE(user_class_subject_final_grades_academic_year,'')
)
WHERE deleted_at IS NULL;

-- Bantu query
CREATE INDEX IF NOT EXISTS ix_ucsf_user_class_live
ON user_class_subject_final_grades (user_class_subject_final_grades_user_class_id)
WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_ucsf_class_subject_live
ON user_class_subject_final_grades (user_class_subject_final_grades_class_subject_id)
WHERE deleted_at IS NULL;

DROP TRIGGER IF EXISTS trg_touch_ucsf ON user_class_subject_final_grades;
CREATE TRIGGER trg_touch_ucsf
BEFORE UPDATE ON user_class_subject_final_grades
FOR EACH ROW EXECUTE FUNCTION fn_touch_updated_at_generic();

-- Validasi konsistensi tenant & class match (DEFERRABLE)
CREATE OR REPLACE FUNCTION fn_ucsf_validate_links()
RETURNS TRIGGER AS $$
DECLARE
v_cs_school UUID;
v_cs_class UUID;
v_uc_school UUID;
v_uc_class UUID;
BEGIN
SELECT class_subjects_school_id, class_subjects_class_id
INTO v_cs_school, v_cs_class
FROM class_subjects
WHERE class_subjects_id = NEW.user_class_subject_final_grades_class_subject_id
AND class_subjects_deleted_at IS NULL;

IF v_cs_school IS NULL THEN
RAISE EXCEPTION 'class_subject invalid/terhapus';
END IF;

SELECT user_classes_school_id, user_classes_class_id
INTO v_uc_school, v_uc_class
FROM user_classes
WHERE user_classes_id = NEW.user_class_subject_final_grades_user_class_id;

IF v_uc_school IS NULL THEN
RAISE EXCEPTION 'user_class invalid';
END IF;

-- Tenant harus sama
IF NEW.user_class_subject_final_grades_school_id <> v_cs_school
OR NEW.user_class_subject_final_grades_school_id <> v_uc_school THEN
RAISE EXCEPTION 'School mismatch di final_grades';
END IF;

-- Siswa harus berada di class yang sama dengan class_subject
IF v_uc_class IS NOT NULL AND v_cs_class IS NOT NULL AND v_uc_class <> v_cs_class THEN
RAISE EXCEPTION 'Class mismatch: user_classes.class_id != class_subjects.class_id';
END IF;

RETURN NEW;
END$$ LANGUAGE plpgsql;

DO $$
BEGIN
IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_ucsf_validate_links') THEN
DROP TRIGGER trg_ucsf_validate_links ON user_class_subject_final_grades;
END IF;

CREATE CONSTRAINT TRIGGER trg_ucsf_validate_links
AFTER INSERT OR UPDATE OF
user_class_subject_final_grades_school_id,
user_class_subject_final_grades_user_class_id,
user_class_subject_final_grades_class_subject_id
ON user_class_subject_final_grades
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
EXECUTE FUNCTION fn_ucsf_validate_links();
END$$;

-- =========================================================
-- 5) (Opsional) REPORT CARD SNAPSHOT (membekukan nilai / cetak)
-- =========================================================
CREATE TABLE IF NOT EXISTS report_cards (
report_cards_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
report_cards_school_id UUID NOT NULL REFERENCES schools(school_id) ON DELETE CASCADE,
report_cards_user_class_id UUID NOT NULL REFERENCES user_classes(user_classes_id) ON DELETE CASCADE,

report_cards_academic_year TEXT,
report_cards_term_id UUID NULL REFERENCES academic_terms(academic_terms_id) ON DELETE SET NULL,
report_cards_period_start DATE NOT NULL,
report_cards_period_end DATE NOT NULL,
CHECK (report_cards_period_end >= report_cards_period_start),

-- snapshot agregat
report_cards_gpa_100 NUMERIC(5,2),
report_cards_rank_in_class INT,
report_cards_behavior_100 NUMERIC(5,2),

-- snapshot hadir (opsional)
report_cards_present_count INT,
report_cards_sick_count INT,
report_cards_leave_count INT,
report_cards_absent_count INT,

-- workflow & narasi
report_cards_status TEXT NOT NULL DEFAULT 'draft'
CHECK (report_cards_status IN ('draft','final')),
report_cards_homeroom_comment TEXT,
report_cards_headmaster_comment TEXT,
report_cards_finalized_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
report_cards_finalized_at TIMESTAMP,

created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
updated_at TIMESTAMP,
deleted_at TIMESTAMP
);

-- Unik 1 rapor per scope (live only)
CREATE UNIQUE INDEX IF NOT EXISTS uq_report_cards_scope_live
ON report_cards (
report_cards_school_id,
report_cards_user_class_id,
COALESCE(report_cards_academic_year,''),
COALESCE(report_cards_term_id::text,''),
report_cards_period_start,
report_cards_period_end
) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_report_cards_tenant_userclass_live
ON report_cards (report_cards_school_id, report_cards_user_class_id)
WHERE deleted_at IS NULL;

DROP TRIGGER IF EXISTS trg_touch_report_cards ON report_cards;
CREATE TRIGGER trg_touch_report_cards
BEFORE UPDATE ON report_cards
FOR EACH ROW EXECUTE FUNCTION fn_touch_updated_at_generic();

-- Validasi relasi & tenant (DEFERRABLE)
CREATE OR REPLACE FUNCTION fn_report_cards_validate_links()
RETURNS TRIGGER AS $$
DECLARE
v_uc_school UUID;
v_term_school UUID;
v_term_start DATE;
v_term_end DATE;
BEGIN
-- user_class tenant
SELECT user_classes_school_id
INTO v_uc_school
FROM user_classes
WHERE user_classes_id = NEW.report_cards_user_class_id;

IF v_uc_school IS NULL THEN
RAISE EXCEPTION 'user_class invalid';
END IF;
IF NEW.report_cards_school_id <> v_uc_school THEN
RAISE EXCEPTION 'School mismatch pada report_cards';
END IF;

-- jika term_id diisi: tenant & periode harus berada dalam rentang term
IF NEW.report_cards_term_id IS NOT NULL THEN
SELECT academic_terms_school_id, academic_terms_start_date, academic_terms_end_date
INTO v_term_school, v_term_start, v_term_end
FROM academic_terms
WHERE academic_terms_id = NEW.report_cards_term_id
AND deleted_at IS NULL;

    IF v_term_school IS NULL THEN
      RAISE EXCEPTION 'academic_term invalid/terhapus';
    END IF;
    IF v_term_school <> NEW.report_cards_school_id THEN
      RAISE EXCEPTION 'School mismatch: report_cards vs academic_terms';
    END IF;
    IF NOT (NEW.report_cards_period_start >= v_term_start AND NEW.report_cards_period_end <= v_term_end) THEN
      RAISE EXCEPTION 'Periode rapor tidak berada dalam rentang academic_term';
    END IF;

END IF;

RETURN NEW;
END$$ LANGUAGE plpgsql;

DO $$
BEGIN
IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_report_cards_validate_links') THEN
DROP TRIGGER trg_report_cards_validate_links ON report_cards;
END IF;

CREATE CONSTRAINT TRIGGER trg_report_cards_validate_links
AFTER INSERT OR UPDATE OF
report_cards_school_id, report_cards_user_class_id, report_cards_term_id,
report_cards_period_start, report_cards_period_end
ON report_cards
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
EXECUTE FUNCTION fn_report_cards_validate_links();
END$$;

---

-- Report Card Items (snapshot nilai per mapel)

---

CREATE TABLE IF NOT EXISTS report_card_items (
report_card_items_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
report_card_items_report_card_id UUID NOT NULL REFERENCES report_cards(report_cards_id) ON DELETE CASCADE,
report_card_items_class_subject_id UUID NOT NULL REFERENCES class_subjects(class_subjects_id) ON DELETE RESTRICT,

-- salinan dari user_class_subject_final_grades saat lock
report_card_items_daily_100 NUMERIC(5,2),
report_card_items_uts_100 NUMERIC(5,2),
report_card_items_uas_100 NUMERIC(5,2),
report_card_items_behavior_100 NUMERIC(5,2),
report_card_items_final_100 NUMERIC(5,2),
report_card_items_letter_grade TEXT,
report_card_items_notes TEXT,
report_card_items_teacher_user_id UUID REFERENCES users(id) ON DELETE SET NULL,

created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
updated_at TIMESTAMP,
deleted_at TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_report_card_items_unique_live
ON report_card_items (report_card_items_report_card_id, report_card_items_class_subject_id)
WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_report_card_items_rc_live
ON report_card_items (report_card_items_report_card_id)
WHERE deleted_at IS NULL;

DROP TRIGGER IF EXISTS trg_touch_report_card_items ON report_card_items;
CREATE TRIGGER trg_touch_report_card_items
BEFORE UPDATE ON report_card_items
FOR EACH ROW EXECUTE FUNCTION fn_touch_updated_at_generic();

-- (opsional) validasi tenant & class match secara ketat saat insert/update item
CREATE OR REPLACE FUNCTION fn_report_card_items_validate_links()
RETURNS TRIGGER AS $$
DECLARE
v_rc_school UUID;
v_rc_user_class UUID;
v_cs_class UUID;
v_uc_class UUID;
BEGIN
-- pastikan item mengacu ke rapor yang valid
SELECT report_cards_school_id, report_cards_user_class_id
INTO v_rc_school, v_rc_user_class
FROM report_cards
WHERE report_cards_id = NEW.report_card_items_report_card_id
AND deleted_at IS NULL;

IF v_rc_school IS NULL THEN
RAISE EXCEPTION 'report_card invalid/terhapus';
END IF;

-- kelas dari class_subject
SELECT class_subjects_class_id
INTO v_cs_class
FROM class_subjects
WHERE class_subjects_id = NEW.report_card_items_class_subject_id
AND class_subjects_deleted_at IS NULL;

IF v_cs_class IS NULL THEN
RAISE EXCEPTION 'class_subject invalid/terhapus';
END IF;

-- kelas dari user_class
SELECT user_classes_class_id
INTO v_uc_class
FROM user_classes
WHERE user_classes_id = v_rc_user_class;

IF v_uc_class IS NULL THEN
RAISE EXCEPTION 'user_class invalid';
END IF;

IF v_uc_class <> v_cs_class THEN
RAISE EXCEPTION 'Class mismatch pada report_card_items';
END IF;

RETURN NEW;
END$$ LANGUAGE plpgsql;

DO $$
BEGIN
IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_report_card_items_validate_links') THEN
DROP TRIGGER trg_report_card_items_validate_links ON report_card_items;
END IF;

CREATE CONSTRAINT TRIGGER trg_report_card_items_validate_links
AFTER INSERT OR UPDATE OF
report_card_items_report_card_id, report_card_items_class_subject_id
ON report_card_items
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
EXECUTE FUNCTION fn_report_card_items_validate_links();
END$$;

-- =========================================================
-- Catatan performa:
-- - Semua UNIQUE memakai partial index “WHERE deleted_at IS NULL” agar soft-delete aman.
-- - Index komposit diawali kolom tenant (…\_school_id) supaya selective untuk multi-tenant.
-- - TRGM disiapkan untuk kolom judul/teks yang dicari bebas.
-- - Trigger DEFERRABLE menjaga konsistensi lintas tabel tanpa mengorbankan kecepatan insert bulk.
-- =========================================================
