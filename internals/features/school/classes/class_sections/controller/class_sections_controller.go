// file: internals/features/lembaga/classes/sections/main/controller/class_section_controller.go
package controller

import (
	"encoding/json"
	"errors"
	"log"
	"mime/multipart"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
	helperOSS "masjidku_backend/internals/helpers/oss"

	semstats "masjidku_backend/internals/features/lembaga/stats/semester_stats/service"
	secDTO "masjidku_backend/internals/features/school/classes/class_sections/dto"
	secModel "masjidku_backend/internals/features/school/classes/class_sections/model"
	classModel "masjidku_backend/internals/features/school/classes/classes/model"
)

type ClassSectionController struct {
	DB *gorm.DB
}

func NewClassSectionController(db *gorm.DB) *ClassSectionController {
	return &ClassSectionController{DB: db}
}

/* ================= Handlers (ADMIN) ================= */

// enum valid utk enrollment mode (sinkron dg SQL enum class_section_csst_enrollment_mode)
var validEnrollmentModes = map[string]struct{}{
	"self_select": {},
	"assigned":    {},
	"hybrid":      {},
}

func isValidEnrollmentMode(s string) bool {
	_, ok := validEnrollmentModes[strings.ToLower(strings.TrimSpace(s))]
	return ok
}

// helper untuk validasi & snapshot teacher/assistant
type TeacherSnapshot struct {
	Name      string
	Phone     *string
	AvatarURL *string
}

func validateAndSnapshotTeacher(
	tx *gorm.DB,
	masjidID uuid.UUID,
	teacherID uuid.UUID,
	role string, // "teacher" / "assistant_teacher"
) (*TeacherSnapshot, error) {
	log.Printf("[SECTIONS][CREATE] 🔎 validating %s=%s", role, teacherID)

	var row struct {
		MasjidID  uuid.UUID
		FullName  string
		Phone     *string
		AvatarURL *string
	}

	if err := tx.Raw(`
		SELECT mt.masjid_teacher_masjid_id AS masjid_id,
		       ut.user_teacher_name         AS full_name,
		       ut.user_teacher_whatsapp_url AS phone,
		       ut.user_teacher_avatar_url   AS avatar_url
		FROM masjid_teachers mt
		JOIN user_teachers ut
		  ON ut.user_teacher_id = mt.masjid_teacher_user_teacher_id
		WHERE mt.masjid_teacher_id = ? AND mt.masjid_teacher_deleted_at IS NULL
	`, teacherID).Scan(&row).Error; err != nil {
		log.Printf("[SECTIONS][CREATE] ❌ %s validate db error: %v", role, err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi "+role)
	}

	if row.MasjidID == uuid.Nil {
		log.Printf("[SECTIONS][CREATE] ❌ %s not found id=%s", role, teacherID)
		return nil, fiber.NewError(fiber.StatusBadRequest, role+" tidak ditemukan")
	}

	if row.MasjidID != masjidID {
		log.Printf("[SECTIONS][CREATE] ❌ %s masjid mismatch row=%s expect=%s", role, row.MasjidID, masjidID)
		return nil, fiber.NewError(fiber.StatusForbidden, role+" bukan milik masjid Anda")
	}

	log.Printf("[SECTIONS][CREATE] ✅ %s snapshot='%s'", role, row.FullName)
	return &TeacherSnapshot{
		Name:      row.FullName,
		Phone:     row.Phone,
		AvatarURL: row.AvatarURL,
	}, nil
}

type ClassParentAndTermSnapshot struct {
	MasjidID uuid.UUID

	// class
	ClassSlug string

	// parent
	ParentName  string
	ParentCode  *string
	ParentSlug  *string
	ParentLevel *int16

	// term
	TermID   *uuid.UUID
	TermName *string
	TermSlug *string
	TermYear *string
}

func snapshotClassParentAndTerm(
	tx *gorm.DB,
	masjidID uuid.UUID,
	classID uuid.UUID,
) (*ClassParentAndTermSnapshot, error) {
	log.Printf("[SECTIONS][SNAP] 🔎 class->parent snapshot class_id=%s", classID)

	var row ClassParentAndTermSnapshot
	if err := tx.Raw(`
		SELECT
			c.class_masjid_id                         AS masjid_id,
			c.class_slug                              AS class_slug,

			cp.class_parent_name                      AS parent_name,
			cp.class_parent_code                      AS parent_code,
			cp.class_parent_slug                      AS parent_slug,
			cp.class_parent_level                     AS parent_level,

			c.class_term_id                           AS term_id,
			at.academic_term_name                     AS term_name,
			at.academic_term_slug                     AS term_slug,
			at.academic_term_academic_year            AS term_year
		FROM classes c
		JOIN class_parents cp
		  ON cp.class_parent_id = c.class_parent_id
		 AND cp.class_parent_masjid_id = c.class_masjid_id
		 AND cp.class_parent_deleted_at IS NULL
		LEFT JOIN academic_terms at
		  ON at.academic_term_id = c.class_term_id
		 AND at.academic_term_masjid_id = c.class_masjid_id
		 AND at.academic_term_deleted_at IS NULL
		WHERE c.class_id = ? AND c.class_deleted_at IS NULL
	`, classID).Scan(&row).Error; err != nil {
		log.Printf("[SECTIONS][SNAP] ❌ query error: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil snapshot class/parent/term")
	}

	if row.MasjidID == uuid.Nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Class tidak ditemukan")
	}
	if row.MasjidID != masjidID {
		return nil, fiber.NewError(fiber.StatusForbidden, "Class bukan milik masjid Anda")
	}

	// Normalisasi ringan
	row.ClassSlug = strings.TrimSpace(row.ClassSlug)
	row.ParentName = strings.TrimSpace(row.ParentName)
	if row.ParentCode != nil {
		v := strings.TrimSpace(*row.ParentCode)
		row.ParentCode = strPtrNZ(v)
	}
	if row.ParentSlug != nil {
		v := strings.TrimSpace(*row.ParentSlug)
		row.ParentSlug = strPtrNZ(v)
	}
	if row.TermName != nil {
		v := strings.TrimSpace(*row.TermName)
		row.TermName = strPtrNZ(v)
	}
	if row.TermSlug != nil {
		v := strings.TrimSpace(*row.TermSlug)
		row.TermSlug = strPtrNZ(v)
	}
	if row.TermYear != nil {
		v := strings.TrimSpace(*row.TermYear)
		row.TermYear = strPtrNZ(v)
	}

	return &row, nil
}

