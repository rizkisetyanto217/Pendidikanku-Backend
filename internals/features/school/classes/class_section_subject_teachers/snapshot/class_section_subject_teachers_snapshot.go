package snapshot

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

/* ======================================================
   Snapshot struct
====================================================== */

// CSSTSnapshot sekarang lebih kaya, tapi tetap kompatibel:
//   - field lama: Name, TeacherID, SectionID (TETAP ADA)
//   - tambah: CSSTID, SchoolID, SubjectID, SectionName, SubjectName,
//     SubjectCode, SubjectSlug, TeacherName, Slug, TitlePrefix/Suffix,
//     ClassSectionSlug, SchoolTeacherSlug
type CSSTSnapshot struct {
	// legacy / dipakai controller
	Name      *string    `json:"name,omitempty"`
	TeacherID *uuid.UUID `json:"teacher_id,omitempty"`
	SectionID *uuid.UUID `json:"section_id,omitempty"`

	// tambahan
	CSSTID    *uuid.UUID `json:"csst_id,omitempty"`
	SchoolID  *uuid.UUID `json:"school_id,omitempty"`
	SubjectID *uuid.UUID `json:"subject_id,omitempty"`

	SectionName *string `json:"section_name,omitempty"`
	SubjectName *string `json:"subject_name,omitempty"`
	SubjectCode *string `json:"subject_code,omitempty"`
	SubjectSlug *string `json:"subject_slug,omitempty"`
	TeacherName *string `json:"teacher_name,omitempty"`
	Slug        *string `json:"slug,omitempty"`

	TeacherTitlePrefix *string `json:"teacher_title_prefix,omitempty"`
	TeacherTitleSuffix *string `json:"teacher_title_suffix,omitempty"`

	// baru: slug nested
	ClassSectionSlug  *string `json:"class_section_slug,omitempty"`
	SchoolTeacherSlug *string `json:"school_teacher_slug,omitempty"`
}

/* ======================================================
   Helpers metadata kolom
====================================================== */

func tableColumns(tx *gorm.DB, table string) (map[string]struct{}, error) {
	type colRow struct {
		ColumnName string `gorm:"column:column_name"`
	}
	var rows []colRow

	q := `
		SELECT column_name
		FROM information_schema.columns
		WHERE table_name = ?
		  AND table_schema = ANY (current_schemas(true))
	`

	if err := tx.Raw(q, table).Scan(&rows).Error; err != nil {
		return nil, err
	}

	out := make(map[string]struct{}, len(rows))
	for _, r := range rows {
		out[strings.ToLower(strings.TrimSpace(r.ColumnName))] = struct{}{}
	}
	return out, nil
}

func firstExisting(cols map[string]struct{}, cands ...string) string {
	for _, c := range cands {
		if _, ok := cols[strings.ToLower(c)]; ok {
			return c
		}
	}
	return ""
}

/* ======================================================
   Main: Validate & snapshot
====================================================== */

