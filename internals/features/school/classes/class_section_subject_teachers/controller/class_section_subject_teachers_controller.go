// file: internals/features/lembaga/class_section_subject_teachers/controller/csst_controller.go
package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	modelSchoolTeacher "schoolku_backend/internals/features/lembaga/school_yayasans/teachers_students/model"
	modelClassSection "schoolku_backend/internals/features/school/classes/class_sections/model"

	// DTO & Model
	dto "schoolku_backend/internals/features/school/classes/class_section_subject_teachers/dto"
	modelCSST "schoolku_backend/internals/features/school/classes/class_section_subject_teachers/model"

	roomSnapshot "schoolku_backend/internals/features/school/academics/rooms/snapshot"
	teacherSnapshot "schoolku_backend/internals/features/users/user_teachers/snapshot"
	helper "schoolku_backend/internals/helpers"
	helperAuth "schoolku_backend/internals/helpers/auth"
)

/* ===================== Helpers kecil ===================== */

func intPtr(v int) *int { return &v }

func ptrStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

/*
=========================================================

	SLUG base: Section + Subject (dari CSB) + Book Title (dari CSB)

=========================================================
*/
func getBaseForSlug(ctx context.Context, tx *gorm.DB, schoolID, sectionID, csbID, schoolTeacherID uuid.UUID) string {
	var sectionName, subjectName, bookTitle string

	_ = tx.Table("class_sections").
		Select("class_section_name").
		Where("class_section_id = ? AND class_section_school_id = ?", sectionID, schoolID).
		Scan(&sectionName).Error

	_ = tx.Table("class_subject_books AS csb").
		Select(`
			COALESCE(csb.class_subject_book_subject_name_snapshot, s.subject_name) AS subject_name,
			COALESCE(csb.class_subject_book_book_title_snapshot, b.book_title)     AS book_title
		`).
		Joins(`LEFT JOIN class_subjects cs ON cs.class_subject_id = csb.class_subject_book_class_subject_id AND cs.class_subject_deleted_at IS NULL`).
		Joins(`LEFT JOIN subjects s ON s.subject_id = cs.class_subject_subject_id AND s.subject_deleted_at IS NULL`).
		Joins(`LEFT JOIN books b ON b.book_id = csb.class_subject_book_book_id AND b.book_deleted_at IS NULL`).
		Where(`csb.class_subject_book_id = ?
		       AND csb.class_subject_book_school_id = ?
		       AND csb.class_subject_book_deleted_at IS NULL`, csbID, schoolID).
		Scan(&struct {
			SubjectName *string
			BookTitle   *string
		}{&subjectName, &bookTitle}).Error

	parts := []string{}
	if strings.TrimSpace(sectionName) != "" {
		parts = append(parts, sectionName)
	}
	if strings.TrimSpace(subjectName) != "" {
		parts = append(parts, subjectName)
	}
	if strings.TrimSpace(bookTitle) != "" {
		parts = append(parts, bookTitle)
	}
	if len(parts) > 0 {
		return strings.Join(parts, " ")
	}
	return fmt.Sprintf("csst-%s-%s-%s",
		strings.Split(sectionID.String(), "-")[0],
		strings.Split(csbID.String(), "-")[0],
		strings.Split(schoolTeacherID.String(), "-")[0],
	)
}

func ensureUniqueSlug(ctx context.Context, tx *gorm.DB, schoolID uuid.UUID, base string) (string, error) {
	return helper.EnsureUniqueSlugCI(
		ctx, tx,
		"class_section_subject_teachers", "class_section_subject_teacher_slug",
		base,
		func(q *gorm.DB) *gorm.DB {
			return q.Where(`
				class_section_subject_teacher_school_id = ?
				AND class_section_subject_teacher_deleted_at IS NULL
			`, schoolID)
		},
		160,
	)
}

