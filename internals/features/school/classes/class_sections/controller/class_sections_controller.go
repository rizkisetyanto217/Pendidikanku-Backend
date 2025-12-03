// file: internals/features/lembaga/classes/sections/main/controller/class_section_controller.go
package controller

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"mime/multipart"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"
	helperOSS "madinahsalam_backend/internals/helpers/oss"

	semstats "madinahsalam_backend/internals/features/lembaga/stats/semester_stats/service"
	csstModel "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/model"
	secDTO "madinahsalam_backend/internals/features/school/classes/class_sections/dto"
	secModel "madinahsalam_backend/internals/features/school/classes/class_sections/model"
	classModel "madinahsalam_backend/internals/features/school/classes/classes/model"

	// Cache (section room snapshot)
	sectionroomsnap "madinahsalam_backend/internals/features/school/classes/class_sections/service"

	// ✅ Cache guru versi baru (school_teachers)
	teachersnap "madinahsalam_backend/internals/features/lembaga/school_yayasans/teachers_students/service"
)

/* =========================================================
   Controller
========================================================= */

type ClassSectionController struct {
	DB *gorm.DB
}

func NewClassSectionController(db *gorm.DB) *ClassSectionController {
	return &ClassSectionController{DB: db}
}

/* =========================================================
   Enrollment mode (validasi ringan)
========================================================= */

var validEnrollmentModes = map[string]struct{}{
	"self_select": {},
	"assigned":    {},
	"hybrid":      {},
}

func isValidEnrollmentMode(s string) bool {
	_, ok := validEnrollmentModes[strings.ToLower(strings.TrimSpace(s))]
	return ok
}

/* =========================================================
   Cache: Class → Parent & Term
========================================================= */

type ClassParentAndTermCache struct {
	SchoolID uuid.UUID

	// class
	ClassID   uuid.UUID
	ClassName string
	ClassSlug string

	// parent
	ParentID    *uuid.UUID
	ParentName  string
	ParentSlug  *string
	ParentLevel *int16

	// term
	TermID       *uuid.UUID
	TermName     *string
	TermSlug     *string
	TermYear     *string
	TermAngkatan *int
}