// ValidateAndSnapshotCSST
// - tetap dinamis via information_schema
// - tapi SELECT lebih banyak kolom, supaya snapshot bisa lebih kaya
func ValidateAndSnapshotCSST(
	tx *gorm.DB,
	expectSchoolID uuid.UUID,
	csstID uuid.UUID,
) (*CSSTSnapshot, error) {
	if csstID == uuid.Nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "csst_id tidak valid")
	}

	cols, err := tableColumns(tx, "class_section_subject_teachers")
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal membaca metadata skema CSST")
	}

	// =========================
	// Mapping kolom dinamis
	// =========================

	idCol := firstExisting(cols,
		"class_section_subject_teacher_id",
		"id",
	)

	schoolCol := firstExisting(cols,
		"class_section_subject_teacher_school_id",
		"school_id",
	)

	// Nama label CSST (bisa beda dari nama section)
	csstNameCol := firstExisting(cols,
		"class_section_subject_teacher_name",
		"name",
	)

	// Nama section (kelas) snapshot
	sectionNameCol := firstExisting(cols,
		"class_section_subject_teacher_class_section_name_snapshot",
		"class_section_name",
		"section_name",
	)

	// Slug section snapshot (kalau ada)
	classSectionSlugCol := firstExisting(cols,
		"class_section_subject_teacher_class_section_slug_snapshot",
		"class_section_slug",
		"section_slug",
	)

	// Teacher id
	teacherCol := firstExisting(cols,
		"class_section_subject_teacher_school_teacher_id", // skema baru
		"class_section_subject_teacher_teacher_id",        // kemungkinan lama
		"school_teacher_id",
		"teacher_id",
	)

	// Section id
	sectionCol := firstExisting(cols,
		"class_section_subject_teacher_class_section_id", // skema baru
		"class_section_subject_teacher_section_id",       // kemungkinan lama
		"class_section_id",
		"section_id",
	)

	// Subject id
	subjectCol := firstExisting(cols,
		"class_section_subject_teacher_subject_id",
		"subject_id",
	)

	// Subject name snapshot
	subjectNameCol := firstExisting(cols,
		"class_section_subject_teacher_subject_name_snapshot",
		"subject_name",
	)

	// Subject code snapshot
	subjectCodeCol := firstExisting(cols,
		"class_section_subject_teacher_subject_code_snapshot",
		"subject_code",
	)

	// Subject slug snapshot
	subjectSlugCol := firstExisting(cols,
		"class_section_subject_teacher_subject_slug_snapshot",
		"subject_slug",
	)

	// Teacher name snapshot (di CSST, kalau ada)
	teacherNameCol := firstExisting(cols,
		"class_section_subject_teacher_teacher_name_snapshot",
		"school_teacher_name",
		"teacher_name",
	)

	// Teacher title prefix/suffix snapshot (di CSST, kalau ada)
	teacherTitlePrefixCol := firstExisting(cols,
		"class_section_subject_teacher_teacher_title_prefix_snapshot",
		"teacher_title_prefix",
		"school_teacher_title_prefix",
	)
	teacherTitleSuffixCol := firstExisting(cols,
		"class_section_subject_teacher_teacher_title_suffix_snapshot",
		"teacher_title_suffix",
		"school_teacher_title_suffix",
	)

	// Teacher slug snapshot (kalau ada di CSST)
	schoolTeacherSlugCol := firstExisting(cols,
		"class_section_subject_teacher_school_teacher_slug_snapshot",
		"school_teacher_slug",
		"teacher_slug",
	)

	// Slug CSST
	slugCol := firstExisting(cols,
		"class_section_subject_teacher_slug",
		"slug",
	)

	deletedCol := firstExisting(cols,
		"class_section_subject_teacher_deleted_at",
		"deleted_at",
	)

	if idCol == "" || schoolCol == "" {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "CSST snapshot: kolom minimal (id/school_id) tidak ditemukan")
	}

	// ========== build expr dinamis per kolom ==========

	toTextExpr := func(col string) string {
		if col == "" {
			return "NULL::text"
		}
		return fmt.Sprintf("csst.%s", col)
	}

	toUUIDExpr := func(col string) string {
		if col == "" {
			return "NULL::uuid"
		}
		return fmt.Sprintf("csst.%s", col)
	}

	csstIDExpr := toUUIDExpr(idCol)
	schoolIDExpr := fmt.Sprintf("csst.%s::text", schoolCol) // supaya aman di-parse
	csstNameExpr := toTextExpr(csstNameCol)
	sectionNameExpr := toTextExpr(sectionNameCol)
	classSectionSlugExpr := toTextExpr(classSectionSlugCol)
	teacherIDExpr := toUUIDExpr(teacherCol)
	sectionIDExpr := toUUIDExpr(sectionCol)
	subjectIDExpr := toUUIDExpr(subjectCol)
	subjectNameExpr := toTextExpr(subjectNameCol)
	subjectCodeExpr := toTextExpr(subjectCodeCol)
	subjectSlugExpr := toTextExpr(subjectSlugCol)
	teacherNameExpr := toTextExpr(teacherNameCol)
	slugExpr := toTextExpr(slugCol)
	teacherTitlePrefixExpr := toTextExpr(teacherTitlePrefixCol)
	teacherTitleSuffixExpr := toTextExpr(teacherTitleSuffixCol)
	schoolTeacherSlugExpr := toTextExpr(schoolTeacherSlugCol)

	whereDeleted := ""
	if deletedCol != "" {
		whereDeleted = fmt.Sprintf(" AND csst.%s IS NULL", deletedCol)
	}

	q := fmt.Sprintf(`
		SELECT
			%s       AS csst_id,
			%s       AS school_id,
			%s       AS csst_name,
			%s       AS section_name,
			%s       AS class_section_slug,
			%s       AS teacher_id,
			%s       AS section_id,
			%s       AS subject_id,
			%s       AS subject_name,
			%s       AS subject_code,
			%s       AS subject_slug,
			%s       AS teacher_name,
			%s       AS slug,
			%s       AS teacher_title_prefix,
			%s       AS teacher_title_suffix,
			%s       AS school_teacher_slug
		FROM class_section_subject_teachers csst
		WHERE csst.%s = ? %s
		LIMIT 1
	`,
		csstIDExpr,
		schoolIDExpr,
		csstNameExpr,
		sectionNameExpr,
		classSectionSlugExpr,
		teacherIDExpr,
		sectionIDExpr,
		subjectIDExpr,
		subjectNameExpr,
		subjectCodeExpr,
		subjectSlugExpr,
		teacherNameExpr,
		slugExpr,
		teacherTitlePrefixExpr,
		teacherTitleSuffixExpr,
		schoolTeacherSlugExpr,
		idCol,
		whereDeleted,
	)

	var row struct {
		CSSTID            *uuid.UUID `gorm:"column:csst_id"`
		SchoolID          string     `gorm:"column:school_id"`
		CSSTName          *string    `gorm:"column:csst_name"`
		SectionName       *string    `gorm:"column:section_name"`
		ClassSectionSlug  *string    `gorm:"column:class_section_slug"`
		TeacherID         *uuid.UUID `gorm:"column:teacher_id"`
		SectionID         *uuid.UUID `gorm:"column:section_id"`
		SubjectID         *uuid.UUID `gorm:"column:subject_id"`
		SubjectName       *string    `gorm:"column:subject_name"`
		SubjectCode       *string    `gorm:"column:subject_code"`
		SubjectSlug       *string    `gorm:"column:subject_slug"`
		TeacherName       *string    `gorm:"column:teacher_name"`
		Slug              *string    `gorm:"column:slug"`
		TeacherTitlePref  *string    `gorm:"column:teacher_title_prefix"`
		TeacherTitleSuf   *string    `gorm:"column:teacher_title_suffix"`
		SchoolTeacherSlug *string    `gorm:"column:school_teacher_slug"`
	}

	if err := tx.Raw(q, csstID).Scan(&row).Error; err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal memuat CSST")
	}
	if strings.TrimSpace(row.SchoolID) == "" {
		return nil, fiber.NewError(fiber.StatusBadRequest, "CSST tidak ditemukan")
	}

	rmz, perr := uuid.Parse(strings.TrimSpace(row.SchoolID))
	if perr != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Format school_id CSST tidak valid")
	}
	if expectSchoolID != uuid.Nil && rmz != uuid.Nil && rmz != expectSchoolID {
		return nil, fiber.NewError(fiber.StatusForbidden, "CSST bukan milik school Anda")
	}

	trimPtr := func(p *string) *string {
		if p == nil {
			return nil
		}
		v := strings.TrimSpace(*p)
		if v == "" {
			return nil
		}
		return &v
	}

	schoolUUID := rmz

	snap := &CSSTSnapshot{
		Name:      trimPtr(row.CSSTName),
		TeacherID: row.TeacherID,
		SectionID: row.SectionID,

		CSSTID:    row.CSSTID,
		SchoolID:  &schoolUUID,
		SubjectID: row.SubjectID,

		SectionName:        trimPtr(row.SectionName),
		SubjectName:        trimPtr(row.SubjectName),
		SubjectCode:        trimPtr(row.SubjectCode),
		SubjectSlug:        trimPtr(row.SubjectSlug),
		TeacherName:        trimPtr(row.TeacherName),
		Slug:               trimPtr(row.Slug),
		TeacherTitlePrefix: trimPtr(row.TeacherTitlePref),
		TeacherTitleSuffix: trimPtr(row.TeacherTitleSuf),

		ClassSectionSlug:  trimPtr(row.ClassSectionSlug),
		SchoolTeacherSlug: trimPtr(row.SchoolTeacherSlug),
	}

	// 🔍 Enrich dari school_teachers kalau name/prefix/suffix/slug belum ada,
	// tapi TeacherID ada (kayak kasus "Hendra / Ustadz / Lc").
	if err := enrichTeacherSnapshotFromSchoolTeacher(tx, expectSchoolID, snap); err != nil {
		return nil, err
	}

	return snap, nil
}

