// internals/features/lembaga/class_section_subject_teachers/controller/csst_controller.go
package controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	modelMasjidTeacher "masjidku_backend/internals/features/lembaga/masjid_yayasans/teachers_students/model"
	modelClassSection "masjidku_backend/internals/features/school/classes/class_sections/model"

	// === pakai model & dto CSST terbaru (sectionsubjectteachers) ===
	dto "masjidku_backend/internals/features/school/classes/class_section_subject_teachers/dto"
	modelCSST "masjidku_backend/internals/features/school/classes/class_section_subject_teachers/model"

	roomSnapshot "masjidku_backend/internals/features/school/academics/rooms/snapshot"
	teacherSnapshot "masjidku_backend/internals/features/users/user_teachers/snapshot"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	snapshotClassSubject "masjidku_backend/internals/features/school/academics/subjects/snapshot"
)

type ClassSectionSubjectTeacherController struct {
	DB *gorm.DB
}

func NewClassSectionSubjectTeacherController(db *gorm.DB) *ClassSectionSubjectTeacherController {
	return &ClassSectionSubjectTeacherController{DB: db}
}

func intPtr(v int) *int { return &v }

// util kecil
func ptrStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// =====================
// Helpers (private)
// =====================

func getBaseForSlug(ctx context.Context, tx *gorm.DB, masjidID, sectionID, classSubjectID, teacherID uuid.UUID) string {
	var sectionName, subjectName string

	_ = tx.Table("class_sections").
		Select("class_section_name").
		Where("class_section_id = ? AND class_section_masjid_id = ?", sectionID, masjidID).
		Scan(&sectionName).Error

	_ = tx.Table("class_subjects AS cs").
		Select("s.subject_name").
		Joins(`JOIN subjects AS s
		           ON s.subject_id = cs.class_subject_subject_id
		          AND s.subject_deleted_at IS NULL`).
		Where(`cs.class_subject_id = ?
		           AND cs.class_subject_masjid_id = ?
		           AND cs.class_subject_deleted_at IS NULL`,
			classSubjectID, masjidID).
		Scan(&subjectName).Error

	var parts []string
	if strings.TrimSpace(sectionName) != "" {
		parts = append(parts, sectionName)
	}
	if strings.TrimSpace(subjectName) != "" {
		parts = append(parts, subjectName)
	}
	if len(parts) > 0 {
		return strings.Join(parts, " ")
	}
	return fmt.Sprintf("csst-%s-%s-%s",
		strings.Split(sectionID.String(), "-")[0],
		strings.Split(classSubjectID.String(), "-")[0],
		strings.Split(teacherID.String(), "-")[0],
	)
}

func ensureUniqueSlug(ctx context.Context, tx *gorm.DB, masjidID uuid.UUID, base string) (string, error) {
	return helper.EnsureUniqueSlugCI(
		ctx, tx,
		"class_section_subject_teachers", "class_section_subject_teacher_slug",
		base,
		func(q *gorm.DB) *gorm.DB {
			return q.Where(`
				class_section_subject_teacher_masjid_id = ?
				AND class_section_subject_teacher_deleted_at IS NULL
			`, masjidID)
		},
		160,
	)
}

func deriveNameFromSlug(slug string) string {
	// "-" -> " ", hilangkan spasi beruntun, crop ke 160 chars
	name := strings.ReplaceAll(slug, "-", " ")
	name = strings.TrimSpace(strings.Join(strings.Fields(name), " "))
	if len(name) > 160 {
		name = name[:160]
	}
	return name
}

