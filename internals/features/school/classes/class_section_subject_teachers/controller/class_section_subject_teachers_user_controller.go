// file: internals/features/school/academics/subject/controller/class_section_subject_teachers_user_controller.go
package controller

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	modelCSST "schoolku_backend/internals/features/school/classes/class_section_subject_teachers/model"
	helper "schoolku_backend/internals/helpers"
	helperAuth "schoolku_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* ===================== Dynamic column detection (CSST) ==================== */

var (
	csstFKOnce sync.Once

	// PK & timestamps & tenant & status (dinamis)
	csstPK           string
	csstCreatedAtCol string
	csstUpdatedAtCol string
	csstDeletedAtCol string
	csstSchoolCol    string
	csstIsActiveCol  string

	// FK kandidat yang mungkin berbeda antar environment
	csstClassSubjectFK string // class_subject_id / class_subjects_id / csst...
	csstSubjectFK      string // subject_id / subjects_id / csst...
	csstSectionFK      string // class_section_id / section_id / class_sections_id / csst...
	csstTeacherFK      string // school teacher id kolom di csst
)

func hasColumn(db *gorm.DB, table, col string) bool {
	var n int64
	_ = db.Raw(`
        SELECT COUNT(*) FROM pg_attribute
        WHERE attrelid = to_regclass(?) AND attname = ? AND NOT attisdropped
    `, "public."+table, col).Scan(&n).Error
	return n > 0
}

func firstExisting(db *gorm.DB, table string, cands ...string) string {
	for _, c := range cands {
		if hasColumn(db, table, c) {
			return c
		}
	}
	return ""
}

func detectCSSTFKs(db *gorm.DB) {
	csstFKOnce.Do(func() {
		const tbl = "class_section_subject_teachers"

		// PK
		csstPK = firstExisting(db, tbl,
			"class_section_subject_teacher_id",
			"class_section_subject_teachers_id",
			"csst_id",
			"id",
		)
		if csstPK == "" {
			csstPK = "class_section_subject_teacher_id"
		}

		// timestamps
		csstCreatedAtCol = firstExisting(db, tbl,
			"class_section_subject_teacher_created_at",
			"class_section_subject_teachers_created_at",
			"created_at",
		)
		if csstCreatedAtCol == "" {
			csstCreatedAtCol = "created_at"
		}
		csstUpdatedAtCol = firstExisting(db, tbl,
			"class_section_subject_teacher_updated_at",
			"class_section_subject_teachers_updated_at",
			"updated_at",
		)
		if csstUpdatedAtCol == "" {
			csstUpdatedAtCol = "updated_at"
		}
		csstDeletedAtCol = firstExisting(db, tbl,
			"class_section_subject_teacher_deleted_at",
			"class_section_subject_teachers_deleted_at",
			"deleted_at",
		)
		if csstDeletedAtCol == "" {
			csstDeletedAtCol = "deleted_at"
		}

		// tenant & status
		csstSchoolCol = firstExisting(db, tbl,
			"class_section_subject_teacher_school_id",
			"class_section_subject_teachers_school_id",
			"school_id",
		)
		if csstSchoolCol == "" {
			csstSchoolCol = "school_id"
		}
		csstIsActiveCol = firstExisting(db, tbl,
			"class_section_subject_teacher_is_active",
			"class_section_subject_teachers_is_active",
			"is_active",
		)
		if csstIsActiveCol == "" {
			csstIsActiveCol = "is_active"
		}

		// FKs
		csstClassSubjectFK = firstExisting(db, tbl,
			"class_section_subject_teacher_class_subject_id",
			"class_section_subject_teachers_class_subject_id",
			"class_section_subject_teachers_class_subjects_id",
			"class_subject_id",
			"class_subjects_id",
		)
		csstSubjectFK = firstExisting(db, tbl,
			"class_section_subject_teacher_subject_id",
			"class_section_subject_teachers_subject_id",
			"subject_id",
			"subjects_id",
		)
		csstSectionFK = firstExisting(db, tbl,
			"class_section_subject_teacher_section_id",
			"class_section_subject_teachers_section_id",
			"class_section_id",
			"section_id",
			"class_sections_id",
		)
		csstTeacherFK = firstExisting(db, tbl,
			"class_section_subject_teacher_teacher_id",
			"class_section_subject_teachers_teacher_id",
			"teacher_id",
		)
	})
}