/*
=========================================================

	Snapshot CSB (gabungan Book + Subject) → JSONB

=========================================================
*/
func buildCSBSnapshotJSON(ctx context.Context, tx *gorm.DB, schoolID, csbID uuid.UUID) (datatypes.JSON, error) {
	type rrow struct {
		BookID     uuid.UUID
		TitleSnap  *string
		AuthorSnap *string
		SlugSnap   *string
		ImageSnap  *string

		SubjectID   *uuid.UUID
		SubjectName *string
		SubjectCode *string
		SubjectSlug *string
	}
	var r rrow
	if err := tx.WithContext(ctx).
		Table("class_subject_books AS csb").
		Select(`
			csb.class_subject_book_book_id AS book_id,
			COALESCE(csb.class_subject_book_book_title_snapshot, b.book_title)        AS title_snap,
			COALESCE(csb.class_subject_book_book_author_snapshot, b.book_author)      AS author_snap,
			COALESCE(csb.class_subject_book_book_slug_snapshot, b.book_slug)          AS slug_snap,
			COALESCE(csb.class_subject_book_book_image_url_snapshot, b.book_image_url) AS image_snap,
			s.subject_id AS subject_id,
			COALESCE(csb.class_subject_book_subject_name_snapshot, s.subject_name) AS subject_name,
			COALESCE(csb.class_subject_book_subject_code_snapshot, s.subject_code) AS subject_code,
			COALESCE(csb.class_subject_book_subject_slug_snapshot, s.subject_slug) AS subject_slug
		`).
		Joins(`LEFT JOIN books b ON b.book_id = csb.class_subject_book_book_id AND b.book_deleted_at IS NULL`).
		Joins(`LEFT JOIN class_subjects cs ON cs.class_subject_id = csb.class_subject_book_class_subject_id AND cs.class_subject_deleted_at IS NULL`).
		Joins(`LEFT JOIN subjects s ON s.subject_id = cs.class_subject_subject_id AND s.subject_deleted_at IS NULL`).
		Where(`csb.class_subject_book_id = ?
		       AND csb.class_subject_book_school_id = ?
		       AND csb.class_subject_book_deleted_at IS NULL`, csbID, schoolID).
		Take(&r).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fiber.NewError(fiber.StatusNotFound, "class_subject_book tidak ditemukan / sudah dihapus")
		}
		return nil, err
	}

	payload := map[string]any{
		"book": map[string]any{
			"id":        r.BookID,
			"title":     r.TitleSnap,
			"author":    r.AuthorSnap,
			"slug":      r.SlugSnap,
			"image_url": r.ImageSnap,
		},
		"subject": map[string]any{
			"id":   r.SubjectID,
			"name": r.SubjectName,
			"code": r.SubjectCode,
			"slug": r.SubjectSlug,
			"url":  nil,
		},
	}
	b, _ := json.Marshal(payload)
	return datatypes.JSON(b), nil
}

/*
=========================================================

	Validasi konsistensi CSB untuk Section

=========================================================
*/
func validateCSBForSection(ctx context.Context, tx *gorm.DB, schoolID, sectionID, csbID uuid.UUID) error {
	// Parent dari Section
	var cls struct{ ClassParentID uuid.UUID }
	if err := tx.WithContext(ctx).
		Table("classes").
		Select("class_parent_id").
		Joins("JOIN class_sections s ON s.class_section_class_id = classes.class_id AND s.class_section_deleted_at IS NULL").
		Where("s.class_section_id = ? AND s.class_section_school_id = ? AND classes.class_deleted_at IS NULL", sectionID, schoolID).
		Take(&cls).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusBadRequest, "Kelas untuk section ini tidak ditemukan / beda tenant")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal cek kelas dari section")
	}

	// Tenant & Parent dari CSB -> ClassSubject
	var csb struct {
		SchoolID uuid.UUID
		ParentID uuid.UUID
	}
	if err := tx.WithContext(ctx).
		Table("class_subject_books AS csb").
		Select(`
			csb.class_subject_book_school_id AS school_id,
			cs.class_subject_parent_id       AS parent_id
		`).
		Joins(`JOIN class_subjects cs ON cs.class_subject_id = csb.class_subject_book_class_subject_id
		       AND cs.class_subject_deleted_at IS NULL`).
		Where("csb.class_subject_book_id = ? AND csb.class_subject_book_deleted_at IS NULL", csbID).
		Take(&csb).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusBadRequest, "class_subject_book tidak ditemukan / sudah dihapus")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal cek class_subject_book")
	}
	if csb.SchoolID != schoolID {
		return fiber.NewError(fiber.StatusBadRequest, "School mismatch: class_subject_book milik school lain")
	}
	if csb.ParentID != cls.ClassParentID {
		return fiber.NewError(fiber.StatusBadRequest, "Mismatch: parent kelas section ≠ parent pada class_subject_book")
	}
	return nil
}