// * CREATE (admin/DKM via masjid context)
// POST /admin/:masjid_id/class-section-subject-teachers
// /admin/:masjid_slug/class-section-subject-teachers
func (ctl *ClassSectionSubjectTeacherController) Create(c *fiber.Ctx) error {
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		fmt.Println("[CSST.Create] resolve masjid context ERROR:", err)
		return err
	}
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		fmt.Println("[CSST.Create] ensure access DKM ERROR:", err)
		return err
	}

	var req dto.CreateClassSectionSubjectTeacherRequest
	if err := c.BodyParser(&req); err != nil {
		fmt.Println("[CSST.Create] BodyParser ERROR:", err)
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid: "+err.Error())
	}
	if err := validator.New().Struct(req); err != nil {
		fmt.Println("[CSST.Create] validator ERROR:", err)
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	fmt.Printf("[CSST.Create] REQUEST masjid=%s section=%s class_subject=%s teacher=%s\n",
		masjidID, req.ClassSectionSubjectTeacherSectionID, req.ClassSectionSubjectTeacherClassSubjectID, req.ClassSectionSubjectTeacherTeacherID)

	return ctl.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		tx = tx.Debug()

		// 1) SECTION exists & same tenant
		var sec modelClassSection.ClassSectionModel
		if err := tx.
			Where("class_section_id = ? AND class_section_masjid_id = ? AND class_section_deleted_at IS NULL",
				req.ClassSectionSubjectTeacherSectionID, masjidID).
			First(&sec).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				fmt.Println("[CSST.Create] SECTION not found / tenant mismatch")
				return helper.JsonError(c, fiber.StatusBadRequest, "Section tidak ditemukan / beda tenant")
			}
			fmt.Println("[CSST.Create] SECTION check ERROR:", err)
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal cek section")
		}
		fmt.Println("[CSST.Create] SECTION ok:", sec.ClassSectionID)

		// 2) CLASS_SUBJECT exists + parent harus sama dgn kelas section
		var cls struct{ ClassParentID uuid.UUID }
		if err := tx.Table("classes").
			Select("class_parent_id").
			Where("class_id = ? AND class_masjid_id = ? AND class_deleted_at IS NULL",
				sec.ClassSectionClassID, masjidID).
			Take(&cls).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				fmt.Println("[CSST.Create] CLASS for section not found / tenant mismatch")
				return helper.JsonError(c, fiber.StatusBadRequest, "Kelas untuk section ini tidak ditemukan / beda tenant")
			}
			fmt.Println("[CSST.Create] CLASS check ERROR:", err)
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal cek kelas dari section")
		}

		var cs struct {
			MasjidID uuid.UUID
			ParentID uuid.UUID
		}
		if err := tx.Table("class_subjects").
			Select("class_subject_masjid_id AS masjid_id, class_subject_parent_id AS parent_id").
			Where("class_subject_id = ? AND class_subject_deleted_at IS NULL",
				req.ClassSectionSubjectTeacherClassSubjectID).
			Take(&cs).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				fmt.Println("[CSST.Create] CLASS_SUBJECT not found / deleted")
				return helper.JsonError(c, fiber.StatusBadRequest, "class_subject tidak ditemukan / sudah dihapus")
			}
			fmt.Println("[CSST.Create] CLASS_SUBJECT check ERROR:", err)
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal cek class_subject")
		}
		if cs.MasjidID != masjidID {
			fmt.Printf("[CSST.Create] CLASS_SUBJECT tenant mismatch: have=%s want=%s\n", cs.MasjidID, masjidID)
			return helper.JsonError(c, fiber.StatusBadRequest, "Masjid mismatch: class_subject milik masjid lain")
		}
		if cs.ParentID != cls.ClassParentID {
			fmt.Printf("[CSST.Create] PARENT mismatch: section.parent=%s subject.parent=%s\n", cls.ClassParentID, cs.ParentID)
			return helper.JsonError(c, fiber.StatusBadRequest, "Mismatch: parent kelas section ≠ parent pada class_subject")
		}
		fmt.Println("[CSST.Create] CLASS_SUBJECT ok")

		// 3) TEACHER exists & same tenant
		if err := tx.
			Where("masjid_teacher_id = ? AND masjid_teacher_masjid_id = ? AND masjid_teacher_deleted_at IS NULL",
				req.ClassSectionSubjectTeacherTeacherID, masjidID).
			First(&modelMasjidTeacher.MasjidTeacherModel{}).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				fmt.Println("[CSST.Create] TEACHER not found / tenant mismatch")
				return helper.JsonError(c, fiber.StatusBadRequest, "Guru tidak ditemukan / bukan guru masjid ini")
			}
			fmt.Println("[CSST.Create] TEACHER check ERROR:", err)
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal cek guru")
		}
		fmt.Println("[CSST.Create] TEACHER ok")

		// 3a) RESOLVE ROOM strictly dari DB
		var finalRoomID *uuid.UUID
		var finalRoomSnap *roomSnapshot.RoomSnapshot
		if req.ClassSectionSubjectTeacherRoomID != nil {
			rs, err := roomSnapshot.ValidateAndSnapshotRoom(tx, masjidID, *req.ClassSectionSubjectTeacherRoomID)
			if err != nil {
				var fe *fiber.Error
				if errors.As(err, &fe) {
					return helper.JsonError(c, fe.Code, fe.Message)
				}
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal validasi ruangan")
			}
			tmp := *rs
			finalRoomSnap = &tmp
			finalRoomID = req.ClassSectionSubjectTeacherRoomID
			fmt.Println("[CSST.Create] ROOM snapshot from explicit CSST room_id")
		} else if sec.ClassSectionClassRoomID != nil {
			rs, err := roomSnapshot.ValidateAndSnapshotRoom(tx, masjidID, *sec.ClassSectionClassRoomID)
			if err != nil {
				var fe *fiber.Error
				if errors.As(err, &fe) {
					return helper.JsonError(c, fe.Code, fe.Message)
				}
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal ambil snapshot ruangan (section)")
			}
			tmp := *rs
			finalRoomSnap = &tmp
			finalRoomID = sec.ClassSectionClassRoomID
			fmt.Println("[CSST.Create] ROOM snapshot from SECTION default room (fresh DB)")
		} else {
			fmt.Println("[CSST.Create] no room provided & section has no default room")
		}

		// 4) Build row dari DTO
		row := req.ToModel()
		row.ClassSectionSubjectTeacherMasjidID = masjidID

		// set room & snapshot (jika ada)
		if finalRoomID != nil {
			row.ClassSectionSubjectTeacherRoomID = finalRoomID
		}
		if finalRoomSnap != nil {
			jb := roomSnapshot.ToJSON(finalRoomSnap)
			row.ClassSectionSubjectTeacherRoomSnapshot = &jb
		}

		// Status active default
		if req.ClassSectionSubjectTeacherIsActive != nil {
			row.ClassSectionSubjectTeacherIsActive = *req.ClassSectionSubjectTeacherIsActive
		} else {
			row.ClassSectionSubjectTeacherIsActive = true
		}

		// Delivery mode default (auto)
		if strings.TrimSpace(string(row.ClassSectionsSubjectTeacherDeliveryMode)) == "" {
			if finalRoomSnap != nil && (finalRoomSnap.IsVirtual || (finalRoomSnap.JoinURL != nil && strings.TrimSpace(*finalRoomSnap.JoinURL) != "")) {
				row.ClassSectionsSubjectTeacherDeliveryMode = modelCSST.ClassDeliveryMode("online")
				fmt.Println("[CSST.Create] delivery_mode auto=online (virtual/join_url)")
			} else {
				row.ClassSectionsSubjectTeacherDeliveryMode = modelCSST.ClassDeliveryMode("offline")
				fmt.Println("[CSST.Create] delivery_mode auto=offline")
			}
		} else {
			fmt.Println("[CSST.Create] delivery_mode input:", row.ClassSectionsSubjectTeacherDeliveryMode)
		}

		// 4a) SNAPSHOT GURU
		if ts, err := teacherSnapshot.BuildTeacherSnapshot(c.Context(), tx, masjidID, req.ClassSectionSubjectTeacherTeacherID); err != nil {
			fmt.Println("[CSST.Create] teacher snapshot ERROR:", err)
			switch {
			case errors.Is(err, gorm.ErrRecordNotFound):
				return helper.JsonError(c, fiber.StatusBadRequest, "Guru tidak valid / sudah dihapus")
			case errors.Is(err, teacherSnapshot.ErrMasjidMismatch):
				return helper.JsonError(c, fiber.StatusForbidden, "Guru bukan milik masjid Anda")
			default:
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat snapshot guru")
			}
		} else if ts != nil {
			if jb, e := teacherSnapshot.ToJSONB(ts); e == nil {
				row.ClassSectionSubjectTeacherTeacherSnapshot = &jb
				fmt.Println("[CSST.Create] teacher snapshot OK")
			}
		}

		// 4b) SNAPSHOT ASISTEN (opsional)
		if req.ClassSectionSubjectTeacherAssistantTeacherID != nil {
			if ats, err := teacherSnapshot.BuildTeacherSnapshot(c.Context(), tx, masjidID, *req.ClassSectionSubjectTeacherAssistantTeacherID); err != nil {
				fmt.Println("[CSST.Create] assistant snapshot ERROR:", err)
				switch {
				case errors.Is(err, gorm.ErrRecordNotFound):
					return helper.JsonError(c, fiber.StatusBadRequest, "Asisten guru tidak valid / sudah dihapus")
				case errors.Is(err, teacherSnapshot.ErrMasjidMismatch):
					return helper.JsonError(c, fiber.StatusForbidden, "Asisten guru bukan milik masjid Anda")
				default:
					return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat snapshot asisten guru")
				}
			} else if ats != nil {
				if jb, e := teacherSnapshot.ToJSONB(ats); e == nil {
					row.ClassSectionSubjectTeacherAssistantTeacherSnapshot = &jb
					fmt.Println("[CSST.Create] assistant snapshot OK")
				}
			}
		}

		// 4c) SNAPSHOT CLASS_SUBJECT (BARU)
		if j, err := snapshotClassSubject.BuildClassSubjectSnapshotJSON(c.Context(), tx, masjidID, req.ClassSectionSubjectTeacherClassSubjectID); err != nil {
			fmt.Println("[CSST.Create] class_subject snapshot ERROR:", err)
			switch {
			case errors.Is(err, gorm.ErrRecordNotFound):
				return helper.JsonError(c, fiber.StatusBadRequest, "Class subject tidak ditemukan / sudah dihapus")
			case errors.Is(err, snapshotClassSubject.ErrMasjidMismatch):
				return helper.JsonError(c, fiber.StatusForbidden, "Class subject milik masjid lain")
			default:
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat snapshot class subject")
			}
		} else {
			row.ClassSectionSubjectTeacherClassSubjectSnapshot = &j
			fmt.Println("[CSST.Create] class_subject snapshot OK")
		}

		// 5) SLUG → unique + NAME auto
		if row.ClassSectionSubjectTeacherSlug != nil {
			s := helper.Slugify(*row.ClassSectionSubjectTeacherSlug, 160)
			row.ClassSectionSubjectTeacherSlug = &s
			fmt.Println("[CSST.Create] input slug sanitized:", s)
		}

		base := strings.TrimSpace(getBaseForSlug(c.Context(), tx, masjidID,
			req.ClassSectionSubjectTeacherSectionID,
			req.ClassSectionSubjectTeacherClassSubjectID,
			req.ClassSectionSubjectTeacherTeacherID,
		))
		fmt.Println("[CSST.Create] slug base:", base)

		candidate := base
		if row.ClassSectionSubjectTeacherSlug != nil && strings.TrimSpace(*row.ClassSectionSubjectTeacherSlug) != "" {
			candidate = *row.ClassSectionSubjectTeacherSlug
		}
		candidate = helper.Slugify(candidate, 160)
		fmt.Println("[CSST.Create] slug candidate:", candidate)

		uniqueSlug, err := ensureUniqueSlug(c.Context(), tx, masjidID, candidate)
		if err != nil {
			fmt.Println("[CSST.Create] ensureUniqueSlug ERROR:", err)
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
		}
		row.ClassSectionSubjectTeacherSlug = &uniqueSlug
		row.ClassSectionSubjectTeacherName = deriveNameFromSlug(uniqueSlug)
		fmt.Printf("[CSST.Create] slug unique=%s | name=%s\n", uniqueSlug, row.ClassSectionSubjectTeacherName)

		// 6) INSERT
		fmt.Printf("[CSST.Create] INSERT row: masjid=%s section=%s subject=%s teacher=%s room=%v slug=%s name=%s mode=%s active=%v\n",
			row.ClassSectionSubjectTeacherMasjidID,
			row.ClassSectionSubjectTeacherSectionID,
			row.ClassSectionSubjectTeacherClassSubjectID,
			row.ClassSectionSubjectTeacherTeacherID,
			func() any {
				if row.ClassSectionSubjectTeacherRoomID == nil {
					return "<nil>"
				}
				return *row.ClassSectionSubjectTeacherRoomID
			}(),
			func() string {
				if row.ClassSectionSubjectTeacherSlug == nil {
					return "<nil>"
				}
				return *row.ClassSectionSubjectTeacherSlug
			}(),
			row.ClassSectionSubjectTeacherName,
			string(row.ClassSectionsSubjectTeacherDeliveryMode),
			row.ClassSectionSubjectTeacherIsActive,
		)

		if err := tx.Create(&row).Error; err != nil {
			fmt.Println("[CSST.Create] INSERT ERROR:", err)
			msg := strings.ToLower(err.Error())
			switch {
			case strings.Contains(msg, "uq_csst_one_active_per_section_subject_alive"),
				strings.Contains(msg, "uq_csst_unique_alive"),
				strings.Contains(msg, "duplicate"),
				strings.Contains(msg, "unique"):
				return helper.JsonError(c, fiber.StatusConflict, "Penugasan atau slug sudah terdaftar (duplikat).")
			case strings.Contains(msg, "uq_csst_slug_per_tenant_alive"):
				return helper.JsonError(c, fiber.StatusConflict, "Slug sudah digunakan pada tenant ini.")
			case strings.Contains(msg, "23503"), strings.Contains(msg, "foreign key"):
				switch {
				case strings.Contains(msg, "class_sections"):
					return helper.JsonError(c, fiber.StatusBadRequest, "FK gagal (SECTION): section tidak ditemukan / beda tenant")
				case strings.Contains(msg, "class_subjects"):
					return helper.JsonError(c, fiber.StatusBadRequest, "FK gagal (CLASS_SUBJECTS): tidak ditemukan / beda tenant")
				case strings.Contains(msg, "masjid_teachers"):
					return helper.JsonError(c, fiber.StatusBadRequest, "FK gagal (GURU): guru tidak ditemukan / beda tenant")
				case strings.Contains(msg, "class_rooms"):
					return helper.JsonError(c, fiber.StatusBadRequest, "FK gagal (ROOM): ruangan tidak ditemukan / beda tenant")
				default:
					return helper.JsonError(c, fiber.StatusBadRequest, "FK gagal: pastikan section/class_subjects/guru/room valid")
				}
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Insert gagal: "+err.Error())
		}

		fmt.Println("[CSST.Create] INSERT OK")
		return helper.JsonCreated(c, "Penugasan guru berhasil dibuat", dto.FromClassSectionSubjectTeacherModel(row))
	})
}