/* ======================================================
   Enrich teacher snapshot from school_teachers
====================================================== */

func enrichTeacherSnapshotFromSchoolTeacher(
	tx *gorm.DB,
	expectSchoolID uuid.UUID,
	cs *CSSTSnapshot,
) error {
	if cs == nil || cs.TeacherID == nil || *cs.TeacherID == uuid.Nil {
		return nil
	}

	// Cek apa sudah lengkap semuanya
	hasName := cs.TeacherName != nil && strings.TrimSpace(*cs.TeacherName) != ""
	hasPref := cs.TeacherTitlePrefix != nil && strings.TrimSpace(*cs.TeacherTitlePrefix) != ""
	hasSuf := cs.TeacherTitleSuffix != nil && strings.TrimSpace(*cs.TeacherTitleSuffix) != ""
	hasSlug := cs.SchoolTeacherSlug != nil && strings.TrimSpace(*cs.SchoolTeacherSlug) != ""

	// Kalau semua sudah ada, nggak perlu query lagi
	if hasName && hasPref && hasSuf && hasSlug {
		return nil
	}

	type row struct {
		SchoolID string  `gorm:"column:school_id"`
		Name     *string `gorm:"column:name"`
		Prefix   *string `gorm:"column:title_prefix"`
		Suffix   *string `gorm:"column:title_suffix"`
		Slug     *string `gorm:"column:slug"`
	}

	// Ambil dari SNAPSHOT di tabel school_teachers
	q := `
		SELECT
			school_teacher_school_id::text                      AS school_id,
			school_teacher_user_teacher_name_snapshot           AS name,
			school_teacher_user_teacher_title_prefix_snapshot   AS title_prefix,
			school_teacher_user_teacher_title_suffix_snapshot   AS title_suffix,
			school_teacher_slug                                 AS slug
		FROM school_teachers
		WHERE school_teacher_id = ?
		  AND school_teacher_deleted_at IS NULL
		LIMIT 1
	`

	var r row
	if err := tx.Raw(q, *cs.TeacherID).Scan(&r).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal memuat guru untuk snapshot CSST")
	}
	if strings.TrimSpace(r.SchoolID) == "" {
		// guru tidak ditemukan → nggak fatal, skip saja
		return nil
	}

	rmz, perr := uuid.Parse(strings.TrimSpace(r.SchoolID))
	if perr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Format school_id guru tidak valid")
	}
	if expectSchoolID != uuid.Nil && rmz != uuid.Nil && rmz != expectSchoolID {
		return fiber.NewError(fiber.StatusForbidden, "Guru bukan milik school Anda")
	}

	trimPtr := func(p *string) *string {
		if p == nil {
			return nil
		}
		v := strings.TrimSpace(*p)
		if v == "" {
			return nil
		}
		return &v
	}

	// Isi hanya kalau masih kosong
	if !hasName {
		cs.TeacherName = trimPtr(r.Name)
	}
	if !hasPref {
		cs.TeacherTitlePrefix = trimPtr(r.Prefix)
	}
	if !hasSuf {
		cs.TeacherTitleSuffix = trimPtr(r.Suffix)
	}
	if !hasSlug {
		cs.SchoolTeacherSlug = trimPtr(r.Slug)
	}

	return nil
}