/* ======================= Lite structs for joined data ===================== */

type subjectLite struct {
	SubjectID   *uuid.UUID `json:"subject_id,omitempty"   gorm:"column:subject_id"`
	SubjectCode *string    `json:"subject_code,omitempty" gorm:"column:subject_code"`
	SubjectName *string    `json:"subject_name,omitempty" gorm:"column:subject_name"`
	SubjectSlug *string    `json:"subject_slug,omitempty" gorm:"column:subject_slug"`
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
	ClassSectionID   *uuid.UUID `json:"class_section_id,omitempty"   gorm:"column:class_section_id"`
	ClassSectionName *string    `json:"class_section_name,omitempty" gorm:"column:class_section_name"`
	ClassSectionCode *string    `json:"class_section_code,omitempty" gorm:"column:class_section_code"`
}

type csstJoinedRow struct {
	modelCSST.ClassSectionSubjectTeacherModel

	// class_subjects
	ClassSubjectID        *uuid.UUID `gorm:"column:class_subject_id"`
	ClassSubjectSubjectID *uuid.UUID `gorm:"column:class_subject_subject_id"`

	// subjects
	SubjectID   *uuid.UUID `gorm:"column:subject_id"`
	SubjectCode *string    `gorm:"column:subject_code"`
	SubjectName *string    `gorm:"column:subject_name"`
	SubjectSlug *string    `gorm:"column:subject_slug"`

	// sections
	ClassSectionID   *uuid.UUID `gorm:"column:class_section_id"`
	ClassSectionName *string    `gorm:"column:class_section_name"`
	ClassSectionCode *string    `gorm:"column:class_section_code"`

	// teachers
	TeacherIDJoin     *uuid.UUID `gorm:"column:teacher_id_join"`
	TeacherUserID     *uuid.UUID `gorm:"column:teacher_user_id"`
	TeacherTitle      *string    `gorm:"column:teacher_title"`
	TeacherCode       *string    `gorm:"column:teacher_code"`
	TeacherEmployment *string    `gorm:"column:teacher_employment"`
	TeacherIsActive   *bool      `gorm:"column:teacher_is_active"`
	TeacherName       *string    `gorm:"column:teacher_name"`
}

// ⬇️ DI SINI kita embed model langsung, supaya field CSST muncul langsung di root JSON
type csstItemWithRefs struct {
	modelCSST.ClassSectionSubjectTeacherModel

	ClassSubject *struct {
		ClassSubjectID        *uuid.UUID `json:"class_subject_id,omitempty"`
		ClassSubjectSubjectID *uuid.UUID `json:"class_subject_subject_id,omitempty"`
	} `json:"class_subject,omitempty"`

	Subject *subjectLite `json:"subject,omitempty"`
	Section *sectionLite `json:"section,omitempty"`
	Teacher *teacherLite `json:"teacher,omitempty"`
}

