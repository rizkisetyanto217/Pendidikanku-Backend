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
}

type csstItemWithRefs struct {
	CSST         modelCSST.ClassSectionSubjectTeacherModel `json:"csst"`
	ClassSubject *struct {
		ClassSubjectsID        *uuid.UUID `json:"class_subjects_id,omitempty"`
		ClassSubjectsSubjectID *uuid.UUID `json:"class_subjects_subject_id,omitempty"`
	} `json:"class_subject,omitempty"`
	Subject *subjectLite `json:"subject,omitempty"`
	Section *sectionLite `json:"section,omitempty"`
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
	if r.SubjectsID != nil || r.SubjectsName != nil || r.SubjectsCode != nil {
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

// -------------------------------------------------------------
// LIST handler
// GET /api/{a|u}/class-section-subject-teachers/list
// Optional :id => detail by id
// Filter qparams: section_id, class_subject_id, subject_id, teacher_id, masjid_id, q
// -------------------------------------------------------------
func (ctl *ClassSectionSubjectTeacherController) List(c *fiber.Ctx) error {
	detectCSSTFKs(ctl.DB)

	masjidIDs, err := helperAuth.GetMasjidIDsFromToken(c)
	if err != nil {
		return err
	}

	// GET /:id ? ...
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

	// sorting
	orderBy := "csst.class_section_subject_teachers_created_at"
	if q.OrderBy != nil {
		switch strings.ToLower(*q.OrderBy) {
		case "created_at":
			orderBy = "csst.class_section_subject_teachers_created_at"
		case "updated_at":
			orderBy = "csst.class_section_subject_teachers_updated_at"
		case "subject_name":
			orderBy = "COALESCE(s.subjects_name,'')"
		case "subject_code":
			orderBy = "COALESCE(s.subjects_code,'')"
		case "section_name":
			orderBy = "COALESCE(sec.class_sections_name,'')"
		case "section_code":
			orderBy = "COALESCE(sec.class_sections_code,'')"
		}
	}
	sort := "ASC"
	if q.Sort != nil && strings.ToLower(*q.Sort) == "desc" {
		sort = "DESC"
	}

	// base
	tx := ctl.DB.
		Table("class_section_subject_teachers AS csst").
		Where("csst.class_section_subject_teachers_masjid_id IN ?", masjidIDs)

	if q.WithDeleted == nil || !*q.WithDeleted {
		tx = tx.Where("csst.class_section_subject_teachers_deleted_at IS NULL")
	}
	if q.IsActive != nil {
		tx = tx.Where("csst.class_section_subject_teachers_is_active = ?", *q.IsActive)
	}

	// SELECT kolom
	selectCols := []string{"csst.*"}
	joinedS := false
	joinedSec := false

	// JOIN class_subjects & subjects bila kolom FK tersedia
	if csstClassSubjectFK != "" {
		tx = tx.Joins(fmt.Sprintf(`
            LEFT JOIN class_subjects AS cs
              ON cs.class_subjects_id = csst.%s
        `, csstClassSubjectFK))
		selectCols = append(selectCols,
			"cs.class_subjects_id AS class_subjects_id",
			"cs.class_subjects_subject_id AS class_subjects_subject_id",
		)

		tx = tx.Joins(`LEFT JOIN subjects AS s ON s.subjects_id = cs.class_subjects_subject_id`)
		selectCols = append(selectCols,
			"s.subjects_id AS subjects_id",
			"s.subjects_code AS subjects_code",
			"s.subjects_name AS subjects_name",
			"s.subjects_slug AS subjects_slug",
		)
		joinedS = true
	} else if csstSubjectFK != "" {
		tx = tx.Joins(fmt.Sprintf(`LEFT JOIN subjects AS s ON s.subjects_id = csst.%s`, csstSubjectFK))
		selectCols = append(selectCols,
			"s.subjects_id AS subjects_id",
			"s.subjects_code AS subjects_code",
			"s.subjects_name AS subjects_name",
			"s.subjects_slug AS subjects_slug",
		)
		joinedS = true
	}

	// JOIN class_sections bila kolom FK tersedia
	if csstSectionFK != "" {
		tx = tx.Joins(fmt.Sprintf(`LEFT JOIN class_sections AS sec ON sec.class_sections_id = csst.%s`, csstSectionFK))
		selectCols = append(selectCols,
			"sec.class_sections_id   AS class_sections_id",
			"sec.class_sections_name AS class_sections_name",
			"sec.class_sections_code AS class_sections_code",
		)
		joinedSec = true
	}

	// FILTERS
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
	if subjectID != "" && joinedS {
		if _, e := uuid.Parse(subjectID); e == nil {
			tx = tx.Where("s.subjects_id = ?", subjectID)
		}
	}
	if qtext != "" {
		if joinedS && joinedSec {
			tx = tx.Where("(LOWER(s.subjects_name) LIKE ? OR LOWER(s.subjects_code) LIKE ? OR LOWER(sec.class_sections_name) LIKE ? OR LOWER(sec.class_sections_code) LIKE ?)",
				"%"+qtext+"%", "%"+qtext+"%", "%"+qtext+"%", "%"+qtext+"%")
		} else if joinedS {
			tx = tx.Where("(LOWER(s.subjects_name) LIKE ? OR LOWER(s.subjects_code) LIKE ?)", "%"+qtext+"%", "%"+qtext+"%")
		} else if joinedSec {
			tx = tx.Where("(LOWER(sec.class_sections_name) LIKE ? OR LOWER(sec.class_sections_code) LIKE ?)", "%"+qtext+"%", "%"+qtext+"%")
		}
	}

	// GET BY ID
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

	// COUNT (ikuti filter yang sama; aman dari join duplicates)
	var total int64
	if err := tx.Session(&gorm.Session{}).
		Select("csst.class_section_subject_teachers_id").
		Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung total data")
	}

	// LIST
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
		"limit":  *q.Limit,
		"offset": *q.Offset,
		"total":  int(total),
	})
}