func applyClassParentAndTermSnapshotToSection(mcs *secModel.ClassSectionModel, s *ClassParentAndTermSnapshot) {
	// ---------- CLASS SNAPSHOT ----------
	if slug := strings.TrimSpace(s.ClassSlug); slug != "" {
		classSnap := map[string]any{
			"slug": slug,
		}
		if b, err := json.Marshal(classSnap); err == nil {
			mcs.ClassSectionClassSnapshot = datatypes.JSON(b)
		}
	} else {
		// kosongkan jika memang tidak ada
		mcs.ClassSectionClassSnapshot = datatypes.JSON([]byte("null"))
	}

	// ---------- PARENT SNAPSHOT ----------
	parentSnap := map[string]any{}
	if name := strings.TrimSpace(s.ParentName); name != "" {
		parentSnap["name"] = name
	}
	if s.ParentCode != nil && strings.TrimSpace(*s.ParentCode) != "" {
		parentSnap["code"] = strings.TrimSpace(*s.ParentCode)
	}
	if s.ParentSlug != nil && strings.TrimSpace(*s.ParentSlug) != "" {
		parentSnap["slug"] = strings.TrimSpace(*s.ParentSlug)
	}
	// level di schema SQL disimpan di snapshot -> 'level' (string),
	// kolom generated `class_section_parent_level_snap` membaca dari ->>'level'
	if s.ParentLevel != nil {
		parentSnap["level"] = strconv.FormatInt(int64(*s.ParentLevel), 10)
	}
	if len(parentSnap) > 0 {
		if b, err := json.Marshal(parentSnap); err == nil {
			mcs.ClassSectionParentSnapshot = datatypes.JSON(b)
		}
	} else {
		mcs.ClassSectionParentSnapshot = datatypes.JSON([]byte("null"))
	}

	// ---------- TERM SNAPSHOT ----------
	mcs.ClassSectionTermID = s.TermID
	termSnap := map[string]any{}
	if s.TermName != nil && strings.TrimSpace(*s.TermName) != "" {
		termSnap["name"] = strings.TrimSpace(*s.TermName)
	}
	if s.TermSlug != nil && strings.TrimSpace(*s.TermSlug) != "" {
		termSnap["slug"] = strings.TrimSpace(*s.TermSlug)
	}
	if s.TermYear != nil && strings.TrimSpace(*s.TermYear) != "" {
		termSnap["year_label"] = strings.TrimSpace(*s.TermYear)
	}
	if len(termSnap) > 0 {
		if b, err := json.Marshal(termSnap); err == nil {
			mcs.ClassSectionTermSnapshot = datatypes.JSON(b)
		}
	} else {
		mcs.ClassSectionTermSnapshot = datatypes.JSON([]byte("null"))
	}

	// housekeeping
	ts := time.Now()
	mcs.ClassSectionSnapshotUpdatedAt = &ts
}