type ClassSectionSubjectTeacherController struct {
	DB *gorm.DB
}

func NewClassSectionSubjectTeacherController(db *gorm.DB) *ClassSectionSubjectTeacherController {
	return &ClassSectionSubjectTeacherController{DB: db}
}

/* ======================== CREATE ======================== */
// POST /admin/:school_id/class-section-subject-teachers
func (ctl *ClassSectionSubjectTeacherController) Create(c *fiber.Ctx) error {
	mc, err := helperAuth.ResolveSchoolContext(c)
	if err != nil {
		return err
	}
	schoolID, err := helperAuth.EnsureSchoolAccessDKM(c, mc)
	if err != nil {
		return err
	}

	var req dto.CreateClassSectionSubjectTeacherRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid: "+err.Error())
	}
	if err := validator.New().Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return ctl.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		tx = tx.Debug()

		// 1) SECTION exists & same tenant
		var sec modelClassSection.ClassSectionModel
		if err := tx.
			Where("class_section_id = ? AND class_section_school_id = ? AND class_section_deleted_at IS NULL",
				req.ClassSectionSubjectTeacherClassSectionID, schoolID).
			First(&sec).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, fiber.StatusBadRequest, "Section tidak ditemukan / beda tenant")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal cek section")
		}

		// 2) VALIDASI CSB konsisten dengan section (tenant + parent)
		if err := validateCSBForSection(c.Context(), tx, schoolID, req.ClassSectionSubjectTeacherClassSectionID, req.ClassSectionSubjectTeacherClassSubjectBookID); err != nil {
			var fe *fiber.Error
			if errors.As(err, &fe) {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
		}

		// 3) TEACHER exists & same tenant
		if err := tx.
			Where("school_teacher_id = ? AND school_teacher_school_id = ? AND school_teacher_deleted_at IS NULL",
				req.ClassSectionSubjectTeacherSchoolTeacherID, schoolID).
			First(&modelSchoolTeacher.SchoolTeacherModel{}).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, fiber.StatusBadRequest, "Guru tidak ditemukan / bukan guru school ini")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal cek guru")
		}

		// 3a) ROOM resolve (request > default section room/snapshot)
		var finalClassRoomID *uuid.UUID
		var finalRoomSnap *roomSnapshot.RoomSnapshot
		var finalRoomJSON *datatypes.JSON

		if req.ClassSectionSubjectTeacherClassRoomID != nil {
			// explicit override dari request
			rs, err := roomSnapshot.ValidateAndSnapshotRoom(tx, schoolID, *req.ClassSectionSubjectTeacherClassRoomID)
			if err != nil {
				var fe *fiber.Error
				if errors.As(err, &fe) {
					return helper.JsonError(c, fe.Code, fe.Message)
				}
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal validasi ruangan")
			}
			tmp := *rs
			finalRoomSnap = &tmp
			finalClassRoomID = req.ClassSectionSubjectTeacherClassRoomID
		} else {
			// pakai default dari Section
			if sec.ClassSectionClassRoomIDSnapshot != nil {
				rs, err := roomSnapshot.ValidateAndSnapshotRoom(tx, schoolID, *sec.ClassSectionClassRoomIDSnapshot)
				if err != nil {
					var fe *fiber.Error
					if errors.As(err, &fe) {
						return helper.JsonError(c, fe.Code, fe.Message)
					}
					return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal ambil snapshot ruangan (section)")
				}
				tmp := *rs
				finalRoomSnap = &tmp
				idCopy := *sec.ClassSectionClassRoomIDSnapshot
				finalClassRoomID = &idCopy
			} else if sec.ClassSectionClassRoomSnapshot != nil && len(sec.ClassSectionClassRoomSnapshot) > 0 {
				jb := datatypes.JSON(sec.ClassSectionClassRoomSnapshot)
				finalRoomJSON = &jb
			}
		}

		// 4) Build row dari DTO
		row := req.ToModel()
		row.ClassSectionSubjectTeacherSchoolID = schoolID

		if finalClassRoomID != nil {
			row.ClassSectionSubjectTeacherClassRoomID = finalClassRoomID
		}
		if finalRoomSnap != nil {
			jb := roomSnapshot.ToJSON(finalRoomSnap)
			row.ClassSectionSubjectTeacherClassRoomSnapshot = &jb
		} else if finalRoomJSON != nil {
			row.ClassSectionSubjectTeacherClassRoomSnapshot = finalRoomJSON
		}

		// Default delivery mode jika kosong
		if strings.TrimSpace(string(row.ClassSectionSubjectTeacherDeliveryMode)) == "" {
			if finalRoomSnap != nil && (finalRoomSnap.IsVirtual || (finalRoomSnap.JoinURL != nil && strings.TrimSpace(*finalRoomSnap.JoinURL) != "")) {
				row.ClassSectionSubjectTeacherDeliveryMode = modelCSST.DeliveryModeOnline
			} else {
				row.ClassSectionSubjectTeacherDeliveryMode = modelCSST.DeliveryModeOffline
			}
		}

		// 4a) SNAPSHOT GURU
		if ts, err := teacherSnapshot.BuildTeacherSnapshot(c.Context(), tx, schoolID, req.ClassSectionSubjectTeacherSchoolTeacherID); err != nil {
			switch {
			case errors.Is(err, gorm.ErrRecordNotFound):
				return helper.JsonError(c, fiber.StatusBadRequest, "Guru tidak valid / sudah dihapus")
			case errors.Is(err, teacherSnapshot.ErrSchoolMismatch):
				return helper.JsonError(c, fiber.StatusForbidden, "Guru bukan milik school Anda")
			default:
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat snapshot guru")
			}
		} else if ts != nil {
			if jb, e := teacherSnapshot.ToJSONB(ts); e == nil {
				row.ClassSectionSubjectTeacherSchoolTeacherSnapshot = &jb
			}
		}

		// 4b) SNAPSHOT ASISTEN (opsional)
		if req.ClassSectionSubjectTeacherAssistantSchoolTeacherID != nil {
			if ats, err := teacherSnapshot.BuildTeacherSnapshot(c.Context(), tx, schoolID, *req.ClassSectionSubjectTeacherAssistantSchoolTeacherID); err != nil {
				switch {
				case errors.Is(err, gorm.ErrRecordNotFound):
					return helper.JsonError(c, fiber.StatusBadRequest, "Asisten guru tidak valid / sudah dihapus")
				case errors.Is(err, teacherSnapshot.ErrSchoolMismatch):
					return helper.JsonError(c, fiber.StatusForbidden, "Asisten guru bukan milik school Anda")
				default:
					return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat snapshot asisten guru")
				}
			} else if ats != nil {
				if jb, e := teacherSnapshot.ToJSONB(ats); e == nil {
					row.ClassSectionSubjectTeacherAssistantSchoolTeacherSnapshot = &jb
				}
			}
		}

		// 4c) SNAPSHOT CLASS_SUBJECT_BOOK (gabungan)
		if j, err := buildCSBSnapshotJSON(c.Context(), tx, schoolID, row.ClassSectionSubjectTeacherClassSubjectBookID); err != nil {
			var fe *fiber.Error
			if errors.As(err, &fe) {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat snapshot class_subject_book")
		} else {
			row.ClassSectionSubjectTeacherClassSubjectBookSnapshot = &j
		}

		// 5) SLUG unik (tanpa name)
		if row.ClassSectionSubjectTeacherSlug != nil {
			s := helper.Slugify(*row.ClassSectionSubjectTeacherSlug, 160)
			row.ClassSectionSubjectTeacherSlug = &s
		}
		base := strings.TrimSpace(getBaseForSlug(
			c.Context(), tx, schoolID,
			row.ClassSectionSubjectTeacherClassSectionID,
			row.ClassSectionSubjectTeacherClassSubjectBookID,
			row.ClassSectionSubjectTeacherSchoolTeacherID,
		))
		candidate := base
		if row.ClassSectionSubjectTeacherSlug != nil && strings.TrimSpace(*row.ClassSectionSubjectTeacherSlug) != "" {
			candidate = *row.ClassSectionSubjectTeacherSlug
		}
		candidate = helper.Slugify(candidate, 160)
		uniqueSlug, err := ensureUniqueSlug(c.Context(), tx, schoolID, candidate)
		if err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
		}
		row.ClassSectionSubjectTeacherSlug = &uniqueSlug

		// 6) INSERT
		if err := tx.Create(&row).Error; err != nil {
			msg := strings.ToLower(err.Error())
			switch {
			case strings.Contains(msg, "uq_csst_one_active_per_section_csb_alive"),
				strings.Contains(msg, "uq_csst_unique_alive"),
				strings.Contains(msg, "duplicate"),
				strings.Contains(msg, "unique"):
				if strings.Contains(msg, "uq_csst_slug_per_tenant_alive") {
					return helper.JsonError(c, fiber.StatusConflict, "Slug sudah digunakan pada tenant ini.")
				}
				return helper.JsonError(c, fiber.StatusConflict, "Penugasan sudah terdaftar (duplikat).")
			case strings.Contains(msg, "23503"), strings.Contains(msg, "foreign key"):
				switch {
				case strings.Contains(msg, "class_sections"):
					return helper.JsonError(c, fiber.StatusBadRequest, "FK gagal (SECTION): section tidak ditemukan / beda tenant")
				case strings.Contains(msg, "class_subject_books"):
					return helper.JsonError(c, fiber.StatusBadRequest, "FK gagal (CLASS_SUBJECT_BOOK): tidak ditemukan / beda tenant")
				case strings.Contains(msg, "school_teachers"):
					return helper.JsonError(c, fiber.StatusBadRequest, "FK gagal (GURU): guru tidak ditemukan / beda tenant")
				case strings.Contains(msg, "class_rooms"):
					return helper.JsonError(c, fiber.StatusBadRequest, "FK gagal (ROOM): ruangan tidak ditemukan / beda tenant")
				default:
					return helper.JsonError(c, fiber.StatusBadRequest, "FK gagal: pastikan section/CSB/guru/room valid")
				}
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Insert gagal: "+err.Error())
		}

		return helper.JsonCreated(c, "Penugasan guru berhasil dibuat", dto.FromClassSectionSubjectTeacherModel(row))
	})
}

