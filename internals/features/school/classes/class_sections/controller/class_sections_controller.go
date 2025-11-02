// file: internals/features/lembaga/classes/sections/main/controller/class_section_controller.go
package controller

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"mime/multipart"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	helper "schoolku_backend/internals/helpers"
	helperAuth "schoolku_backend/internals/helpers/auth"
	helperOSS "schoolku_backend/internals/helpers/oss"

	semstats "schoolku_backend/internals/features/lembaga/stats/semester_stats/service"
	secDTO "schoolku_backend/internals/features/school/classes/class_sections/dto"
	secModel "schoolku_backend/internals/features/school/classes/class_sections/model"
	classModel "schoolku_backend/internals/features/school/classes/classes/model"

	teacherSnapshot "schoolku_backend/internals/features/users/user_teachers/snapshot"

	roomSnapshot "schoolku_backend/internals/features/school/academics/rooms/snapshot"
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

type ClassParentAndTermSnapshot struct {
	SchoolID uuid.UUID

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

type LeaderStudentSnapshot struct {
	ID          uuid.UUID `json:"id"` // user_id
	Name        string    `json:"name"`
	WhatsappURL *string   `json:"whatsapp_url,omitempty"`
	AvatarURL   *string   `json:"avatar_url,omitempty"`
	// opsional jika ingin disimpan:
	// ParentWhatsappURL *string `json:"parent_whatsapp_url,omitempty"`
}

func validateAndSnapshotLeaderStudent(
	tx *gorm.DB,
	schoolID uuid.UUID,
	leaderSchoolStudentID uuid.UUID,
) (*LeaderStudentSnapshot, error) {
	log.Printf("[SECTIONS][CREATE] 🔎 validating leader_student=%s", leaderSchoolStudentID)

	var row struct {
		SchoolID    uuid.UUID
		UserID      uuid.UUID
		FullName    *string
		WhatsappURL *string
		AvatarURL   *string
		ParentWA    *string
	}

	// Pastikan ketua adalah siswa milik school ini (tenant-safe)
	if err := tx.Raw(`
		SELECT
			ms.school_student_school_id             AS school_id,
			up.user_profile_user_id                 AS user_id,
			COALESCE(up.user_profile_full_name_snapshot, '') AS full_name,
			up.user_profile_whatsapp_url            AS whatsapp_url,
			up.user_profile_avatar_url              AS avatar_url,
			up.user_profile_parent_whatsapp_url     AS parent_wa
		FROM school_students ms
		JOIN user_profiles up
		  ON up.user_profile_user_id = ms.school_student_user_id
		WHERE ms.school_student_id = ?
		  AND ms.school_student_deleted_at IS NULL
		  AND up.user_profile_deleted_at IS NULL
	`, leaderSchoolStudentID).Scan(&row).Error; err != nil {
		log.Printf("[SECTIONS][CREATE] ❌ leader validate db error: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi ketua kelas")
	}

	if row.SchoolID == uuid.Nil {
		log.Printf("[SECTIONS][CREATE] ❌ leader not found id=%s", leaderSchoolStudentID)
		return nil, fiber.NewError(fiber.StatusBadRequest, "Ketua kelas tidak ditemukan")
	}
	if row.SchoolID != schoolID {
		log.Printf("[SECTIONS][CREATE] ❌ leader school mismatch row=%s expect=%s", row.SchoolID, schoolID)
		return nil, fiber.NewError(fiber.StatusForbidden, "Ketua kelas bukan milik school Anda")
	}

	trim := func(p *string) *string {
		if p == nil {
			return nil
		}
		v := strings.TrimSpace(*p)
		if v == "" {
			return nil
		}
		return &v
	}

	name := ""
	if row.FullName != nil {
		name = strings.TrimSpace(*row.FullName)
	}
	if name == "" {
		name = "Siswa"
	}

	// Pilih WA siswa; kalau kosong, bisa pakai WA orang tua (kalau mau)
	wa := trim(row.WhatsappURL)
	if wa == nil {
		wa = trim(row.ParentWA)
	}

	log.Printf("[SECTIONS][CREATE] ✅ leader snapshot='%s'", name)

	return &LeaderStudentSnapshot{
		ID:          row.UserID, // simpan user_id sebagai ID
		Name:        name,
		WhatsappURL: wa,
		AvatarURL:   trim(row.AvatarURL),
	}, nil
}

func snapshotClassParentAndTerm(
	tx *gorm.DB,
	schoolID uuid.UUID,
	classID uuid.UUID,
) (*ClassParentAndTermSnapshot, error) {
	log.Printf("[SECTIONS][SNAP] 🔎 class->parent snapshot class_id=%s", classID)

	var row ClassParentAndTermSnapshot
	if err := tx.Raw(`
		SELECT
			c.class_school_id                         AS school_id,
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
		 AND cp.class_parent_school_id = c.class_school_id
		 AND cp.class_parent_deleted_at IS NULL
		LEFT JOIN academic_terms at
		  ON at.academic_term_id = c.class_term_id
		 AND at.academic_term_school_id = c.class_school_id
		 AND at.academic_term_deleted_at IS NULL
		WHERE c.class_id = ? AND c.class_deleted_at IS NULL
	`, classID).Scan(&row).Error; err != nil {
		log.Printf("[SECTIONS][SNAP] ❌ query error: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil snapshot class/parent/term")
	}

	if row.SchoolID == uuid.Nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Class tidak ditemukan")
	}
	if row.SchoolID != schoolID {
		return nil, fiber.NewError(fiber.StatusForbidden, "Class bukan milik school Anda")
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

// acak indeks [0..n)
func randIdx(n int) (int, error) {
	var b [1]byte
	if n <= 0 {
		return 0, fmt.Errorf("n must be > 0")
	}
	if _, err := rand.Read(b[:]); err != nil {
		return 0, err
	}
	return int(b[0]) % n, nil
}

func pickTwoDistinct(max int) (int, int, error) {
	i1, err := randIdx(max)
	if err != nil {
		return 0, 0, err
	}
	i2, err := randIdx(max)
	if err != nil {
		return 0, 0, err
	}
	for i2 == i1 {
		i2, err = randIdx(max)
		if err != nil {
			return 0, 0, err
		}
	}
	if i2 < i1 {
		i1, i2 = i2, i1
	}
	return i1, i2, nil
}

// Bentuk plaintext join-code: "<slug>-<partA><partB>"
func buildSectionJoinCode(slug string, id uuid.UUID) (string, error) {
	parts := strings.Split(id.String(), "-") // umumnya 5 bagian
	if len(parts) == 5 {
		i1, i2, err := pickTwoDistinct(5)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s-%s%s", slug, parts[i1], parts[i2]), nil
	}
	// fallback jika format UUID tidak standar
	s := strings.ReplaceAll(id.String(), "-", "")
	if len(s) >= 16 {
		return fmt.Sprintf("%s-%s%s", slug, s[:8], s[len(s)-8:]), nil
	}
	return fmt.Sprintf("%s-%s", slug, s), nil
}

func bcryptHash(s string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(s), bcrypt.DefaultCost)
}

//* Main Controller

// POST /admin/class-sections
// POST /admin/class-sections
func (ctrl *ClassSectionController) CreateClassSection(c *fiber.Ctx) error {
	log.Printf("[SECTIONS][CREATE] ▶️ incoming request")

	// ---- School context ----
	mc, err := helperAuth.ResolveSchoolContext(c)
	if err != nil {
		return err
	}
	var schoolID uuid.UUID
	switch {
	case mc.ID != uuid.Nil:
		schoolID = mc.ID
	case strings.TrimSpace(mc.Slug) != "":
		id, er := helperAuth.GetSchoolIDBySlug(c, strings.TrimSpace(mc.Slug))
		if er != nil {
			return helper.JsonError(c, fiber.StatusNotFound, "School (slug) tidak ditemukan")
		}
		schoolID = id
	default:
		id, er := helperAuth.GetSchoolIDFromTokenPreferTeacher(c)
		if er != nil || id == uuid.Nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "School context tidak ditemukan")
		}
		schoolID = id
	}
	if err := helperAuth.EnsureStaffSchool(c, schoolID); err != nil {
		return err
	}

	// ---- Parse req ----
	var req secDTO.ClassSectionCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	req.ClassSectionSchoolID = schoolID
	req.Normalize()

	// ---- Validasi ringan ----
	if strings.TrimSpace(req.ClassSectionName) == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Nama section wajib diisi")
	}
	if req.ClassSectionCapacity != nil && *req.ClassSectionCapacity < 0 {
		return fiber.NewError(fiber.StatusBadRequest, "Capacity tidak boleh negatif")
	}
	if req.ClassSectionCSSTMaxSubjectsPerStudent != nil && *req.ClassSectionCSSTMaxSubjectsPerStudent < 0 {
		return fiber.NewError(fiber.StatusBadRequest, "Batas maksimal mapel per siswa tidak boleh negatif")
	}
	if req.ClassSectionCSSTEnrollmentMode != nil && strings.TrimSpace(*req.ClassSectionCSSTEnrollmentMode) != "" {
		if !isValidEnrollmentMode(*req.ClassSectionCSSTEnrollmentMode) {
			return fiber.NewError(fiber.StatusBadRequest, "Mode enrolment CSST tidak valid (self_select | assigned | hybrid)")
		}
	}

	// ---- TX ----
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

	// ---- Validasi class belong-to school ----
	{
		var cls classModel.ClassModel
		if err := tx.Select("class_id, class_school_id").
			Where("class_id = ? AND class_deleted_at IS NULL", req.ClassSectionClassID).
			First(&cls).Error; err != nil {
			_ = tx.Rollback()
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusBadRequest, "Class tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi class")
		}
		if cls.ClassSchoolID != schoolID {
			_ = tx.Rollback()
			return fiber.NewError(fiber.StatusForbidden, "Class bukan milik school Anda")
		}
	}

	// ---- Map req -> model ----
	m := req.ToModel()
	m.ClassSectionSchoolID = schoolID

	// ---- Snapshot relasi (class→parent/term) ----
	if snap, err := snapshotClassParentAndTerm(tx, schoolID, req.ClassSectionClassID); err != nil {
		_ = tx.Rollback()
		return err
	} else {
		applyClassParentAndTermSnapshotToSection(m, snap)
	}

	// ---- Snapshot TEACHER / ASSISTANT ----
	if req.ClassSectionTeacherID != nil {
		ts, err := teacherSnapshot.BuildTeacherSnapshot(c.Context(), tx, schoolID, *req.ClassSectionTeacherID)
		if err != nil {
			_ = tx.Rollback()
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, fiber.StatusBadRequest, "Guru tidak ditemukan / sudah dihapus")
			}
			if fe, ok := err.(*fiber.Error); ok && fe.Code == fiber.StatusForbidden {
				return helper.JsonError(c, fiber.StatusForbidden, "Guru bukan milik school Anda")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal validasi guru")
		}
		if ts != nil {
			if jb, e := teacherSnapshot.ToJSONB(ts); e == nil {
				m.ClassSectionTeacherSnapshot = jb
			}
		}
	}
	if req.ClassSectionAssistantTeacherID != nil {
		as, err := teacherSnapshot.BuildTeacherSnapshot(c.Context(), tx, schoolID, *req.ClassSectionAssistantTeacherID)
		if err != nil {
			_ = tx.Rollback()
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, fiber.StatusBadRequest, "Asisten guru tidak valid / sudah dihapus")
			}
			if fe, ok := err.(*fiber.Error); ok && fe.Code == fiber.StatusForbidden {
				return helper.JsonError(c, fiber.StatusForbidden, "Asisten guru bukan milik school Anda")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal validasi asisten guru")
		}
		if as != nil {
			if jb, e := teacherSnapshot.ToJSONB(as); e == nil {
				m.ClassSectionAssistantTeacherSnapshot = jb
			}
		}
	}

	// ---- Snapshot LEADER STUDENT ----
	if req.ClassSectionLeaderStudentID != nil {
		snap, err := validateAndSnapshotLeaderStudent(tx, schoolID, *req.ClassSectionLeaderStudentID)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
		if b, e := json.Marshal(snap); e == nil {
			m.ClassSectionLeaderStudentSnapshot = datatypes.JSON(b)
		}
	}

	// ==== ⬇️ Snapshot ROOM (ikuti pola Update) ⬇️ ====
	if m.ClassSectionClassRoomID != nil {
		rs, err := roomSnapshot.ValidateAndSnapshotRoom(tx, schoolID, *m.ClassSectionClassRoomID)
		if err != nil {
			_ = tx.Rollback()
			if fe, ok := err.(*fiber.Error); ok {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal validasi ruang kelas")
		}
		roomSnapshot.ApplyRoomSnapshotToSection(m, rs)
	}
	// ==== ⬆️ Snapshot ROOM (ikuti pola Update) ⬆️ ====

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
	uniqueSlug, uErr := helper.EnsureUniqueSlugCI(
		c.Context(), tx,
		"class_sections", "class_section_slug",
		baseSlug,
		func(q *gorm.DB) *gorm.DB {
			return q.Where("class_section_school_id = ? AND class_section_deleted_at IS NULL", schoolID)
		},
		160,
	)
	if uErr != nil {
		_ = tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
	}
	m.ClassSectionSlug = uniqueSlug

	// ---- Generate join code (student/teacher) ----
	if m.ClassSectionID == uuid.Nil {
		m.ClassSectionID = uuid.New()
	}
	plainCode, err := buildSectionJoinCode(m.ClassSectionSlug, m.ClassSectionID)
	if err != nil {
		_ = tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membangun join code")
	}
	hashed, err := bcryptHash(plainCode)
	if err != nil {
		_ = tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal meng-hash join code")
	}
	now := time.Now()
	m.ClassSectionCode = &plainCode
	m.ClassSectionStudentCodeHash = hashed
	m.ClassSectionStudentCodeSetAt = &now

	tPlain, _ := buildSectionJoinCode(m.ClassSectionSlug+"-t", m.ClassSectionID)
	tHash, _ := bcryptHash(tPlain)
	m.ClassSectionTeacherCodeHash = tHash
	m.ClassSectionTeacherCodeSetAt = &now

	// ---- INSERT ----
	if err := tx.Create(m).Error; err != nil {
		_ = tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat section")
	}

	// (opsional) upload image, update stats, dll...

	if err := tx.Commit().Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return helper.JsonCreated(c, "Section berhasil dibuat", fiber.Map{
		"section":            secDTO.FromModelClassSection(m),
		"uploaded_image_url": "",
		"join_code_preview":  plainCode,
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

	// ---- Guard staff school ----
	if err := helperAuth.EnsureStaffSchool(c, existing.ClassSectionSchoolID); err != nil {
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
		var tSchool uuid.UUID
		if err := tx.Raw(`
             SELECT school_teacher_school_id
             FROM school_teachers
             WHERE school_teacher_id = ? AND school_teacher_deleted_at IS NULL
         `, *req.ClassSectionTeacherID.Value).Scan(&tSchool).Error; err != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal validasi pengajar")
		}
		if tSchool == uuid.Nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusBadRequest, "Pengajar tidak ditemukan")
		}
		if tSchool != existing.ClassSectionSchoolID {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusForbidden, "Pengajar bukan milik school Anda")
		}
	}

	// =========================================================
	// 🔹 SNAPSHOT ROOM: siapkan hasil validasi/snapshot bila field room dipatch
	// =========================================================
	var (
		roomSnapRequested bool
		roomSnap          *roomSnapshot.RoomSnapshot
	)
	if req.ClassSectionClassRoomID.Present {
		roomSnapRequested = true
		if req.ClassSectionClassRoomID.Value != nil {
			rs, err := roomSnapshot.ValidateAndSnapshotRoom(
				tx,
				existing.ClassSectionSchoolID,
				*req.ClassSectionClassRoomID.Value,
			)
			if err != nil {
				_ = tx.Rollback()
				var fe *fiber.Error
				if errors.As(err, &fe) {
					return helper.JsonError(c, fe.Code, fe.Message)
				}
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal validasi ruang kelas")
			}
			roomSnap = rs
		} else {
			// dipatch NULL → clear snapshot
			roomSnap = nil
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
					"class_section_school_id = ? AND class_section_id <> ? AND class_section_deleted_at IS NULL",
					existing.ClassSectionSchoolID, existing.ClassSectionID,
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
					"class_section_school_id = ? AND class_section_id <> ? AND class_section_deleted_at IS NULL",
					existing.ClassSectionSchoolID, existing.ClassSectionID,
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

	// =========================================================
	// 🔹 Apply perubahan model dasar dulu (room_id baru ikut terpasang)
	// =========================================================
	req.Apply(&existing)

	// =========================================================
	// 🔹 Apply room snapshot jika field-nya memang dipatch
	// =========================================================
	if roomSnapRequested {
		roomSnapshot.ApplyRoomSnapshotToSection(&existing, roomSnap)
	}

	// ---- Save ----
	if err := tx.Save(&existing).Error; err != nil {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui section")
	}

	// ---- Upload image (opsional, multipart) ----
	uploadedURL := ""
	if fh := pickImageFile(c, "image", "file"); fh != nil {
		log.Printf("[SECTIONS][PATCH] 📤 uploading image filename=%s size=%d", fh.Filename, fh.Size)
		if url, upErr := helperOSS.UploadImageToOSSScoped(existing.ClassSectionSchoolID, "classes/sections", fh); upErr == nil && strings.TrimSpace(url) != "" {
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
		if err := stats.EnsureForSchool(tx, existing.ClassSectionSchoolID); err != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal inisialisasi lembaga_stats: "+err.Error())
		}
		delta := -1
		if newActive {
			delta = +1
		}
		if err := stats.IncActiveSections(tx, existing.ClassSectionSchoolID, delta); err != nil {
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

	// Guard akses staff pada school terkait
	if err := helperAuth.EnsureStaffSchool(c, m.ClassSectionSchoolID); err != nil {
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
		if err := stats.EnsureForSchool(tx, m.ClassSectionSchoolID); err != nil {
			_ = tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal inisialisasi lembaga_stats: "+err.Error())
		}
		if err := stats.IncActiveSections(tx, m.ClassSectionSchoolID, -1); err != nil {
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