func strPtrNZ(v string) *string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	x := v
	return &x
}

// POST /admin/class-sections
func (ctrl *ClassSectionController) CreateClassSection(c *fiber.Ctx) error {
	start := time.Now()
	log.Printf("[SECTIONS][CREATE] ▶️ incoming request")

	// ---- Masjid context ----
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		log.Printf("[SECTIONS][CREATE] ❌ resolve masjid ctx error: %v", err)
		return err
	}
	var masjidID uuid.UUID
	switch {
	case mc.ID != uuid.Nil:
		masjidID = mc.ID
		log.Printf("[SECTIONS][CREATE] 🕌 masjid_id from ctx=%s", masjidID)
	case strings.TrimSpace(mc.Slug) != "":
		id, er := helperAuth.GetMasjidIDBySlug(c, strings.TrimSpace(mc.Slug))
		if er != nil {
			log.Printf("[SECTIONS][CREATE] ❌ masjid slug(%s) not found: %v", mc.Slug, er)
			return helper.JsonError(c, fiber.StatusNotFound, "Masjid (slug) tidak ditemukan")
		}
		masjidID = id
		log.Printf("[SECTIONS][CREATE] 🕌 masjid_id from slug=%s → %s", mc.Slug, masjidID)
	default:
		id, er := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
		if er != nil || id == uuid.Nil {
			log.Printf("[SECTIONS][CREATE] ❌ masjid context not found via token: %v", er)
			return helper.JsonError(c, fiber.StatusBadRequest, "Masjid context tidak ditemukan")
		}
		masjidID = id
		log.Printf("[SECTIONS][CREATE] 🕌 masjid_id from token=%s", masjidID)
	}
	if err := helperAuth.EnsureStaffMasjid(c, masjidID); err != nil {
		log.Printf("[SECTIONS][CREATE] ❌ ensure staff masjid failed: %v", err)
		return err
	}

	// ---- Parse req ----
	var req secDTO.ClassSectionCreateRequest
	if err := c.BodyParser(&req); err != nil {
		log.Printf("[SECTIONS][CREATE] ❌ body parse error: %v", err)
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	req.ClassSectionMasjidID = masjidID
	req.Normalize()
	log.Printf("[SECTIONS][CREATE] 📩 req: class_id=%s teacher_id=%v assistant_id=%v room_id=%v name='%s' slug_in='%s'",
		req.ClassSectionClassID, req.ClassSectionTeacherID, req.ClassSectionAssistantTeacherID,
		req.ClassSectionClassRoomID, req.ClassSectionName, req.ClassSectionSlug)

	// ---- Sanity dasar ----
	if strings.TrimSpace(req.ClassSectionName) == "" {
		log.Printf("[SECTIONS][CREATE] ❌ name kosong")
		return fiber.NewError(fiber.StatusBadRequest, "Nama section wajib diisi")
	}
	if req.ClassSectionCapacity != nil && *req.ClassSectionCapacity < 0 {
		log.Printf("[SECTIONS][CREATE] ❌ capacity negatif=%d", *req.ClassSectionCapacity)
		return fiber.NewError(fiber.StatusBadRequest, "Capacity tidak boleh negatif")
	}

	// ---- Sanity tambahan (CSST settings & features) ----
	if req.ClassSectionCSSTMaxSubjectsPerStudent != nil && *req.ClassSectionCSSTMaxSubjectsPerStudent < 0 {
		log.Printf("[SECTIONS][CREATE] ❌ csst_max_subjects_per_student negatif=%d", *req.ClassSectionCSSTMaxSubjectsPerStudent)
		return fiber.NewError(fiber.StatusBadRequest, "Batas maksimal mapel per siswa tidak boleh negatif")
	}
	if req.ClassSectionCSSTEnrollmentMode != nil && strings.TrimSpace(*req.ClassSectionCSSTEnrollmentMode) != "" {
		if !isValidEnrollmentMode(*req.ClassSectionCSSTEnrollmentMode) {
			log.Printf("[SECTIONS][CREATE] ❌ enrollment_mode invalid=%s", *req.ClassSectionCSSTEnrollmentMode)
			return fiber.NewError(fiber.StatusBadRequest, "Mode enrolment CSST tidak valid (self_select | assigned | hybrid)")
		}
	}

	// ---- TX ----
	tx := ctrl.DB.WithContext(c.Context()).Begin()
	if tx.Error != nil {
		log.Printf("[SECTIONS][CREATE] ❌ begin tx error: %v", tx.Error)
		return fiber.NewError(fiber.StatusInternalServerError, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			log.Printf("[SECTIONS][CREATE] 💥 panic recovered: %+v", r)
			panic(r)
		}
	}()

	// ---- Validasi class se-masjid ----
	{
		var cls classModel.ClassModel
		if err := tx.Select("class_id, class_masjid_id").
			Where("class_id = ? AND class_deleted_at IS NULL", req.ClassSectionClassID).
			First(&cls).Error; err != nil {
			_ = tx.Rollback()
			if errors.Is(err, gorm.ErrRecordNotFound) {
				log.Printf("[SECTIONS][CREATE] ❌ class not found id=%s", req.ClassSectionClassID)
				return fiber.NewError(fiber.StatusBadRequest, "Class tidak ditemukan")
			}
			log.Printf("[SECTIONS][CREATE] ❌ class validate db error: %v", err)
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi class")
		}
		if cls.ClassMasjidID != masjidID {
			_ = tx.Rollback()
			log.Printf("[SECTIONS][CREATE] ❌ class masjid mismatch row=%s expect=%s", cls.ClassMasjidID, masjidID)
			return fiber.NewError(fiber.StatusForbidden, "Class bukan milik masjid Anda")
		}
		log.Printf("[SECTIONS][CREATE] ✅ class validated")
	}

	// ---- Map ke model ----
	m := req.ToModel()
	m.ClassSectionMasjidID = masjidID

	// ---- Snapshot class->parent (+term)
	if snap, err := snapshotClassParentAndTerm(tx, masjidID, req.ClassSectionClassID); err != nil {
		_ = tx.Rollback()
		return err
	} else {
		applyClassParentAndTermSnapshotToSection(m, snap)
	}
	// Teacher
	if req.ClassSectionTeacherID != nil {
		snap, err := validateAndSnapshotTeacher(tx, masjidID, *req.ClassSectionTeacherID, "teacher")
		if err != nil {
			_ = tx.Rollback()
			return err
		}
		teacherSnap := map[string]any{
			"name": snap.Name,
		}
		if snap.Phone != nil && strings.TrimSpace(*snap.Phone) != "" {
			teacherSnap["phone"] = strings.TrimSpace(*snap.Phone)
		}
		if snap.AvatarURL != nil && strings.TrimSpace(*snap.AvatarURL) != "" {
			teacherSnap["avatar_url"] = strings.TrimSpace(*snap.AvatarURL)
		}
		if b, e := json.Marshal(teacherSnap); e == nil {
			m.ClassSectionTeacherSnapshot = datatypes.JSON(b)
		}
	}

	// Assistant Teacher
	if req.ClassSectionAssistantTeacherID != nil {
		snap, err := validateAndSnapshotTeacher(tx, masjidID, *req.ClassSectionAssistantTeacherID, "assistant_teacher")
		if err != nil {
			_ = tx.Rollback()
			return err
		}
		asstSnap := map[string]any{
			"name": snap.Name,
		}
		if snap.Phone != nil && strings.TrimSpace(*snap.Phone) != "" {
			asstSnap["phone"] = strings.TrimSpace(*snap.Phone)
		}
		if snap.AvatarURL != nil && strings.TrimSpace(*snap.AvatarURL) != "" {
			asstSnap["avatar_url"] = strings.TrimSpace(*snap.AvatarURL)
		}
		if b, e := json.Marshal(asstSnap); e == nil {
			m.ClassSectionAssistantTeacherSnapshot = datatypes.JSON(b)
		}
	}

	// ---- Validasi room ----
	if req.ClassSectionClassRoomID != nil {
		log.Printf("[SECTIONS][CREATE] 🔎 validating room=%s", *req.ClassSectionClassRoomID)
		var rMasjid uuid.UUID
		if err := tx.Raw(`
			SELECT class_room_masjid_id
			FROM class_rooms
			WHERE class_room_id = ? AND class_room_deleted_at IS NULL
		`, *req.ClassSectionClassRoomID).Scan(&rMasjid).Error; err != nil {
			_ = tx.Rollback()
			log.Printf("[SECTIONS][CREATE] ❌ room validate db error: %v", err)
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi ruang kelas")
		}
		if rMasjid == uuid.Nil {
			_ = tx.Rollback()
			log.Printf("[SECTIONS][CREATE] ❌ room not found id=%s", *req.ClassSectionClassRoomID)
			return fiber.NewError(fiber.StatusBadRequest, "Ruang kelas tidak ditemukan")
		}
		if rMasjid != masjidID {
			_ = tx.Rollback()
			log.Printf("[SECTIONS][CREATE] ❌ room masjid mismatch row=%s expect=%s", rMasjid, masjidID)
			return fiber.NewError(fiber.StatusForbidden, "Ruang kelas bukan milik masjid Anda")
		}
		log.Printf("[SECTIONS][CREATE] ✅ room validated")
	}

	// ---- Slug unik ----
	var baseSlug string
	if s := strings.TrimSpace(req.ClassSectionSlug); s != "" {
		baseSlug = helper.Slugify(s, 160)
	} else {
		baseSlug = helper.Slugify(strings.TrimSpace(req.ClassSectionName), 160)
		if baseSlug == "" {
			baseSlug = "section"
		}
	}
	log.Printf("[SECTIONS][CREATE] 🧩 baseSlug='%s'", baseSlug)
	uniqueSlug, uErr := helper.EnsureUniqueSlugCI(
		c.Context(), tx,
		"class_sections", "class_section_slug",
		baseSlug,
		func(q *gorm.DB) *gorm.DB {
			return q.Where("class_section_masjid_id = ? AND class_section_deleted_at IS NULL", masjidID)
		},
		160,
	)
	if uErr != nil {
		_ = tx.Rollback()
		log.Printf("[SECTIONS][CREATE] ❌ ensure unique slug error: %v", uErr)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
	}
	m.ClassSectionSlug = uniqueSlug
	log.Printf("[SECTIONS][CREATE] ✅ unique_slug='%s'", uniqueSlug)

	// ---- Simpan ----
	if err := tx.Create(m).Error; err != nil {
		_ = tx.Rollback()
		log.Printf("[SECTIONS][CREATE] ❌ insert error: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat section")
	}
	log.Printf("[SECTIONS][CREATE] 💾 created section_id=%s", m.ClassSectionID)

	// ---- Optional upload image ----
	uploadedURL := ""
	if fh := pickImageFile(c, "image", "file"); fh != nil {
		log.Printf("[SECTIONS][CREATE] 📤 uploading image filename=%s size=%d", fh.Filename, fh.Size)
		if url, upErr := helperOSS.UploadImageToOSSScoped(masjidID, "classes/sections", fh); upErr == nil && strings.TrimSpace(url) != "" {
			uploadedURL = url
			objKey := ""
			if k, e := helperOSS.ExtractKeyFromPublicURL(uploadedURL); e == nil {
				objKey = k
			} else if k2, e2 := helperOSS.KeyFromPublicURL(uploadedURL); e2 == nil {
				objKey = k2
			}
			_ = tx.Table("class_sections").
				Where("class_section_id = ?", m.ClassSectionID).
				Updates(map[string]any{
					"class_section_image_url":        uploadedURL,
					"class_section_image_object_key": objKey,
				}).Error
			m.ClassSectionImageURL = &uploadedURL
			m.ClassSectionImageObjectKey = &objKey
			log.Printf("[SECTIONS][CREATE] ✅ image set url=%s key=%s", uploadedURL, objKey)
		} else {
			log.Printf("[SECTIONS][CREATE] ⚠️ upload image failed: %v", upErr)
		}
	}

	// ---- Update lembaga_stats bila active ----
	if m.ClassSectionIsActive {
		log.Printf("[SECTIONS][CREATE] 📊 updating lembaga_stats (active +1)")
		statsSvc := semstats.NewLembagaStatsService()
		if err := statsSvc.EnsureForMasjid(tx, masjidID); err != nil {
			_ = tx.Rollback()
			log.Printf("[SECTIONS][CREATE] ❌ ensure stats error: %v", err)
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal inisialisasi lembaga_stats: "+err.Error())
		}
		if err := statsSvc.IncActiveSections(tx, masjidID, +1); err != nil {
			_ = tx.Rollback()
			log.Printf("[SECTIONS][CREATE] ❌ inc active sections error: %v", err)
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal update lembaga_stats: "+err.Error())
		}
		log.Printf("[SECTIONS][CREATE] ✅ lembaga_stats updated (+1 active section)")
	}

	if err := tx.Commit().Error; err != nil {
		log.Printf("[SECTIONS][CREATE] ❌ commit error: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	log.Printf("[SECTIONS][CREATE] ✅ done in %s", time.Since(start))

	return helper.JsonCreated(c, "Section berhasil dibuat", fiber.Map{
		"section":            secDTO.FromModelClassSection(m),
		"uploaded_image_url": uploadedURL,
	})
}

// PATCH /admin/class-sections/:id   (PATCH semantics)
func (ctrl *ClassSectionController) UpdateClassSection(c *fiber.Ctx) error {
	// ---- Parse section ID ----
	sectionID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	// ---- Decode request (support JSON & multipart) ----
	var req secDTO.ClassSectionPatchRequest
	if err := secDTO.DecodePatchClassSectionFromRequest(c, &req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// ---- Begin TX ----
	tx := ctrl.DB.WithContext(c.Context()).Begin()
	if tx.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
	}()

	// ---- Ambil data existing (lock) ----
	var existing secModel.ClassSectionModel
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("class_section_id = ? AND class_section_deleted_at IS NULL", sectionID).
		First(&existing).Error; err != nil {
		_ = tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Section tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// ---- Guard staff masjid ----
	if err := helperAuth.EnsureStaffMasjid(c, existing.ClassSectionMasjidID); err != nil {
		_ = tx.Rollback()
		return err
	}

	// ---- Sanity check ringan ----
	if req.ClassSectionName.Present && req.ClassSectionName.Value != nil {
		if strings.TrimSpace(*req.ClassSectionName.Value) == "" {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusBadRequest, "Nama section wajib diisi")
		}
	}
	if req.ClassSectionCapacity.Present && req.ClassSectionCapacity.Value != nil && *req.ClassSectionCapacity.Value < 0 {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusBadRequest, "Capacity tidak boleh negatif")
	}

	// ---- Sanity tambahan (CSST settings) ----
	if req.ClassSectionCSSTMaxSubjectsPerStudent.Present && req.ClassSectionCSSTMaxSubjectsPerStudent.Value != nil {
		if *req.ClassSectionCSSTMaxSubjectsPerStudent.Value < 0 {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusBadRequest, "Batas maksimal mapel per siswa tidak boleh negatif")
		}
	}
	if req.ClassSectionCSSTEnrollmentMode.Present && req.ClassSectionCSSTEnrollmentMode.Value != nil {
		if !isValidEnrollmentMode(*req.ClassSectionCSSTEnrollmentMode.Value) {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusBadRequest, "Mode enrolment CSST tidak valid (self_select | assigned | hybrid)")
		}
	}

	// ---- Validasi teacher kalau diubah ----
	if req.ClassSectionTeacherID.Present && req.ClassSectionTeacherID.Value != nil {
		var tMasjid uuid.UUID
		if err := tx.Raw(`
			SELECT masjid_teacher_masjid_id
			FROM masjid_teachers
			WHERE masjid_teacher_id = ? AND masjid_teacher_deleted_at IS NULL
		`, *req.ClassSectionTeacherID.Value).Scan(&tMasjid).Error; err != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal validasi pengajar")
		}
		if tMasjid == uuid.Nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusBadRequest, "Pengajar tidak ditemukan")
		}
		if tMasjid != existing.ClassSectionMasjidID {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusForbidden, "Pengajar bukan milik masjid Anda")
		}
	}

	// ---- Validasi room kalau diubah ----
	if req.ClassSectionClassRoomID.Present && req.ClassSectionClassRoomID.Value != nil {
		var rMasjid uuid.UUID
		if err := tx.Raw(`
			SELECT class_room_masjid_id
			FROM class_rooms
			WHERE class_room_id = ? AND class_room_deleted_at IS NULL
		`, *req.ClassSectionClassRoomID.Value).Scan(&rMasjid).Error; err != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal validasi ruang kelas")
		}
		if rMasjid == uuid.Nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusBadRequest, "Ruang kelas tidak ditemukan")
		}
		if rMasjid != existing.ClassSectionMasjidID {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusForbidden, "Ruang kelas bukan milik masjid Anda")
		}
	}

	// ---- Slug handling (pola ClassParent) ----
	if req.ClassSectionSlug.Present && req.ClassSectionSlug.Value != nil {
		// jika slug dipatch
		base := helper.Slugify(strings.TrimSpace(*req.ClassSectionSlug.Value), 160)
		if base == "" {
			n := existing.ClassSectionName
			if req.ClassSectionName.Present && req.ClassSectionName.Value != nil {
				n = strings.TrimSpace(*req.ClassSectionName.Value)
			}
			base = helper.Slugify(n, 160)
			if base == "" {
				base = "section"
			}
		}
		uniq, e := helper.EnsureUniqueSlugCI(
			c.Context(), tx,
			"class_sections", "class_section_slug",
			base,
			func(q *gorm.DB) *gorm.DB {
				return q.Where(
					"class_section_masjid_id = ? AND class_section_id <> ? AND class_section_deleted_at IS NULL",
					existing.ClassSectionMasjidID, existing.ClassSectionID,
				)
			},
			160,
		)
		if e != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
		}
		req.ClassSectionSlug.Value = &uniq
	} else if req.ClassSectionName.Present && req.ClassSectionName.Value != nil {
		// slug tidak dipatch, tapi name berubah → generate slug baru yg unik
		base := helper.Slugify(strings.TrimSpace(*req.ClassSectionName.Value), 160)
		if base == "" {
			base = "section"
		}
		uniq, e := helper.EnsureUniqueSlugCI(
			c.Context(), tx,
			"class_sections", "class_section_slug",
			base,
			func(q *gorm.DB) *gorm.DB {
				return q.Where(
					"class_section_masjid_id = ? AND class_section_id <> ? AND class_section_deleted_at IS NULL",
					existing.ClassSectionMasjidID, existing.ClassSectionID,
				)
			},
			160,
		)
		if e != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
		}
		req.ClassSectionSlug.Present = true
		req.ClassSectionSlug.Value = &uniq
	}

	// ---- Track perubahan status aktif ----
	wasActive := existing.ClassSectionIsActive
	newActive := wasActive
	if req.ClassSectionIsActive.Present && req.ClassSectionIsActive.Value != nil {
		newActive = *req.ClassSectionIsActive.Value
	}

	// ---- Apply perubahan ----
	req.Apply(&existing)
	if err := tx.Save(&existing).Error; err != nil {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui section")
	}

	// ---- Upload image (opsional, multipart) ----
	uploadedURL := ""
	if fh := pickImageFile(c, "image", "file"); fh != nil {
		log.Printf("[SECTIONS][PATCH] 📤 uploading image filename=%s size=%d", fh.Filename, fh.Size)
		if url, upErr := helperOSS.UploadImageToOSSScoped(existing.ClassSectionMasjidID, "classes/sections", fh); upErr == nil && strings.TrimSpace(url) != "" {
			uploadedURL = url

			newObjKey := ""
			if k, e := helperOSS.ExtractKeyFromPublicURL(uploadedURL); e == nil {
				newObjKey = k
			} else if k2, e2 := helperOSS.KeyFromPublicURL(uploadedURL); e2 == nil {
				newObjKey = k2
			}

			// ambil lama
			var oldURL, oldObjKey string
			{
				type row struct {
					URL string `gorm:"column:class_section_image_url"`
					Key string `gorm:"column:class_section_image_object_key"`
				}
				var r row
				_ = tx.Table("class_sections").
					Select("class_section_image_url, class_section_image_object_key").
					Where("class_section_id = ?", existing.ClassSectionID).
					Take(&r).Error
				oldURL = strings.TrimSpace(r.URL)
				oldObjKey = strings.TrimSpace(r.Key)
			}

			// move lama ke spam
			movedURL := ""
			if oldURL != "" {
				if mv, mvErr := helperOSS.MoveToSpamByPublicURLENV(oldURL, 0); mvErr == nil {
					movedURL = mv
					if k, e := helperOSS.ExtractKeyFromPublicURL(movedURL); e == nil {
						oldObjKey = k
					} else if k2, e2 := helperOSS.KeyFromPublicURL(movedURL); e2 == nil {
						oldObjKey = k2
					}
				}
			}

			deletePendingUntil := time.Now().Add(30 * 24 * time.Hour)
			_ = tx.Table("class_sections").
				Where("class_section_id = ?", existing.ClassSectionID).
				Updates(map[string]any{
					"class_section_image_url":        uploadedURL,
					"class_section_image_object_key": newObjKey,
					"class_section_image_url_old": func() any {
						if movedURL == "" {
							return gorm.Expr("NULL")
						}
						return movedURL
					}(),
					"class_section_image_object_key_old": func() any {
						if oldObjKey == "" {
							return gorm.Expr("NULL")
						}
						return oldObjKey
					}(),
					"class_section_image_delete_pending_until": deletePendingUntil,
				}).Error
		}
	}

	// ---- Update stats kalau status aktif berubah ----
	if wasActive != newActive {
		stats := semstats.NewLembagaStatsService()
		if err := stats.EnsureForMasjid(tx, existing.ClassSectionMasjidID); err != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal inisialisasi lembaga_stats: "+err.Error())
		}
		delta := -1
		if newActive {
			delta = +1
		}
		if err := stats.IncActiveSections(tx, existing.ClassSectionMasjidID, delta); err != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal update lembaga_stats: "+err.Error())
		}
	}

	// ---- Commit dulu ----
	if err := tx.Commit().Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// ---- Re-fetch terbaru untuk response ----
	var updated secModel.ClassSectionModel
	if err := ctrl.DB.WithContext(c.Context()).
		Where("class_section_id = ?", sectionID).
		First(&updated).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data terbaru")
	}

	return helper.JsonUpdated(c, "Section berhasil diperbarui", secDTO.FromModelClassSection(&updated))
}

