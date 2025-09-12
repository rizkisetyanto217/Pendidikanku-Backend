package controller

import (
	"errors"
	"fmt"
	modelCSST "masjidku_backend/internals/features/school/subject_books/subject/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// -------------------------------------------------------------
// HELPER: deteksi nama kolom FK dinamis di tabel CSST
// -------------------------------------------------------------
var (
	csstFKOnce sync.Once

	// kandidat nama kolom yang mungkin dipakai di CSST
	csstClassSubjectFK string // ex: class_section_subject_teachers_class_subjects_id / class_subjects_id / dst
	csstSubjectFK      string // ex: class_section_subject_teachers_subject_id / subjects_id
	csstSectionFK      string // ex: class_section_subject_teachers_section_id / section_id
)

func hasColumn(db *gorm.DB, table, col string) bool {
	var n int64
	_ = db.Raw(`
        SELECT COUNT(*)
        FROM pg_attribute
        WHERE attrelid = to_regclass(?) AND attname = ? AND NOT attisdropped
    `, "public."+table, col).Scan(&n).Error
	return n > 0
}

func detectCSSTFKs(db *gorm.DB) {
	csstFKOnce.Do(func() {
		for _, c := range []string{
			"class_section_subject_teachers_class_subjects_id",
			"class_section_subject_teachers_class_subject_id",
			"class_subjects_id",
		} {
			if hasColumn(db, "class_section_subject_teachers", c) {
				csstClassSubjectFK = c
				break
			}
		}
		for _, c := range []string{
			"class_section_subject_teachers_subject_id",
			"subjects_id",
		} {
			if hasColumn(db, "class_section_subject_teachers", c) {
				csstSubjectFK = c
				break
			}
		}
		for _, c := range []string{
			"class_section_subject_teachers_section_id",
			"section_id",
		} {
			if hasColumn(db, "class_section_subject_teachers", c) {
				csstSectionFK = c
				break
			}
		}
	})
}

// -------------------------------------------------------------
// LITE structs untuk join
// -------------------------------------------------------------
type subjectLite struct {
	SubjectsID   *uuid.UUID `json:"subjects_id,omitempty"   gorm:"column:subjects_id"`
	SubjectsCode *string    `json:"subjects_code,omitempty" gorm:"column:subjects_code"`
	SubjectsName *string    `json:"subjects_name,omitempty" gorm:"column:subjects_name"`
	SubjectsSlug *string    `json:"subjects_slug,omitempty" gorm:"column:subjects_slug"`
}

type teacherLite struct {
	TeacherID         *uuid.UUID `json:"teacher_id,omitempty"         gorm:"column:teacher_id_join"`
	TeacherUserID     *uuid.UUID `json:"teacher_user_id,omitempty"     gorm:"column:teacher_user_id"`
	TeacherTitle      *string    `json:"teacher_title,omitempty"       gorm:"column:teacher_title"`
	TeacherCode       *string    `json:"teacher_code,omitempty"        gorm:"column:teacher_code"`
	TeacherEmployment *string    `json:"teacher_employment,omitempty"  gorm:"column:teacher_employment"`
	TeacherIsActive   *bool      `json:"teacher_is_active,omitempty"   gorm:"column:teacher_is_active"`
	TeacherName       *string    `json:"teacher_name,omitempty"        gorm:"column:teacher_name"`
}


type sectionLite struct {
	ClassSectionsID   *uuid.UUID `json:"class_sections_id,omitempty"   gorm:"column:class_sections_id"`
	ClassSectionsName *string    `json:"class_sections_name,omitempty" gorm:"column:class_sections_name"`
	ClassSectionsCode *string    `json:"class_sections_code,omitempty" gorm:"column:class_sections_code"`
}

type csstJoinedRow struct {
	modelCSST.ClassSectionSubjectTeacherModel

	// class_subjects
	ClassSubjectsID        *uuid.UUID `json:"class_subjects_id,omitempty"         gorm:"column:class_subjects_id"`
	ClassSubjectsSubjectID *uuid.UUID `json:"class_subjects_subject_id,omitempty" gorm:"column:class_subjects_subject_id"`

	// subjects
	SubjectsID   *uuid.UUID `gorm:"column:subjects_id"`
	SubjectsCode *string    `gorm:"column:subjects_code"`
	SubjectsName *string    `gorm:"column:subjects_name"`
	SubjectsSlug *string    `gorm:"column:subjects_slug"`

	// sections
	ClassSectionsID   *uuid.UUID `gorm:"column:class_sections_id"`
	ClassSectionsName *string    `gorm:"column:class_sections_name"`
	ClassSectionsCode *string    `gorm:"column:class_sections_code"`

	// teachers (alias harus match SELECT di query)
	TeacherIDJoin     *uuid.UUID `gorm:"column:teacher_id_join"`
	TeacherUserID     *uuid.UUID `gorm:"column:teacher_user_id"`
	TeacherTitle      *string    `gorm:"column:teacher_title"`
	TeacherCode       *string    `gorm:"column:teacher_code"`
	TeacherEmployment *string    `gorm:"column:teacher_employment"`
	TeacherIsActive   *bool      `gorm:"column:teacher_is_active"`
	TeacherName       *string    `gorm:"column:teacher_name"`
}

type csstItemWithRefs struct {
	CSST         modelCSST.ClassSectionSubjectTeacherModel `json:"csst"`
	ClassSubject *struct {
		ClassSubjectsID        *uuid.UUID `json:"class_subjects_id,omitempty"`
		ClassSubjectsSubjectID *uuid.UUID `json:"class_subjects_subject_id,omitempty"`
	} `json:"class_subject,omitempty"`
	Subject *subjectLite `json:"subject,omitempty"`
	Section *sectionLite `json:"section,omitempty"`
	Teacher *teacherLite `json:"teacher,omitempty"`
}


func toCSSTResp(r csstJoinedRow) csstItemWithRefs {
	out := csstItemWithRefs{CSST: r.ClassSectionSubjectTeacherModel}

	if r.ClassSubjectsID != nil || r.ClassSubjectsSubjectID != nil {
		out.ClassSubject = &struct {
			ClassSubjectsID        *uuid.UUID `json:"class_subjects_id,omitempty"`
			ClassSubjectsSubjectID *uuid.UUID `json:"class_subjects_subject_id,omitempty"`
		}{
			ClassSubjectsID:        r.ClassSubjectsID,
			ClassSubjectsSubjectID: r.ClassSubjectsSubjectID,
		}
	}
	if r.SubjectsID != nil || r.SubjectsName != nil || r.SubjectsCode != nil || r.SubjectsSlug != nil {
		out.Subject = &subjectLite{
			SubjectsID:   r.SubjectsID,
			SubjectsCode: r.SubjectsCode,
			SubjectsName: r.SubjectsName,
			SubjectsSlug: r.SubjectsSlug,
		}
	}
	if r.ClassSectionsID != nil || r.ClassSectionsName != nil || r.ClassSectionsCode != nil {
		out.Section = &sectionLite{
			ClassSectionsID:   r.ClassSectionsID,
			ClassSectionsName: r.ClassSectionsName,
			ClassSectionsCode: r.ClassSectionsCode,
		}
	}
	// ← ini yang sebelumnya tidak ada
	if r.TeacherIDJoin != nil || r.TeacherUserID != nil || r.TeacherName != nil || r.TeacherTitle != nil || r.TeacherCode != nil || r.TeacherEmployment != nil || r.TeacherIsActive != nil {
		out.Teacher = &teacherLite{
			TeacherID:         r.TeacherIDJoin,
			TeacherUserID:     r.TeacherUserID,
			TeacherTitle:      r.TeacherTitle,
			TeacherCode:       r.TeacherCode,
			TeacherEmployment: r.TeacherEmployment,
			TeacherIsActive:   r.TeacherIsActive,
			TeacherName:       r.TeacherName,
		}
	}

	return out
}

// -------------------------------------------------------------
// Query params
// -------------------------------------------------------------
type listQuery struct {
	IsActive    *bool   `query:"is_active"`
	WithDeleted *bool   `query:"with_deleted"`
	Limit       *int    `query:"limit"`
	Offset      *int    `query:"offset"`
	OrderBy     *string `query:"order_by"` // created_at|updated_at|subject_name|subject_code|section_name|section_code
	Sort        *string `query:"sort"`     // asc|desc
}

// helper: parse include=subject,section,class_subject,teacher,all
func parseInclude(raw string) map[string]bool {
	m := map[string]bool{}
	if raw == "" {
		return m
	}
	parts := strings.Split(strings.ToLower(strings.TrimSpace(raw)), ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		m[p] = true
	}
	// alias normalization
	if m["subjects"] || m["s"] { m["subject"] = true }
	if m["sections"] || m["sec"] { m["section"] = true }
	if m["cs"] { m["class_subject"] = true }
	if m["t"] { m["teacher"] = true }
	if m["all"] {
		m["subject"], m["section"], m["class_subject"], m["teacher"] = true, true, true, true
	}
	return m
}

// -------------------------------------------------------------
// LIST handler
// GET /api/{a|u}/class-section-subject-teachers/list
// Optional :id => detail by id
// Filter qparams: section_id, class_subject_id, subject_id, teacher_id, masjid_id, q
// Sorting: created_at|updated_at|subject_name|subject_code|section_name|section_code (+ sort asc|desc)
// -------------------------------------------------------------
func (ctl *ClassSectionSubjectTeacherController) List(c *fiber.Ctx) error {
	detectCSSTFKs(ctl.DB)

	masjidIDs, err := helperAuth.GetMasjidIDsFromToken(c)
	if err != nil {
		return err
	}

	// path :id (detail)
	var pathID *uuid.UUID
	if s := strings.TrimSpace(c.Params("id")); s != "" {
		id, e := uuid.Parse(s)
		if e != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "id tidak valid")
		}
		pathID = &id
	}

	// qparams umum
	var q listQuery
	q.Limit, q.Offset = intPtr(20), intPtr(0)
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}
	if q.Limit == nil || *q.Limit <= 0 || *q.Limit > 200 {
		q.Limit = intPtr(20)
	}
	if q.Offset == nil || *q.Offset < 0 {
		q.Offset = intPtr(0)
	}

	// qparams tambahan
	sectionID := strings.TrimSpace(c.Query("section_id"))
	classSubID := strings.TrimSpace(c.Query("class_subject_id"))
	subjectID := strings.TrimSpace(c.Query("subject_id"))
	teacherID := strings.TrimSpace(c.Query("teacher_id"))
	masjidIDOne := strings.TrimSpace(c.Query("masjid_id"))
	qtext := strings.TrimSpace(strings.ToLower(c.Query("q")))
	includes := parseInclude(c.Query("include"))

	// sorting (guard bila butuh relasi)
	orderBy := "csst.class_section_subject_teachers_created_at"
	if q.OrderBy != nil {
		switch strings.ToLower(*q.OrderBy) {
		case "created_at":
			orderBy = "csst.class_section_subject_teachers_created_at"
		case "updated_at":
			orderBy = "csst.class_section_subject_teachers_updated_at"
		case "subject_name":
			if !includes["subject"] {
				return helper.JsonError(c, fiber.StatusBadRequest, "order_by=subject_name memerlukan include=subject")
			}
			orderBy = "COALESCE(s.subjects_name,'')"
		case "subject_code":
			if !includes["subject"] {
				return helper.JsonError(c, fiber.StatusBadRequest, "order_by=subject_code memerlukan include=subject")
			}
			orderBy = "COALESCE(s.subjects_code,'')"
		case "section_name":
			if !includes["section"] {
				return helper.JsonError(c, fiber.StatusBadRequest, "order_by=section_name memerlukan include=section")
			}
			orderBy = "COALESCE(sec.class_sections_name,'')"
		case "section_code":
			if !includes["section"] {
				return helper.JsonError(c, fiber.StatusBadRequest, "order_by=section_code memerlukan include=section")
			}
			orderBy = "COALESCE(sec.class_sections_code,'')"
		default:
			return helper.JsonError(c, fiber.StatusBadRequest, "order_by tidak dikenal")
		}
	}
	sort := "ASC"
	if q.Sort != nil && strings.ToLower(*q.Sort) == "desc" {
		sort = "DESC"
	}

	// ================= BASE QUERY (DATA) =================
	tx := ctl.DB.
		Table("class_section_subject_teachers AS csst").
		Where("csst.class_section_subject_teachers_masjid_id IN ?", masjidIDs)

	if q.WithDeleted == nil || !*q.WithDeleted {
		tx = tx.Where("csst.class_section_subject_teachers_deleted_at IS NULL")
	}
	if q.IsActive != nil {
		tx = tx.Where("csst.class_section_subject_teachers_is_active = ?", *q.IsActive)
	}

	// SELECT kolom (polos)
	selectCols := []string{"csst.*"}

	// ===== JOIN kondisional sesuai include =====
	// class_subjects — hindari double join saat include subject juga aktif
	needJoinCS := (includes["class_subject"] || (includes["subject"] && csstClassSubjectFK != "")) && csstClassSubjectFK != ""
	if needJoinCS {
		tx = tx.Joins(fmt.Sprintf(`
			LEFT JOIN class_subjects AS cs
			  ON cs.class_subjects_id = csst.%s
		`, csstClassSubjectFK))
		if includes["class_subject"] {
			selectCols = append(selectCols,
				"cs.class_subjects_id AS class_subjects_id",
				"cs.class_subjects_subject_id AS class_subjects_subject_id",
			)
		}
	}

	// subjects
	if includes["subject"] {
		if csstClassSubjectFK != "" {
			tx = tx.Joins(`LEFT JOIN subjects AS s ON s.subjects_id = cs.class_subjects_subject_id`)
		} else if csstSubjectFK != "" {
			tx = tx.Joins(fmt.Sprintf(`LEFT JOIN subjects AS s ON s.subjects_id = csst.%s`, csstSubjectFK))
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "include=subject tidak tersedia (FK subjects tidak terdeteksi)")
		}
		selectCols = append(selectCols,
			"s.subjects_id AS subjects_id",
			"s.subjects_code AS subjects_code",
			"s.subjects_name AS subjects_name",
			"s.subjects_slug AS subjects_slug",
		)
	}

	// sections
	if includes["section"] {
		if csstSectionFK == "" {
			return helper.JsonError(c, fiber.StatusBadRequest, "include=section tidak tersedia (FK class_sections tidak terdeteksi)")
		}
		tx = tx.Joins(fmt.Sprintf(`LEFT JOIN class_sections AS sec ON sec.class_sections_id = csst.%s`, csstSectionFK))
		selectCols = append(selectCols,
			"sec.class_sections_id   AS class_sections_id",
			"sec.class_sections_name AS class_sections_name",
			"sec.class_sections_code AS class_sections_code",
		)
	}

	// teacher → join ke masjid_teachers + users (untuk nama)
	if includes["teacher"] {
		tx = tx.
			Joins(`LEFT JOIN masjid_teachers AS mt ON mt.masjid_teacher_id = csst.class_section_subject_teachers_teacher_id`).
			Joins(`LEFT JOIN users AS u ON u.id = mt.masjid_teacher_user_id`)
		selectCols = append(selectCols,
			"mt.masjid_teacher_id          AS teacher_id_join",
			"mt.masjid_teacher_user_id     AS teacher_user_id",
			"mt.masjid_teacher_title       AS teacher_title",
			"mt.masjid_teacher_code        AS teacher_code",
			"mt.masjid_teacher_employment  AS teacher_employment",
			"mt.masjid_teacher_is_active   AS teacher_is_active",
			"COALESCE(u.full_name, u.user_name) AS teacher_name",
		)
	}

	// ============= FILTERS (di DATA tx) =============
	if teacherID != "" {
		if _, e := uuid.Parse(teacherID); e == nil {
			tx = tx.Where("csst.class_section_subject_teachers_teacher_id = ?", teacherID)
		}
	}
	if masjidIDOne != "" {
		if _, e := uuid.Parse(masjidIDOne); e == nil {
			tx = tx.Where("csst.class_section_subject_teachers_masjid_id = ?", masjidIDOne)
		}
	}
	if sectionID != "" {
		if csstSectionFK == "" {
			return helper.JsonError(c, fiber.StatusBadRequest, "filter section_id memerlukan FK section; aktifkan include=section")
		}
		if !includes["section"] {
			return helper.JsonError(c, fiber.StatusBadRequest, "filter section_id memerlukan include=section")
		}
		if _, e := uuid.Parse(sectionID); e == nil {
			tx = tx.Where(fmt.Sprintf("csst.%s = ?", csstSectionFK), sectionID)
		}
	}
	if classSubID != "" {
		if csstClassSubjectFK == "" {
			return helper.JsonError(c, fiber.StatusBadRequest, "filter class_subject_id memerlukan FK class_subject; aktifkan include=class_subject")
		}
		if !includes["class_subject"] {
			return helper.JsonError(c, fiber.StatusBadRequest, "filter class_subject_id memerlukan include=class_subject")
		}
		if _, e := uuid.Parse(classSubID); e == nil {
			tx = tx.Where(fmt.Sprintf("csst.%s = ?", csstClassSubjectFK), classSubID)
		}
	}
	if subjectID != "" {
		if !includes["subject"] {
			return helper.JsonError(c, fiber.StatusBadRequest, "filter subject_id memerlukan include=subject")
		}
		if _, e := uuid.Parse(subjectID); e == nil {
			tx = tx.Where("s.subjects_id = ?", subjectID)
		}
	}
	if qtext != "" {
		switch {
		case includes["subject"] && includes["section"]:
			tx = tx.Where("(LOWER(s.subjects_name) LIKE ? OR LOWER(s.subjects_code) LIKE ? OR LOWER(sec.class_sections_name) LIKE ? OR LOWER(sec.class_sections_code) LIKE ?)",
				"%"+qtext+"%", "%"+qtext+"%", "%"+qtext+"%", "%"+qtext+"%")
		case includes["subject"]:
			tx = tx.Where("(LOWER(s.subjects_name) LIKE ? OR LOWER(s.subjects_code) LIKE ?)",
				"%"+qtext+"%", "%"+qtext+"%")
		case includes["section"]:
			tx = tx.Where("(LOWER(sec.class_sections_name) LIKE ? OR LOWER(sec.class_sections_code) LIKE ?)",
				"%"+qtext+"%", "%"+qtext+"%")
		// polos: q diabaikan
		}
	}

	// ===== DETAIL BY ID =====
	if pathID != nil {
		var row csstJoinedRow
		if err := tx.
			Select(strings.Join(selectCols, ", ")).
			Where("csst.class_section_subject_teachers_id = ?", *pathID).
			Take(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, fiber.StatusNotFound, "data tidak ditemukan")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
		}
		return helper.JsonOK(c, "OK", toCSSTResp(row))
	}

	// ================= COUNT (ringan) =================
	countTx := ctl.DB.
		Table("class_section_subject_teachers AS csst").
		Where("csst.class_section_subject_teachers_masjid_id IN ?", masjidIDs)

	if q.WithDeleted == nil || !*q.WithDeleted {
		countTx = countTx.Where("csst.class_section_subject_teachers_deleted_at IS NULL")
	}
	if q.IsActive != nil {
		countTx = countTx.Where("csst.class_section_subject_teachers_is_active = ?", *q.IsActive)
	}
	// filter dasar yg tak butuh join
	if teacherID != "" {
		if _, e := uuid.Parse(teacherID); e == nil {
			countTx = countTx.Where("csst.class_section_subject_teachers_teacher_id = ?", teacherID)
		}
	}
	if masjidIDOne != "" {
		if _, e := uuid.Parse(masjidIDOne); e == nil {
			countTx = countTx.Where("csst.class_section_subject_teachers_masjid_id = ?", masjidIDOne)
		}
	}
	if classSubID != "" && csstClassSubjectFK != "" {
		if _, e := uuid.Parse(classSubID); e == nil {
			countTx = countTx.Where(fmt.Sprintf("csst.%s = ?", csstClassSubjectFK), classSubID)
		}
	}
	if sectionID != "" && csstSectionFK != "" {
		if _, e := uuid.Parse(sectionID); e == nil {
			countTx = countTx.Where(fmt.Sprintf("csst.%s = ?", csstSectionFK), sectionID)
		}
	}

	// join hanya jika perlu untuk filter/search q/subject
	needSubjectJoin := (subjectID != "" || (qtext != "" && (includes["subject"] || includes["section"])))
	needSectionJoin := (qtext != "" && (includes["section"] || includes["subject"]))
	if needSubjectJoin {
		if csstClassSubjectFK != "" {
			countTx = countTx.
				Joins(fmt.Sprintf(`LEFT JOIN class_subjects AS cs ON cs.class_subjects_id = csst.%s`, csstClassSubjectFK)).
				Joins(`LEFT JOIN subjects AS s ON s.subjects_id = cs.class_subjects_subject_id`)
		} else if csstSubjectFK != "" {
			countTx = countTx.Joins(fmt.Sprintf(`LEFT JOIN subjects AS s ON s.subjects_id = csst.%s`, csstSubjectFK))
		}
	}
	if needSectionJoin && csstSectionFK != "" {
		countTx = countTx.Joins(fmt.Sprintf(`LEFT JOIN class_sections AS sec ON sec.class_sections_id = csst.%s`, csstSectionFK))
	}
	if qtext != "" {
		switch {
		case includes["subject"] && includes["section"]:
			countTx = countTx.Where("(LOWER(s.subjects_name) LIKE ? OR LOWER(s.subjects_code) LIKE ? OR LOWER(sec.class_sections_name) LIKE ? OR LOWER(sec.class_sections_code) LIKE ?)",
				"%"+qtext+"%", "%"+qtext+"%", "%"+qtext+"%", "%"+qtext+"%")
		case includes["subject"]:
			countTx = countTx.Where("(LOWER(s.subjects_name) LIKE ? OR LOWER(s.subjects_code) LIKE ?)",
				"%"+qtext+"%", "%"+qtext+"%")
		case includes["section"]:
			countTx = countTx.Where("(LOWER(sec.class_sections_name) LIKE ? OR LOWER(sec.class_sections_code) LIKE ?)",
				"%"+qtext+"%", "%"+qtext+"%")
		}
	}

	var total int64
	countQuery := countTx.Select("csst.class_section_subject_teachers_id")
	if needSubjectJoin || needSectionJoin {
		countQuery = countQuery.Distinct("csst.class_section_subject_teachers_id")
	}
	if err := countQuery.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung total data")
	}

	// ================= LIST =================
	var rows []csstJoinedRow
	if err := tx.
		Select(strings.Join(selectCols, ", ")).
		Order(orderBy + " " + sort).
		Limit(*q.Limit).
		Offset(*q.Offset).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	out := make([]csstItemWithRefs, 0, len(rows))
	for _, r := range rows {
		out = append(out, toCSSTResp(r))
	}

	return helper.JsonList(c, out, fiber.Map{
		"limit":   *q.Limit,
		"offset":  *q.Offset,
		"total":   int(total),
		"include": includes,
	})
}