func toCSSTResp(r csstJoinedRow) csstItemWithRefs {
	out := csstItemWithRefs{
		ClassSectionSubjectTeacherModel: r.ClassSectionSubjectTeacherModel,
	}

	if r.ClassSubjectID != nil || r.ClassSubjectSubjectID != nil {
		out.ClassSubject = &struct {
			ClassSubjectID        *uuid.UUID `json:"class_subject_id,omitempty"`
			ClassSubjectSubjectID *uuid.UUID `json:"class_subject_subject_id,omitempty"`
		}{
			ClassSubjectID:        r.ClassSubjectID,
			ClassSubjectSubjectID: r.ClassSubjectSubjectID,
		}
	}
	if r.SubjectID != nil || r.SubjectName != nil || r.SubjectCode != nil || r.SubjectSlug != nil {
		out.Subject = &subjectLite{
			SubjectID:   r.SubjectID,
			SubjectCode: r.SubjectCode,
			SubjectName: r.SubjectName,
			SubjectSlug: r.SubjectSlug,
		}
	}
	if r.ClassSectionID != nil || r.ClassSectionName != nil || r.ClassSectionCode != nil {
		out.Section = &sectionLite{
			ClassSectionID:   r.ClassSectionID,
			ClassSectionName: r.ClassSectionName,
			ClassSectionCode: r.ClassSectionCode,
		}
	}
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

/* ============================= Query params ============================== */

type listQuery struct {
	IsActive    *bool   `query:"is_active"`
	WithDeleted *bool   `query:"with_deleted"`
	Limit       *int    `query:"limit"`
	Offset      *int    `query:"offset"`
	OrderBy     *string `query:"order_by"` // created_at|updated_at|subject_name|subject_code|section_name|section_code
	Sort        *string `query:"sort"`     // asc|desc
}

/* ================================ Handler (NO-INCLUDE, NO-JOIN) ================================ */

func (ctl *ClassSectionSubjectTeacherController) List(c *fiber.Ctx) error {
	detectCSSTFKs(ctl.DB)
	_ = csstSubjectFK

	// === School context: token → :school_id → :school_slug ===
	var schoolID uuid.UUID

	// 1) Dari token: active_school (sesuaikan dgn helper kamu)
	if sid, err := helperAuth.GetActiveSchoolID(c); err == nil && sid != uuid.Nil {
		schoolID = sid
	}

	// 2) Kalau belum ada, coba dari path :school_id
	if schoolID == uuid.Nil {
		if raw := strings.TrimSpace(c.Params("school_id")); raw != "" {
			id, err := uuid.Parse(raw)
			if err != nil || id == uuid.Nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "school_id path tidak valid")
			}
			schoolID = id
		}
	}

	// 3) Kalau masih kosong, coba dari :school_slug di path
	if schoolID == uuid.Nil {
		if slug := strings.TrimSpace(c.Params("school_slug")); slug != "" {
			id, er := helperAuth.GetSchoolIDBySlug(c, slug)
			if er != nil {
				if errors.Is(er, gorm.ErrRecordNotFound) {
					return helper.JsonError(c, fiber.StatusNotFound, "School (slug) tidak ditemukan")
				}
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal resolve school dari slug")
			}
			schoolID = id
		}
	}

	// 4) Kalau tetep nggak ada → context missing
	if schoolID == uuid.Nil {
		return helper.JsonError(
			c,
			helperAuth.ErrSchoolContextMissing.Code,
			helperAuth.ErrSchoolContextMissing.Message,
		)
	}

	// ❌ Jangan pakai EnsureStaffSchool di endpoint user (read-only)
	// if err := helperAuth.EnsureStaffSchool(c, schoolID); err != nil {
	// 	return err
	// }

	// path :id (detail)
	var pathID *uuid.UUID
	if s := strings.TrimSpace(c.Params("id")); s != "" {
		id, e := uuid.Parse(s)
		if e != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "id tidak valid")
		}
		pathID = &id
	}

	// query params umum
	var q listQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}

	// ===== Paging (standar jsonresponse) =====
	p := helper.ResolvePaging(c, 20, 200)
	if q.Offset != nil && *q.Offset >= 0 {
		p.Offset = *q.Offset
	}
	limit, offset := p.Limit, p.Offset

	sectionID := strings.TrimSpace(c.Query("section_id"))
	classSubID := strings.TrimSpace(c.Query("class_subject_id"))
	subjectID := strings.TrimSpace(c.Query("subject_id"))
	teacherID := strings.TrimSpace(c.Query("teacher_id"))
	qtext := strings.TrimSpace(strings.ToLower(c.Query("q")))

	orderBy := fmt.Sprintf("csst.%s", csstCreatedAtCol)
	if q.OrderBy != nil {
		switch strings.ToLower(*q.OrderBy) {
		case "created_at":
			orderBy = fmt.Sprintf("csst.%s", csstCreatedAtCol)
		case "updated_at":
			orderBy = fmt.Sprintf("csst.%s", csstUpdatedAtCol)
		case "display_name":
			orderBy = "COALESCE(csst.class_section_subject_teacher_display_name,'')"
		case "subject_name":
			orderBy = "COALESCE(csst.class_section_subject_teacher_class_subject_name_snap,'')"
		case "section_name":
			orderBy = "COALESCE(csst.class_section_subject_teacher_section_name_snap,'')"
		case "teacher_name":
			orderBy = "COALESCE(csst.class_section_subject_teacher_teacher_name_snap,'')"
		default:
			return helper.JsonError(c, fiber.StatusBadRequest, "order_by tidak dikenal (gunakan: created_at, updated_at, display_name, subject_name, section_name, teacher_name)")
		}
	}
	sort := "ASC"
	if q.Sort != nil && strings.ToLower(*q.Sort) == "desc" {
		sort = "DESC"
	}

	// BASE QUERY
	tx := ctl.DB.
		Table("class_section_subject_teachers AS csst").
		Where(fmt.Sprintf("csst.%s = ?", csstSchoolCol), schoolID)

	if q.WithDeleted == nil || !*q.WithDeleted {
		tx = tx.Where(fmt.Sprintf("csst.%s IS NULL", csstDeletedAtCol))
	}
	if q.IsActive != nil {
		tx = tx.Where(fmt.Sprintf("csst.%s = ?", csstIsActiveCol), *q.IsActive)
	}

	selectCols := []string{"csst.*"}

	// FILTERS
	if teacherID != "" && csstTeacherFK != "" {
		if _, e := uuid.Parse(teacherID); e == nil {
			tx = tx.Where(fmt.Sprintf("csst.%s = ?", csstTeacherFK), teacherID)
		}
	}
	if sectionID != "" && csstSectionFK != "" {
		if _, e := uuid.Parse(sectionID); e == nil {
			tx = tx.Where(fmt.Sprintf("csst.%s = ?", csstSectionFK), sectionID)
		}
	}
	if classSubID != "" && csstClassSubjectFK != "" {
		if _, e := uuid.Parse(classSubID); e == nil {
			tx = tx.Where(fmt.Sprintf("csst.%s = ?", csstClassSubjectFK), classSubID)
		}
	}
	if subjectID != "" && csstClassSubjectFK != "" {
		if _, e := uuid.Parse(subjectID); e == nil {
			tx = tx.Where(fmt.Sprintf(`
				EXISTS (
					SELECT 1 FROM class_subjects cs
					WHERE cs.class_subject_id = csst.%s
					  AND cs.class_subject_school_id = csst.%s
					  AND cs.class_subject_subject_id = ?
				)
			`, csstClassSubjectFK, csstSchoolCol), subjectID)
		}
	}
	if qtext != "" {
		tx = tx.Where("LOWER(csst.class_section_subject_teacher_display_name) LIKE ?", "%"+qtext+"%")
	}

	// DETAIL BY ID
	if pathID != nil {
		var row csstJoinedRow
		if err := tx.
			Select(strings.Join(selectCols, ", ")).
			Where(fmt.Sprintf("csst.%s = ?", csstPK), *pathID).
			Take(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, fiber.StatusNotFound, "data tidak ditemukan")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
		}
		return helper.JsonOK(c, "OK", toCSSTResp(row))
	}

	// COUNT
	countTx := ctl.DB.
		Table("class_section_subject_teachers AS csst").
		Where(fmt.Sprintf("csst.%s = ?", csstSchoolCol), schoolID)

	if q.WithDeleted == nil || !*q.WithDeleted {
		countTx = countTx.Where(fmt.Sprintf("csst.%s IS NULL", csstDeletedAtCol))
	}
	if q.IsActive != nil {
		countTx = countTx.Where(fmt.Sprintf("csst.%s = ?", csstIsActiveCol), *q.IsActive)
	}
	if teacherID != "" && csstTeacherFK != "" {
		if _, e := uuid.Parse(teacherID); e == nil {
			countTx = countTx.Where(fmt.Sprintf("csst.%s = ?", csstTeacherFK), teacherID)
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
	if subjectID != "" && csstClassSubjectFK != "" {
		if _, e := uuid.Parse(subjectID); e == nil {
			countTx = countTx.Where(fmt.Sprintf(`
				EXISTS (
					SELECT 1 FROM class_subjects cs
					WHERE cs.class_subject_id = csst.%s
					  AND cs.class_subject_school_id = csst.%s
					  AND cs.class_subject_subject_id = ?
				)
			`, csstClassSubjectFK, csstSchoolCol), subjectID)
		}
	}
	if qtext != "" {
		countTx = countTx.Where("LOWER(csst.class_section_subject_teacher_display_name) LIKE ?", "%"+qtext+"%")
	}

	var total int64
	if err := countTx.Select(fmt.Sprintf("csst.%s", csstPK)).Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung total data")
	}

	// LIST
	var rows []csstJoinedRow
	if err := tx.
		Select(strings.Join(selectCols, ", ")).
		Order(orderBy + " " + sort).
		Limit(limit).
		Offset(offset).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	out := make([]csstItemWithRefs, 0, len(rows))
	for _, r := range rows {
		out = append(out, toCSSTResp(r))
	}

	pg := helper.BuildPaginationFromOffset(total, offset, limit)
	return helper.JsonList(c, "ok", out, pg)
}