func snapshotClassParentAndTerm(tx *gorm.DB, schoolID, classID uuid.UUID) (*ClassParentAndTermCache, error) {
	// deteksi tabel parent (class_parents vs class_parent)
	parentTbl := "class_parents"
	{
		var r *string
		_ = tx.Raw(`SELECT to_regclass('class_parents')::text`).Scan(&r).Error
		if r == nil || *r == "" {
			_ = tx.Raw(`SELECT to_regclass('class_parent')::text`).Scan(&r).Error
			if r != nil && *r != "" {
				parentTbl = "class_parent"
			}
		}
	}

	q := fmt.Sprintf(`
		SELECT
			c.class_school_id              AS school_id,

			-- class
			c.class_id                     AS class_id,
			COALESCE(c.class_name,'')      AS class_name,
			COALESCE(c.class_slug,'')      AS class_slug,

			-- parent (LEFT JOIN, bisa NULL)
			cp.class_parent_id             AS parent_id,
			cp.class_parent_name           AS parent_name,
			cp.class_parent_slug           AS parent_slug,
			cp.class_parent_level          AS parent_level,

			-- term (LEFT JOIN, bisa NULL)
			c.class_academic_term_id       AS term_id,
			at.academic_term_name          AS term_name,
			at.academic_term_slug          AS term_slug,
			at.academic_term_academic_year AS term_year,
			at.academic_term_angkatan      AS term_angkatan
		FROM classes c
		LEFT JOIN %s cp
		  ON cp.class_parent_id = c.class_class_parent_id
		 AND cp.class_parent_school_id = c.class_school_id
		 AND cp.class_parent_deleted_at IS NULL
		LEFT JOIN academic_terms at
		  ON at.academic_term_id = c.class_academic_term_id
		 AND at.academic_term_school_id = c.class_school_id
		 AND at.academic_term_deleted_at IS NULL
		WHERE c.class_id = ? AND c.class_deleted_at IS NULL
	`, parentTbl)

	type dbRow struct {
		SchoolID     uuid.UUID  `gorm:"column:school_id"`
		ClassID      uuid.UUID  `gorm:"column:class_id"`
		ClassName    string     `gorm:"column:class_name"`
		ClassSlug    string     `gorm:"column:class_slug"`
		ParentID     *uuid.UUID `gorm:"column:parent_id"`
		ParentName   *string    `gorm:"column:parent_name"`
		ParentSlug   *string    `gorm:"column:parent_slug"`
		ParentLevel  *int16     `gorm:"column:parent_level"`
		TermID       *uuid.UUID `gorm:"column:term_id"`
		TermName     *string    `gorm:"column:term_name"`
		TermSlug     *string    `gorm:"column:term_slug"`
		TermYear     *string    `gorm:"column:term_year"`
		TermAngkatan *int       `gorm:"column:term_angkatan"`
	}
	var r dbRow

	if err := tx.Raw(q, classID).Scan(&r).Error; err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if r.SchoolID == uuid.Nil || r.ClassID == uuid.Nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Class tidak ditemukan")
	}
	if r.SchoolID != schoolID {
		return nil, fiber.NewError(fiber.StatusForbidden, "Class bukan milik school Anda")
	}

	trim := func(p *string) *string {
		if p == nil {
			return nil
		}
		s := strings.TrimSpace(*p)
		if s == "" {
			return nil
		}
		return &s
	}

	return &ClassParentAndTermCache{
		SchoolID: r.SchoolID,

		ClassID:   r.ClassID,
		ClassName: strings.TrimSpace(r.ClassName),
		ClassSlug: strings.TrimSpace(r.ClassSlug),

		ParentID: r.ParentID,
		ParentName: func() string {
			if r.ParentName != nil {
				return strings.TrimSpace(*r.ParentName)
			}
			return ""
		}(),
		ParentSlug:  trim(r.ParentSlug),
		ParentLevel: r.ParentLevel,

		TermID:       r.TermID,
		TermName:     trim(r.TermName),
		TermSlug:     trim(r.TermSlug),
		TermYear:     trim(r.TermYear),
		TermAngkatan: r.TermAngkatan,
	}, nil
}

func applyClassParentAndTermCacheToSection(mcs *secModel.ClassSectionModel, s *ClassParentAndTermCache) {
	// ---------- CLASS ----------
	name := strings.TrimSpace(s.ClassName)
	if name == "" {
		name = strings.TrimSpace(s.ClassSlug)
	}
	if name != "" {
		mcs.ClassSectionClassNameCache = &name
	} else {
		mcs.ClassSectionClassNameCache = nil
	}

	slug := strings.TrimSpace(s.ClassSlug)
	if slug != "" {
		mcs.ClassSectionClassSlugCache = &slug
	} else {
		mcs.ClassSectionClassSlugCache = nil
	}

	// ---------- PARENT ----------
	mcs.ClassSectionClassParentID = s.ParentID

	pName := strings.TrimSpace(s.ParentName)
	if pName != "" {
		mcs.ClassSectionClassParentNameCache = &pName
	} else {
		mcs.ClassSectionClassParentNameCache = nil
	}

	if s.ParentSlug != nil {
		ps := strings.TrimSpace(*s.ParentSlug)
		if ps != "" {
			mcs.ClassSectionClassParentSlugCache = &ps
		} else {
			mcs.ClassSectionClassParentSlugCache = nil
		}
	} else {
		mcs.ClassSectionClassParentSlugCache = nil
	}

	mcs.ClassSectionClassParentLevelCache = s.ParentLevel

	// ---------- TERM ----------
	mcs.ClassSectionAcademicTermID = s.TermID
	mcs.ClassSectionAcademicTermNameCache = s.TermName
	mcs.ClassSectionAcademicTermSlugCache = s.TermSlug
	mcs.ClassSectionAcademicTermAcademicYearCache = s.TermYear
	mcs.ClassSectionAcademicTermAngkatanCache = s.TermAngkatan

}

/* =========================================================
   Helpers: Join-code & util
========================================================= */

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