/*
======================================================

	ToJSON: bentuk final JSON snapshot

======================================================
*/
func ToJSON(cs *CSSTSnapshot) datatypes.JSON {
	if cs == nil {
		return datatypes.JSON([]byte("null"))
	}

	m := map[string]any{
		"captured_at": time.Now().UTC(),
		"source":      "generator_v2",
	}

	// identitas utama
	if cs.CSSTID != nil {
		m["csst_id"] = *cs.CSSTID
	}
	// ❌ school_id sengaja tidak dimasukkan ke JSON,
	// karena sudah ada di tabel assessment
	if cs.SectionID != nil {
		m["section_id"] = *cs.SectionID
	}
	if cs.SubjectID != nil {
		m["subject_id"] = *cs.SubjectID
	}
	if cs.TeacherID != nil {
		m["teacher_id"] = *cs.TeacherID
	}

	// siapkan trimmed values untuk fallback
	var (
		nameVal             string
		sectionNameVal      string
		classSectionSlugVal string
		subjectNameVal      string
		subjectCodeVal      string
		subjectSlugVal      string
		teacherNameVal      string
		teacherSlugVal      string
		slugVal             string
		titlePrefixVal      string
		titleSuffixVal      string
	)

	if cs.Name != nil {
		nameVal = strings.TrimSpace(*cs.Name)
	}
	if cs.SectionName != nil {
		sectionNameVal = strings.TrimSpace(*cs.SectionName)
	}
	if cs.ClassSectionSlug != nil {
		classSectionSlugVal = strings.TrimSpace(*cs.ClassSectionSlug)
	}
	if cs.SubjectName != nil {
		subjectNameVal = strings.TrimSpace(*cs.SubjectName)
	}
	if cs.SubjectCode != nil {
		subjectCodeVal = strings.TrimSpace(*cs.SubjectCode)
	}
	if cs.SubjectSlug != nil {
		subjectSlugVal = strings.TrimSpace(*cs.SubjectSlug)
	}
	if cs.TeacherName != nil {
		teacherNameVal = strings.TrimSpace(*cs.TeacherName)
	}
	if cs.SchoolTeacherSlug != nil {
		teacherSlugVal = strings.TrimSpace(*cs.SchoolTeacherSlug)
	}
	if cs.Slug != nil {
		slugVal = strings.TrimSpace(*cs.Slug)
	}
	if cs.TeacherTitlePrefix != nil {
		titlePrefixVal = strings.TrimSpace(*cs.TeacherTitlePrefix)
	}
	if cs.TeacherTitleSuffix != nil {
		titleSuffixVal = strings.TrimSpace(*cs.TeacherTitleSuffix)
	}

	// === label name dengan fallback ===
	label := ""
	if nameVal != "" {
		label = nameVal
	} else if sectionNameVal != "" && subjectNameVal != "" {
		// ex: "Kelas Balaghoh B — Sejarah Ilmu Balaghah"
		label = sectionNameVal + " — " + subjectNameVal
	} else if subjectNameVal != "" {
		label = subjectNameVal
	} else if sectionNameVal != "" {
		label = sectionNameVal
	}
	if label != "" {
		m["name"] = label
	}

	// info ringkas lain di root:
	// - subject_* TIDAK lagi ditaruh di root (digroup di "subject")
	// - school_id juga tidak ditaruh di root
	if teacherNameVal != "" {
		m["teacher_name"] = teacherNameVal
	}
	if slugVal != "" {
		m["slug"] = slugVal
	}

	// nested class_section
	if cs.SectionID != nil && sectionNameVal != "" {
		csMap := map[string]any{
			"id":   *cs.SectionID,
			"name": sectionNameVal,
		}
		if classSectionSlugVal != "" {
			csMap["slug"] = classSectionSlugVal
		}
		m["class_section"] = csMap
	}

	// nested subject (SEMUA field subject_* di sini)
	if cs.SubjectID != nil || subjectNameVal != "" || subjectCodeVal != "" || subjectSlugVal != "" {
		sub := map[string]any{}

		if cs.SubjectID != nil {
			sub["id"] = *cs.SubjectID
		}
		if subjectCodeVal != "" {
			sub["code"] = subjectCodeVal
		}
		if subjectNameVal != "" {
			sub["name"] = subjectNameVal
		}
		if subjectSlugVal != "" {
			sub["slug"] = subjectSlugVal
		}

		m["subject"] = sub
	}

	// nested school_teacher
	if cs.TeacherID != nil || teacherNameVal != "" || titlePrefixVal != "" || titleSuffixVal != "" || teacherSlugVal != "" {
		st := map[string]any{}
		if cs.TeacherID != nil {
			st["id"] = *cs.TeacherID
		}
		if teacherNameVal != "" {
			st["name"] = teacherNameVal
		}
		if teacherSlugVal != "" {
			st["slug"] = teacherSlugVal
		}
		if titlePrefixVal != "" {
			st["title_prefix"] = titlePrefixVal
		}
		if titleSuffixVal != "" {
			st["title_suffix"] = titleSuffixVal
		}
		m["school_teacher"] = st
	}

	b, _ := json.Marshal(m)
	return datatypes.JSON(b)
}