/* ======================== UPDATE (partial) ======================== */
// PUT /admin/:school_id/class-section-subject-teachers/:id
func (ctl *ClassSectionSubjectTeacherController) Update(c *fiber.Ctx) error {
	mc, err := helperAuth.ResolveSchoolContext(c)
	if err != nil {
		return err
	}
	schoolID, err := helperAuth.EnsureSchoolAccessDKM(c, mc)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	var req dto.UpdateClassSectionSubjectTeacherRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := validator.New().Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return ctl.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		// Ambil row
		var row modelCSST.ClassSectionSubjectTeacherModel
		if err := tx.
			Where("class_section_subject_teacher_id = ? AND class_section_subject_teacher_deleted_at IS NULL", id).
			First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, http.StatusNotFound, "Data tidak ditemukan")
			}
			return helper.JsonError(c, http.StatusInternalServerError, err.Error())
		}
		// Tenant guard
		if row.ClassSectionSubjectTeacherSchoolID != schoolID {
			return helper.JsonError(c, http.StatusForbidden, "Akses ditolak")
		}

		// Precheck konsistensi jika section/CSB berubah
		if req.ClassSectionSubjectTeacherClassSectionID != nil || req.ClassSectionSubjectTeacherClassSubjectBookID != nil {
			sectionID := row.ClassSectionSubjectTeacherClassSectionID
			if req.ClassSectionSubjectTeacherClassSectionID != nil {
				sectionID = *req.ClassSectionSubjectTeacherClassSectionID
			}
			csbID := row.ClassSectionSubjectTeacherClassSubjectBookID
			if req.ClassSectionSubjectTeacherClassSubjectBookID != nil {
				csbID = *req.ClassSectionSubjectTeacherClassSubjectBookID
			}
			if err := validateCSBForSection(c.Context(), tx, schoolID, sectionID, csbID); err != nil {
				var fe *fiber.Error
				if errors.As(err, &fe) {
					return helper.JsonError(c, fe.Code, fe.Message)
				}
				return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
			}
		}

		// Flags perubahan untuk slug/snapshot
		sectionChanged := req.ClassSectionSubjectTeacherClassSectionID != nil &&
			*req.ClassSectionSubjectTeacherClassSectionID != row.ClassSectionSubjectTeacherClassSectionID
		csbChanged := req.ClassSectionSubjectTeacherClassSubjectBookID != nil &&
			*req.ClassSectionSubjectTeacherClassSubjectBookID != row.ClassSectionSubjectTeacherClassSubjectBookID
		teacherChanged := req.ClassSectionSubjectTeacherSchoolTeacherID != nil &&
			*req.ClassSectionSubjectTeacherSchoolTeacherID != row.ClassSectionSubjectTeacherSchoolTeacherID

		// Apply perubahan dasar
		req.Apply(&row)

		// Rebuild snapshot jika CSB berubah
		if csbChanged {
			if j, err := buildCSBSnapshotJSON(c.Context(), tx, schoolID, row.ClassSectionSubjectTeacherClassSubjectBookID); err != nil {
				var fe *fiber.Error
				if errors.As(err, &fe) {
					return helper.JsonError(c, fe.Code, fe.Message)
				}
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat snapshot class_subject_book")
			} else {
				row.ClassSectionSubjectTeacherClassSubjectBookSnapshot = &j
			}
		}

		// SLUG handling (mirror create — tanpa “name”)
		if req.ClassSectionSubjectTeacherSlug != nil {
			if s := strings.TrimSpace(*req.ClassSectionSubjectTeacherSlug); s != "" {
				norm := helper.Slugify(s, 160)
				row.ClassSectionSubjectTeacherSlug = &norm
			} else {
				row.ClassSectionSubjectTeacherSlug = nil
			}
		}

		needEnsureUnique := false
		baseSlug := ""
		if req.ClassSectionSubjectTeacherSlug != nil {
			needEnsureUnique = true
			if row.ClassSectionSubjectTeacherSlug != nil {
				baseSlug = *row.ClassSectionSubjectTeacherSlug
			}
		} else if row.ClassSectionSubjectTeacherSlug == nil || strings.TrimSpace(ptrStr(row.ClassSectionSubjectTeacherSlug)) == "" {
			if sectionChanged || csbChanged || teacherChanged || row.ClassSectionSubjectTeacherSlug == nil {
				needEnsureUnique = true
				baseSlug = getBaseForSlug(
					c.Context(), tx, schoolID,
					row.ClassSectionSubjectTeacherClassSectionID,
					row.ClassSectionSubjectTeacherClassSubjectBookID,
					row.ClassSectionSubjectTeacherSchoolTeacherID,
				)
				baseSlug = helper.Slugify(baseSlug, 160)
			}
		}

		if needEnsureUnique {
			uniqueSlug, err := helper.EnsureUniqueSlugCI(
				c.Context(),
				tx,
				"class_section_subject_teachers",
				"class_section_subject_teacher_slug",
				baseSlug,
				func(q *gorm.DB) *gorm.DB {
					return q.Where(`
						class_section_subject_teacher_school_id = ?
						AND class_section_subject_teacher_deleted_at IS NULL
						AND class_section_subject_teacher_id <> ?
					`, schoolID, row.ClassSectionSubjectTeacherID)
				},
				160,
			)
			if err != nil {
				return helper.JsonError(c, http.StatusInternalServerError, "Gagal menghasilkan slug unik")
			}
			if strings.TrimSpace(uniqueSlug) != "" {
				row.ClassSectionSubjectTeacherSlug = &uniqueSlug
			} else {
				row.ClassSectionSubjectTeacherSlug = nil
			}
		}

		// Persist
		if err := tx.Save(&row).Error; err != nil {
			msg := strings.ToLower(err.Error())
			if strings.Contains(msg, "uq_csst_one_active_per_section_csb_alive") ||
				strings.Contains(msg, "uq_csst_unique_alive") ||
				strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique") {
				if strings.Contains(msg, "uq_csst_slug_per_tenant_alive") {
					return helper.JsonError(c, fiber.StatusConflict, "Slug sudah digunakan pada tenant ini.")
				}
				return helper.JsonError(c, fiber.StatusConflict, "Penugasan guru untuk CSB ini sudah aktif (duplikat).")
			}
			if strings.Contains(msg, "sqlstate 23503") || strings.Contains(msg, "foreign key") {
				switch {
				case strings.Contains(msg, "class_sections"):
					return helper.JsonError(c, fiber.StatusBadRequest, "FK gagal (SECTION): section tidak ditemukan / beda tenant")
				case strings.Contains(msg, "class_subject_books"):
					return helper.JsonError(c, fiber.StatusBadRequest, "FK gagal (CLASS_SUBJECT_BOOK): tidak ditemukan / beda tenant")
				case strings.Contains(msg, "school_teachers"):
					return helper.JsonError(c, fiber.StatusBadRequest, "FK gagal (GURU): guru tidak ditemukan / beda tenant")
				case strings.Contains(msg, "class_rooms"):
					return helper.JsonError(c, fiber.StatusBadRequest, "FK gagal (ROOM): ruangan tidak ditemukan / beda tenant")
				default:
					return helper.JsonError(c, fiber.StatusBadRequest, "FK gagal: pastikan section/CSB/guru/room valid")
				}
			}
			return helper.JsonError(c, http.StatusInternalServerError, err.Error())
		}

		return helper.JsonUpdated(c, "Penugasan guru berhasil diperbarui", dto.FromClassSectionSubjectTeacherModel(row))
	})
}