/* ======================== Helpers (inline room) ======================== */

/*
===============================

	UPDATE (partial)
	PUT /admin/:masjid_id/class-section-subject-teachers/:id

===============================
*/
func (ctl *ClassSectionSubjectTeacherController) Update(c *fiber.Ctx) error {
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(http.StatusBadRequest, "ID tidak valid")
	}

	var req dto.UpdateClassSectionSubjectTeacherRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(http.StatusBadRequest, "Payload tidak valid")
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

		// tenant guard
		if row.ClassSectionSubjectTeacherMasjidID != masjidID {
			return helper.JsonError(c, http.StatusForbidden, "Akses ditolak")
		}

		// (opsional) precheck konsistensi jika section_id / class_subject_id berubah
		if req.ClassSectionSubjectTeacherSectionID != nil || req.ClassSectionSubjectTeacherClassSubjectID != nil {
			sectionID := row.ClassSectionSubjectTeacherSectionID
			if req.ClassSectionSubjectTeacherSectionID != nil {
				sectionID = *req.ClassSectionSubjectTeacherSectionID
			}
			classSubjectID := row.ClassSectionSubjectTeacherClassSubjectID
			if req.ClassSectionSubjectTeacherClassSubjectID != nil {
				classSubjectID = *req.ClassSectionSubjectTeacherClassSubjectID
			}

			// cek section milik tenant + ambil class_id dari section
			var sec modelClassSection.ClassSectionModel
			if err := tx.
				Where("class_sections_id = ? AND class_sections_masjid_id = ? AND class_sections_deleted_at IS NULL", sectionID, masjidID).
				First(&sec).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return helper.JsonError(c, fiber.StatusBadRequest, "Section tidak ditemukan / beda tenant")
				}
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal cek section")
			}

			// cek class_subjects cocok
			var cs struct {
				ClassSubjectsMasjidID uuid.UUID `gorm:"column:class_subjects_masjid_id"`
				ClassSubjectsClassID  uuid.UUID `gorm:"column:class_subjects_class_id"`
			}
			if err := tx.
				Table("class_subjects").
				Select("class_subjects_masjid_id, class_subjects_class_id").
				Where("class_subjects_id = ? AND class_subjects_deleted_at IS NULL", classSubjectID).
				Take(&cs).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return helper.JsonError(c, fiber.StatusBadRequest, "class_subjects tidak ditemukan / sudah dihapus")
				}
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal cek class_subjects")
			}
			if cs.ClassSubjectsMasjidID != masjidID {
				return helper.JsonError(c, fiber.StatusBadRequest, "Masjid mismatch: class_subjects milik masjid lain")
			}
			if cs.ClassSubjectsClassID != sec.ClassSectionClassID {
				return helper.JsonError(c, fiber.StatusBadRequest, "Class mismatch: class_subjects.class_id != class_sections.class_id")
			}
		}

		// Catat apakah ada perubahan yang memengaruhi slug
		sectionChanged := req.ClassSectionSubjectTeacherSectionID != nil &&
			*req.ClassSectionSubjectTeacherSectionID != row.ClassSectionSubjectTeacherSectionID
		csChanged := req.ClassSectionSubjectTeacherClassSubjectID != nil &&
			*req.ClassSectionSubjectTeacherClassSubjectID != row.ClassSectionSubjectTeacherClassSubjectID
		teacherChanged := req.ClassSectionSubjectTeacherTeacherID != nil &&
			*req.ClassSectionSubjectTeacherTeacherID != row.ClassSectionSubjectTeacherTeacherID

		// Apply perubahan lain (belum sentuh slug)
		req.Apply(&row)

		// ===== SLUG handling =====
		// Normalisasi slug jika user mengirimkan
		if req.ClassSectionSubjectTeacherSlug != nil {
			if s := strings.TrimSpace(*req.ClassSectionSubjectTeacherSlug); s != "" {
				norm := helper.Slugify(s, 160)
				row.ClassSectionSubjectTeacherSlug = &norm
			} else {
				// "" → nil
				row.ClassSectionSubjectTeacherSlug = nil
			}
		}

		// Perlu generate/cek unik jika:
		// - user set slug baru (di-normalisasi di atas), atau
		// - slug masih kosong dan ada perubahan section/class_subject/teacher (atau awalnya kosong)
		needEnsureUnique := false
		baseSlug := ""

		if req.ClassSectionSubjectTeacherSlug != nil {
			needEnsureUnique = true
			if row.ClassSectionSubjectTeacherSlug != nil {
				baseSlug = *row.ClassSectionSubjectTeacherSlug
			}
		} else if row.ClassSectionSubjectTeacherSlug == nil || strings.TrimSpace(ptrStr(row.ClassSectionSubjectTeacherSlug)) == "" {
			if sectionChanged || csChanged || teacherChanged || row.ClassSectionSubjectTeacherSlug == nil {
				needEnsureUnique = true

				// Rakitan base dari entity terkait (best-effort)
				var sectionName, className, subjectName, teacherName string

				_ = tx.Table("class_sections").
					Select("class_sections_name").
					Where("class_sections_id = ? AND class_sections_masjid_id = ?", row.ClassSectionSubjectTeacherSectionID, masjidID).
					Scan(&sectionName).Error

				_ = tx.Table("class_subjects cs").
					Select("c.classes_name, s.subjects_name").
					Joins("JOIN classes c ON c.classes_id = cs.class_subjects_class_id AND c.classes_deleted_at IS NULL").
					Joins("JOIN subjects s ON s.subjects_id = cs.class_subjects_subject_id AND s.subjects_deleted_at IS NULL").
					Where("cs.class_subjects_id = ? AND cs.class_subjects_masjid_id = ?", row.ClassSectionSubjectTeacherClassSubjectID, masjidID).
					Scan(&struct {
						ClassesName  *string
						SubjectsName *string
					}{&className, &subjectName}).Error

				_ = tx.Table("masjid_teachers").
					Select("masjid_teacher_name").
					Where("masjid_teacher_id = ? AND masjid_teacher_masjid_id = ?", row.ClassSectionSubjectTeacherTeacherID, masjidID).
					Scan(&teacherName).Error

				parts := []string{}
				if strings.TrimSpace(className) != "" {
					parts = append(parts, className)
				}
				if strings.TrimSpace(sectionName) != "" {
					parts = append(parts, sectionName)
				}
				if strings.TrimSpace(subjectName) != "" {
					parts = append(parts, subjectName)
				}
				if strings.TrimSpace(teacherName) != "" {
					parts = append(parts, teacherName)
				}

				if len(parts) == 0 {
					baseSlug = fmt.Sprintf("csst-%s-%s-%s",
						strings.Split(row.ClassSectionSubjectTeacherSectionID.String(), "-")[0],
						strings.Split(row.ClassSectionSubjectTeacherClassSubjectID.String(), "-")[0],
						strings.Split(row.ClassSectionSubjectTeacherTeacherID.String(), "-")[0],
					)
				} else {
					baseSlug = strings.Join(parts, " ")
				}
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
					// Unik per tenant, soft-delete aware, EXCLUDE diri sendiri
					return q.Where(`
						class_section_subject_teacher_masjid_id = ?
						AND class_section_subject_teacher_deleted_at IS NULL
						AND class_section_subject_teacher_id <> ?
					`, masjidID, row.ClassSectionSubjectTeacherID)
				},
				160,
			)
			if err != nil {
				return helper.JsonError(c, http.StatusInternalServerError, "Gagal menghasilkan slug unik")
			}
			// Jika baseSlug kosong (mis. user set slug "" → nil), uniqueSlug akan dibuat dari fallback di helper.
			if strings.TrimSpace(uniqueSlug) != "" {
				row.ClassSectionSubjectTeacherSlug = &uniqueSlug
			} else {
				row.ClassSectionSubjectTeacherSlug = nil
			}
		}
		// ===== END SLUG =====

		// Persist
		if err := tx.Save(&row).Error; err != nil {
			msg := strings.ToLower(err.Error())
			if strings.Contains(msg, "uq_csst_one_active_per_section_subject_alive") ||
				strings.Contains(msg, "uq_csst_unique_alive") ||
				strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique") {
				// bisa karena duplikat kombinasi assignment, atau slug bentrok
				// cek slug index spesifik
				if strings.Contains(msg, "uq_csst_slug_per_tenant_alive") {
					return helper.JsonError(c, fiber.StatusConflict, "Slug sudah digunakan pada tenant ini.")
				}
				return helper.JsonError(c, fiber.StatusConflict, "Penugasan guru untuk class_subjects ini sudah aktif (duplikat).")
			}
			if strings.Contains(msg, "sqlstate 23503") || strings.Contains(msg, "foreign key") {
				switch {
				case strings.Contains(msg, "class_sections"):
					return helper.JsonError(c, fiber.StatusBadRequest, "FK gagal (SECTION): section tidak ditemukan / beda tenant")
				case strings.Contains(msg, "class_subjects"):
					return helper.JsonError(c, fiber.StatusBadRequest, "FK gagal (CLASS_SUBJECTS): tidak ditemukan / beda tenant")
				case strings.Contains(msg, "masjid_teachers"):
					return helper.JsonError(c, fiber.StatusBadRequest, "FK gagal (GURU): guru tidak ditemukan / beda tenant")
				case strings.Contains(msg, "class_rooms"):
					return helper.JsonError(c, fiber.StatusBadRequest, "FK gagal (ROOM): ruangan tidak ditemukan / beda tenant")
				default:
					return helper.JsonError(c, fiber.StatusBadRequest, "FK gagal: pastikan section/class_subjects/guru/room valid")
				}
			}
			return helper.JsonError(c, http.StatusInternalServerError, err.Error())
		}

		return helper.JsonUpdated(c, "Penugasan guru berhasil diperbarui", dto.FromClassSectionSubjectTeacherModel(row))
	})
}

/*
===============================

	DELETE (soft delete)
	DELETE /admin/:masjid_id/class-section-subject-teachers/:id

===============================
*/
func (ctl *ClassSectionSubjectTeacherController) Delete(c *fiber.Ctx) error {
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
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
	if row.ClassSectionSubjectTeacherMasjidID != masjidID {
		return helper.JsonError(c, http.StatusForbidden, "Akses ditolak")
	}
	if row.ClassSectionSubjectTeacherDeletedAt.Valid {
		// idempotent
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