// DELETE /admin/class-sections/:id (soft delete)
func (ctrl *ClassSectionController) SoftDeleteClassSection(c *fiber.Ctx) error {
	sectionID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	tx := ctrl.DB.WithContext(c.Context()).Begin()
	if tx.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
	}()

	var m secModel.ClassSectionModel
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&m, "class_section_id = ? AND class_section_deleted_at IS NULL", sectionID).Error; err != nil {
		_ = tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Section tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// Guard akses staff pada masjid terkait
	if err := helperAuth.EnsureStaffMasjid(c, m.ClassSectionMasjidID); err != nil {
		_ = tx.Rollback()
		return err
	}

	wasActive := m.ClassSectionIsActive
	now := time.Now()

	if err := tx.Model(&secModel.ClassSectionModel{}).
		Where("class_section_id = ?", m.ClassSectionID).
		Updates(map[string]any{
			"class_section_deleted_at": now,
			"class_section_is_active":  false,
			"class_section_updated_at": now,
		}).Error; err != nil {
		_ = tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus section")
	}

	if wasActive {
		stats := semstats.NewLembagaStatsService()
		if err := stats.EnsureForMasjid(tx, m.ClassSectionMasjidID); err != nil {
			_ = tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal inisialisasi lembaga_stats: "+err.Error())
		}
		if err := stats.IncActiveSections(tx, m.ClassSectionMasjidID, -1); err != nil {
			_ = tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal update lembaga_stats: "+err.Error())
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonDeleted(c, "Section berhasil dihapus", fiber.Map{
		"class_section_id": m.ClassSectionID,
	})
}

func pickImageFile(c *fiber.Ctx, names ...string) *multipart.FileHeader {
	for _, n := range names {
		if fh, err := c.FormFile(n); err == nil && fh != nil && fh.Size > 0 {
			return fh
		}
	}
	return nil
}