// Plaintext join-code: "<slug>-<partA><partB>"
func buildSectionJoinCode(slug string, id uuid.UUID) (string, error) {
	parts := strings.Split(id.String(), "-")
	if len(parts) == 5 {
		i1, i2, err := pickTwoDistinct(5)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s-%s%s", slug, parts[i1], parts[i2]), nil
	}
	s := strings.ReplaceAll(id.String(), "-", "")
	if len(s) >= 16 {
		return fmt.Sprintf("%s-%s%s", slug, s[:8], s[len(s)-8:]), nil
	}
	return fmt.Sprintf("%s-%s", slug, s), nil
}

func bcryptHash(s string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(s), bcrypt.DefaultCost)
}

func pickImageFile(c *fiber.Ctx, names ...string) *multipart.FileHeader {
	for _, n := range names {
		if fh, err := c.FormFile(n); err == nil && fh != nil && fh.Size > 0 {
			return fh
		}
	}
	return nil
}

/* =========================================================
   HANDLERS
========================================================= */

// POST /admin/class-sections
func (ctrl *ClassSectionController) CreateClassSection(c *fiber.Ctx) error {
	log.Printf("[SECTIONS][CREATE] ▶️ incoming request")

	// ---- School context: SELALU dari token/context (bukan slug/path) ----
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		// ResolveSchoolIDFromContext sudah balikin JsonError yang proper
		return err
	}

	// ⬇️ hanya DKM/admin yang boleh buat section
	if err := helperAuth.EnsureDKMSchool(c, schoolID); err != nil {
		log.Printf("[SECTIONS][CREATE] ❌ ensure DKM school failed: %v", err)
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusForbidden, "Hanya DKM/Admin yang diizinkan untuk mengelola section")
	}

	// ---- Parse req ----
	var req secDTO.ClassSectionCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	req.ClassSectionSchoolID = schoolID
	req.Normalize()

	// ---- Validasi ringan ----
	if strings.TrimSpace(req.ClassSectionName) == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Nama section wajib diisi")
	}
	// Kuota: total & taken (mirror pola di classes)
	if req.ClassSectionQuotaTotal != nil && *req.ClassSectionQuotaTotal < 0 {
		return helper.JsonError(c, fiber.StatusBadRequest, "Kuota total tidak boleh negatif")
	}
	if req.ClassSectionQuotaTaken != nil && *req.ClassSectionQuotaTaken < 0 {
		return helper.JsonError(c, fiber.StatusBadRequest, "Kuota terpakai tidak boleh negatif")
	}
	if req.ClassSectionQuotaTotal != nil && req.ClassSectionQuotaTaken != nil &&
		*req.ClassSectionQuotaTaken > *req.ClassSectionQuotaTotal {
		return helper.JsonError(c, fiber.StatusBadRequest, "Kuota terpakai tidak boleh melebihi kuota total")
	}

	if req.ClassSectionSubjectTeachersMaxSubjectsPerStudent != nil && *req.ClassSectionSubjectTeachersMaxSubjectsPerStudent < 0 {
		return helper.JsonError(c, fiber.StatusBadRequest, "Batas maksimal mapel per siswa tidak boleh negatif")
	}
	if req.ClassSectionSubjectTeachersEnrollmentMode != nil && strings.TrimSpace(*req.ClassSectionSubjectTeachersEnrollmentMode) != "" {
		if !isValidEnrollmentMode(*req.ClassSectionSubjectTeachersEnrollmentMode) {
			return helper.JsonError(c, fiber.StatusBadRequest, "Mode enrolment Subject-Teachers tidak valid (self_select | assigned | hybrid)")
		}
	}

	// ---- TX ----
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

	// ---- Validasi class belong-to school ----
	{
		var cls classModel.ClassModel
		if err := tx.Select("class_id, class_school_id").
			Where("class_id = ? AND class_deleted_at IS NULL", req.ClassSectionClassID).
			First(&cls).Error; err != nil {
			_ = tx.Rollback()
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, fiber.StatusBadRequest, "Class tidak ditemukan")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal validasi class")
		}
		if cls.ClassSchoolID != schoolID {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusForbidden, "Class bukan milik school Anda")
		}
	}

	// ---- Map req -> model ----
	m := req.ToModel()
	m.ClassSectionSchoolID = schoolID

	// (stats & CSST totals default 0 dari DB / struct, jadi dibiarkan)

	// ==== Cache GURU (opsional, via class_section_school_teacher_id) ====
	if m.ClassSectionSchoolTeacherID != nil {
		ts, err := teachersnap.ValidateAndCacheTeacher(
			tx,
			schoolID,
			*m.ClassSectionSchoolTeacherID,
		)
		if err != nil {
			_ = tx.Rollback()
			var fe *fiber.Error
			if errors.As(err, &fe) {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil snapshot guru: "+err.Error())
		}
		m.ClassSectionSchoolTeacherCache = teachersnap.ToJSON(ts)
	}

	// ==== Cache ASSISTANT GURU (opsional, via class_section_assistant_school_teacher_id) ====
	if m.ClassSectionAssistantSchoolTeacherID != nil {
		ts, err := teachersnap.ValidateAndCacheTeacher(
			tx,
			schoolID,
			*m.ClassSectionAssistantSchoolTeacherID,
		)
		if err != nil {
			_ = tx.Rollback()
			var fe *fiber.Error
			if errors.As(err, &fe) {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil snapshot asisten guru: "+err.Error())
		}
		m.ClassSectionAssistantSchoolTeacherCache = teachersnap.ToJSON(ts)
	}

	// ---- Cache relasi (class→parent/term) ----
	if snap, err := snapshotClassParentAndTerm(tx, schoolID, req.ClassSectionClassID); err != nil {
		_ = tx.Rollback()
		var fe *fiber.Error
		if errors.As(err, &fe) {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil snapshot class: "+err.Error())
	} else {
		applyClassParentAndTermCacheToSection(m, snap)
	}

	// ==== Cache ROOM (opsional, via class_section_class_room_id) ====
	if m.ClassSectionClassRoomID != nil {
		rs, err := sectionroomsnap.ValidateAndCacheRoom(tx, schoolID, *m.ClassSectionClassRoomID)
		if err != nil {
			_ = tx.Rollback()
			var fe *fiber.Error
			if errors.As(err, &fe) {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal validasi ruang kelas")
		}
		sectionroomsnap.ApplyRoomCacheToSection(m, rs)
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
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
	}
	m.ClassSectionSlug = uniqueSlug

	// ---- Generate join code (student/teacher) ----
	if m.ClassSectionID == uuid.Nil {
		m.ClassSectionID = uuid.New()
	}
	plainCode, err := buildSectionJoinCode(m.ClassSectionSlug, m.ClassSectionID)
	if err != nil {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membangun join code")
	}
	hashed, err := bcryptHash(plainCode)
	if err != nil {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal meng-hash join code")
	}
	now := time.Now()
	m.ClassSectionCode = &plainCode
	m.ClassSectionStudentCodeHash = hashed
	m.ClassSectionStudentCodeSetAt = &now

	tPlain, _ := buildSectionJoinCode(m.ClassSectionSlug+"-t", m.ClassSectionID)
	if tHash, e := bcryptHash(tPlain); e == nil {
		m.ClassSectionTeacherCodeHash = tHash
		m.ClassSectionTeacherCodeSetAt = &now
	} else {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal meng-hash teacher join code")
	}

	// ==========================
	// 📤 Upload file dari form-data ("file")
	// ==========================
	if fh, err := c.FormFile("file"); err == nil && fh != nil && fh.Size > 0 {
		ossSvc, err := helperOSS.NewOSSServiceFromEnv("")
		if err != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusBadGateway, "Konfigurasi OSS tidak valid")
		}

		slot := "class-sections"

		publicURL, err := helperOSS.UploadAnyToOSS(c.Context(), ossSvc, schoolID, slot, fh)
		if err != nil {
			_ = tx.Rollback()
			var fe *fiber.Error
			if errors.As(err, &fe) {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return helper.JsonError(c, fiber.StatusBadGateway, "Gagal upload file")
		}

		m.ClassSectionImageURL = &publicURL
		if key, kErr := helperOSS.ExtractKeyFromPublicURL(publicURL); kErr == nil {
			m.ClassSectionImageObjectKey = &key
		}
	}

	// ---- INSERT ----
	if err := tx.Create(m).Error; err != nil {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat section")
	}

	if err := tx.Commit().Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// ✅ Konsisten: data = 1 object section DTO (sudah include stats & CSST totals baru)
	return helper.JsonCreated(c, "Section berhasil dibuat", secDTO.FromModelClassSection(m))
}

// PATCH /admin/class-sections/:id
func (ctrl *ClassSectionController) UpdateClassSection(c *fiber.Ctx) error {
	sectionID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	// 🔐 Selalu pakai school_id dari token/context
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err
	}

	// Decode request (support JSON & multipart via helper DTO)
	var req secDTO.ClassSectionPatchRequest
	if err := secDTO.DecodePatchClassSectionFromRequest(c, &req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// TX
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

	// Ambil existing (lock)
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

	// 🔒 Tenant guard: section harus milik school di token
	if existing.ClassSectionSchoolID != schoolID {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusForbidden, "Section bukan milik school Anda")
	}

	// Guard akses DKM/admin pada school terkait (pakai schoolID dari token)
	if err := helperAuth.EnsureDKMSchool(c, schoolID); err != nil {
		_ = tx.Rollback()
		return err
	}

	// Sanity checks
	if req.ClassSectionName.Present && req.ClassSectionName.Value != nil {
		if strings.TrimSpace(*req.ClassSectionName.Value) == "" {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusBadRequest, "Nama section wajib diisi")
		}
	}

	// Kuota: total & taken
	if req.ClassSectionQuotaTotal.Present && req.ClassSectionQuotaTotal.Value != nil &&
		*req.ClassSectionQuotaTotal.Value < 0 {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusBadRequest, "Kuota total tidak boleh negatif")
	}
	if req.ClassSectionQuotaTaken.Present && req.ClassSectionQuotaTaken.Value != nil &&
		*req.ClassSectionQuotaTaken.Value < 0 {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusBadRequest, "Kuota terpakai tidak boleh negatif")
	}
	if req.ClassSectionQuotaTotal.Present && req.ClassSectionQuotaTotal.Value != nil &&
		req.ClassSectionQuotaTaken.Present && req.ClassSectionQuotaTaken.Value != nil &&
		*req.ClassSectionQuotaTaken.Value > *req.ClassSectionQuotaTotal.Value {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusBadRequest, "Kuota terpakai tidak boleh melebihi kuota total")
	}

	if req.ClassSectionSubjectTeachersMaxSubjectsPerStudent.Present && req.ClassSectionSubjectTeachersMaxSubjectsPerStudent.Value != nil {
		if *req.ClassSectionSubjectTeachersMaxSubjectsPerStudent.Value < 0 {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusBadRequest, "Batas maksimal mapel per siswa tidak boleh negatif")
		}
	}
	if req.ClassSectionSubjectTeachersEnrollmentMode.Present && req.ClassSectionSubjectTeachersEnrollmentMode.Value != nil {
		if !isValidEnrollmentMode(*req.ClassSectionSubjectTeachersEnrollmentMode.Value) {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusBadRequest, "Mode enrolment Subject-Teachers tidak valid (self_select | assigned | hybrid)")
		}
	}

	// Siapkan snapshot ROOM & GURU bila id dipatch
	var (
		roomSnapRequested bool
		roomSnap          *sectionroomsnap.RoomCache

		teacherSnapRequested     bool
		teacherSnapJSON          datatypes.JSON
		asstTeacherSnapRequested bool
		asstTeacherSnapJSON      datatypes.JSON
	)

	if req.ClassSectionClassRoomID.Present {
		roomSnapRequested = true
		if req.ClassSectionClassRoomID.Value != nil {
			rs, err := sectionroomsnap.ValidateAndCacheRoom(
				tx,
				schoolID,
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

	// Cache guru bila school_teacher_id dipatch
	if req.ClassSectionSchoolTeacherID.Present {
		teacherSnapRequested = true

		if req.ClassSectionSchoolTeacherID.Value != nil {
			ts, err := teachersnap.ValidateAndCacheTeacher(
				tx,
				schoolID,
				*req.ClassSectionSchoolTeacherID.Value,
			)
			if err != nil {
				_ = tx.Rollback()
				var fe *fiber.Error
				if errors.As(err, &fe) {
					return helper.JsonError(c, fe.Code, fe.Message)
				}
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal validasi guru")
			}
			teacherSnapJSON = teachersnap.ToJSON(ts)
		} else {
			// dipatch NULL → kosongkan snapshot
			teacherSnapJSON = teachersnap.ToJSON(nil)
		}
	}

	// Cache assistant guru bila assistant_school_teacher_id dipatch
	if req.ClassSectionAssistantSchoolTeacherID.Present {
		asstTeacherSnapRequested = true

		if req.ClassSectionAssistantSchoolTeacherID.Value != nil {
			ts, err := teachersnap.ValidateAndCacheTeacher(
				tx,
				schoolID,
				*req.ClassSectionAssistantSchoolTeacherID.Value,
			)
			if err != nil {
				_ = tx.Rollback()
				var fe *fiber.Error
				if errors.As(err, &fe) {
					return helper.JsonError(c, fe.Code, fe.Message)
				}
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal validasi asisten guru")
			}
			asstTeacherSnapJSON = teachersnap.ToJSON(ts)
		} else {
			// dipatch NULL → kosongkan snapshot asisten
			asstTeacherSnapJSON = teachersnap.ToJSON(nil)
		}
	}

	// Slug handling (unik per tenant)
	if req.ClassSectionSlug.Present && req.ClassSectionSlug.Value != nil {
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
					schoolID, existing.ClassSectionID,
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
					schoolID, existing.ClassSectionID,
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

	// Track perubahan status aktif
	wasActive := existing.ClassSectionIsActive
	newActive := wasActive
	if req.ClassSectionIsActive.Present && req.ClassSectionIsActive.Value != nil {
		newActive = *req.ClassSectionIsActive.Value
	}

	// Apply perubahan model dasar (termasuk relasi IDs, enrollment mode, dll)
	req.Apply(&existing)

	// Apply room snapshot jika field-nya dipatch (clear jika nil)
	if roomSnapRequested {
		sectionroomsnap.ApplyRoomCacheToSection(&existing, roomSnap)
	}

	// Apply teacher snapshot bila field-nya dipatch
	if teacherSnapRequested {
		existing.ClassSectionSchoolTeacherCache = teacherSnapJSON
	}
	if asstTeacherSnapRequested {
		existing.ClassSectionAssistantSchoolTeacherCache = asstTeacherSnapJSON
	}

	// Save
	if err := tx.Save(&existing).Error; err != nil {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui section")
	}

	// Upload image (opsional, multipart)
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
			if err := tx.Table("class_sections").
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
				}).Error; err != nil {
				_ = tx.Rollback()
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan metadata gambar")
			}
		}
	}

	// Update stats lembaga kalau status aktif berubah
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

	// Commit
	if err := tx.Commit().Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Re-fetch terbaru untuk response (sudah include kolom stats & CSST)
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

	// 🔐 Selalu pakai school_id dari token/context
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err
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

	// 🔒 Tenant guard: section harus milik school di token
	if m.ClassSectionSchoolID != schoolID {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusForbidden, "Section bukan milik school Anda")
	}

	// Guard akses DKM/admin pada school terkait
	if err := helperAuth.EnsureDKMSchool(c, schoolID); err != nil {
		_ = tx.Rollback()
		return err
	}

	// 🔒 GUARD: masih dipakai di class_section_subject_teachers (CSST)?
	var csstCount int64
	if err := tx.Model(&csstModel.ClassSectionSubjectTeacherModel{}).
		Where(`
			class_section_subject_teacher_school_id = ?
			AND class_section_subject_teacher_class_section_id = ?
			AND class_section_subject_teacher_deleted_at IS NULL
		`, m.ClassSectionSchoolID, m.ClassSectionID).
		Count(&csstCount).Error; err != nil {

		_ = tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengecek relasi subject teachers")
	}

	if csstCount > 0 {
		_ = tx.Rollback()
		return fiber.NewError(
			fiber.StatusBadRequest,
			"Section tidak dapat dihapus karena masih digunakan oleh pengampu mapel (subject teachers). Mohon hapus/ubah pengampu yang terkait terlebih dahulu.",
		)
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