/* ======================== DELETE (soft) ======================== */
func (ctl *ClassSectionSubjectTeacherController) Delete(c *fiber.Ctx) error {
	mc, err := helperAuth.ResolveSchoolContext(c)
	if err != nil {
		return err
	}
	schoolID, err := helperAuth.EnsureSchoolAccessDKM(c, mc)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(http.StatusBadRequest, "ID tidak valid")
	}

	var row modelCSST.ClassSectionSubjectTeacherModel
	if err := ctl.DB.WithContext(c.Context()).
		First(&row, "class_section_subject_teacher_id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, http.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}
	if row.ClassSectionSubjectTeacherSchoolID != schoolID {
		return helper.JsonError(c, http.StatusForbidden, "Akses ditolak")
	}
	if row.ClassSectionSubjectTeacherDeletedAt.Valid {
		return helper.JsonDeleted(c, "Sudah terhapus", fiber.Map{"id": id})
	}

	if err := ctl.DB.WithContext(c.Context()).
		Model(&modelCSST.ClassSectionSubjectTeacherModel{}).
		Where("class_section_subject_teacher_id = ?", id).
		Update("class_section_subject_teacher_deleted_at", gorm.Expr("NOW()")).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	return helper.JsonDeleted(c, "Penugasan guru berhasil dihapus", fiber.Map{"id": id})
}
