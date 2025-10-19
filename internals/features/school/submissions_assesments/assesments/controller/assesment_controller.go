// file: internals/features/school/assessments/controller/assessment_controller.go
package controller

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"masjidku_backend/internals/features/school/classes/class_section_subject_teachers/snapshot"
	dto "masjidku_backend/internals/features/school/submissions_assesments/assesments/dto"
	model "masjidku_backend/internals/features/school/submissions_assesments/assesments/model"

	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
)

/*
========================================================

	Controller

========================================================
*/
type AssessmentController struct {
	DB        *gorm.DB
	Validator *validator.Validate
}

func NewAssessmentController(db *gorm.DB) *AssessmentController {
	return &AssessmentController{
		DB:        db,
		Validator: validator.New(),
	}
}

/* ========================================================
   Helpers
======================================================== */
// ===== struktur helper utk SELECT sesi =====

// ====== util autofill judul & deskripsi ======
func autofillTitle(current string, csstName *string, annTitle *string) string {
	if t := strings.TrimSpace(current); t != "" {
		return t
	}
	if csstName != nil && strings.TrimSpace(*csstName) != "" {
		return "Penilaian " + strings.TrimSpace(*csstName)
	}
	if annTitle != nil && strings.TrimSpace(*annTitle) != "" {
		return "Penilaian - " + strings.TrimSpace(*annTitle)
	}
	return "Penilaian"
}
func autofillDesc(curr *string, ann *sessRow, col *sessRow) *string {
	has := func(s string) bool { return strings.TrimSpace(s) != "" }
	if curr != nil && has(*curr) {
		return curr
	}
	var parts []string
	if ann != nil {
		line := "Diumumkan pada sesi"
		if ann.Title != nil && has(*ann.Title) {
			line += " \"" + strings.TrimSpace(*ann.Title) + "\""
		}
		if ann.Date != nil {
			line += ", tanggal " + ann.Date.Format("2006-01-02")
		}
		parts = append(parts, line)
	}
	if col != nil {
		line := "Dikumpulkan pada sesi"
		if col.Title != nil && has(*col.Title) {
			line += " \"" + strings.TrimSpace(*col.Title) + "\""
		}
		if col.Date != nil {
			line += ", tanggal " + col.Date.Format("2006-01-02")
		}
		parts = append(parts, line)
	}
	if len(parts) == 0 {
		return curr
	}
	s := strings.Join(parts, ". ") + "."
	return &s
}

type sessRow struct {
	ID       uuid.UUID  `gorm:"column:id"`
	MasjidID uuid.UUID  `gorm:"column:masjid_id"`
	StartsAt *time.Time `gorm:"column:starts_at"`
	Date     *time.Time `gorm:"column:date"`
	Deleted  *time.Time `gorm:"column:deleted_at"`
	Title    *string    `gorm:"column:title"`
}

func (ctl *AssessmentController) fetchSess(c *fiber.Ctx, id uuid.UUID) (*sessRow, error) {
	if id == uuid.Nil {
		return nil, nil
	}
	var r sessRow
	err := ctl.DB.WithContext(c.Context()).
		Table("class_attendance_sessions").
		Select(`
			class_attendance_session_id                  AS id,
			class_attendance_session_masjid_id           AS masjid_id,
			class_attendance_session_starts_at           AS starts_at,
			(class_attendance_session_date)::timestamptz AS date,
			class_attendance_session_deleted_at          AS deleted_at,
			COALESCE(
				NULLIF(TRIM(class_attendance_session_display_title), ''),
				NULLIF(TRIM(class_attendance_session_title), '')
			) AS title
		`).
		Where("class_attendance_session_id = ?", id).
		Take(&r).Error
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func pickTime(s *sessRow) time.Time {
	if s == nil {
		return time.Now().UTC()
	}
	if s.StartsAt != nil {
		return s.StartsAt.UTC()
	}
	if s.Date != nil {
		return s.Date.UTC()
	}
	return time.Now().UTC()
}
func makeSessionSnap(s *sessRow) datatypes.JSONMap {
	if s == nil {
		return datatypes.JSONMap{}
	}
	m := map[string]any{
		"captured_at": time.Now().UTC(),
		"session_id":  s.ID,
	}
	if s.StartsAt != nil {
		m["starts_at"] = s.StartsAt.UTC()
	}
	if s.Date != nil {
		m["date"] = s.Date.UTC()
	}
	if s.Title != nil && strings.TrimSpace(*s.Title) != "" {
		m["title"] = strings.TrimSpace(*s.Title) // ← tadinya "display_title"
	}
	return datatypes.JSONMap(m)
}

func parseUUIDParam(c *fiber.Ctx, name string) (uuid.UUID, error) {
	return uuid.Parse(strings.TrimSpace(c.Params(name)))
}

// validasi guru milik masjid
// assertTeacherBelongsToMasjid: pastikan masjid_teacher milik masjid
// was: func (ctl *AssessmentController) assertTeacherBelongsToMasjid(ctx fiber.Ctx, masjidID uuid.UUID, teacherID *uuid.UUID) error
func (ctl *AssessmentController) assertTeacherBelongsToMasjid(c *fiber.Ctx, masjidID uuid.UUID, teacherID *uuid.UUID) error {
	if teacherID == nil || *teacherID == uuid.Nil {
		return nil
	}

	// Ambil masjid_id dari masjid_teachers
	var row struct {
		M uuid.UUID `gorm:"column:m"`
	}
	if err := ctl.DB.WithContext(c.Context()).
		Table("masjid_teachers").
		Select("masjid_teacher_masjid_id AS m").
		Where("masjid_teacher_id = ? AND masjid_teacher_deleted_at IS NULL", *teacherID).
		Take(&row).Error; err != nil {
		return err
	}

	// Pastikan teacher milik masjid yang sama
	if row.M != masjidID {
		return fiber.NewError(fiber.StatusForbidden, "Guru bukan milik masjid Anda")
	}
	return nil
}

// Resolver akses: DKM/Admin via helper, atau Teacher pada masjid tsb.
func resolveMasjidForDKMOrTeacher(c *fiber.Ctx) (uuid.UUID, error) {
	// 1) Ambil masjid context (path/header/cookie/query/host/token)
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return uuid.Nil, err
	}

	// 2) Coba jalur DKM/Admin
	if id, er := helperAuth.EnsureMasjidAccessDKM(c, mc); er == nil && id != uuid.Nil {
		return id, nil
	}

	// 3) Fallback: izinkan GURU pada masjid ini
	var masjidID uuid.UUID
	if mc.ID != uuid.Nil {
		masjidID = mc.ID
	} else if s := strings.TrimSpace(mc.Slug); s != "" {
		id, er := helperAuth.GetMasjidIDBySlug(c, s)
		if er != nil || id == uuid.Nil {
			return uuid.Nil, fiber.NewError(fiber.StatusNotFound, "Masjid (slug) tidak ditemukan")
		}
		masjidID = id
	} else {
		return uuid.Nil, helperAuth.ErrMasjidContextMissing
	}

	if helperAuth.IsTeacherInMasjid(c, masjidID) {
		return masjidID, nil
	}
	return uuid.Nil, helperAuth.ErrMasjidContextForbidden
}

/* ===============================
   Handlers
=============================== */

// POST /assessments
func (ctl *AssessmentController) Create(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	var req dto.CreateAssessmentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	req.Normalize()

	// 🔒 resolve & authorize
	mid, err := resolveMasjidForDKMOrTeacher(c)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	// Enforce tenant
	req.AssessmentMasjidID = mid

	// =============== CSST SNAPSHOT (opsional, +auto teacher & auto-title) ===============
	var csstSnap datatypes.JSONMap
	var csstName *string
	if req.AssessmentClassSectionSubjectTeacherID != nil && *req.AssessmentClassSectionSubjectTeacherID != uuid.Nil {
		cs, er := snapshot.ValidateAndSnapshotCSST(ctl.DB.WithContext(c.Context()), mid, *req.AssessmentClassSectionSubjectTeacherID)
		if er != nil {
			if fe, ok := er.(*fiber.Error); ok {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return helper.JsonError(c, fiber.StatusBadRequest, er.Error())
		}
		// Auto-isi created_by_teacher_id
		if req.AssessmentCreatedByTeacherID == nil && cs.TeacherID != nil && *cs.TeacherID != uuid.Nil {
			tid := *cs.TeacherID
			req.AssessmentCreatedByTeacherID = &tid
		}
		// simpan csst name utk autofill judul
		csstName = cs.Name

		// Simpan snapshot (convert datatypes.JSON → JSONMap)
		if jb := snapshot.ToJSON(cs); len(jb) > 0 {
			var m map[string]any
			_ = json.Unmarshal(jb, &m)
			csstSnap = datatypes.JSONMap(m)
		}
	}

	// DTO validation
	if err := ctl.Validator.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Validasi creator teacher (opsional)
	// before: if err := ctl.assertTeacherBelongsToMasjid(*c, mid, req.AssessmentCreatedByTeacherID); err != nil {
	if err := ctl.assertTeacherBelongsToMasjid(c, mid, req.AssessmentCreatedByTeacherID); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// ====== Tentukan mode dari presence sesi ======
	hasAnn := req.AssessmentAnnounceSessionID != nil && *req.AssessmentAnnounceSessionID != uuid.Nil
	hasCol := req.AssessmentCollectSessionID != nil && *req.AssessmentCollectSessionID != uuid.Nil
	mode := "date"
	if hasAnn || hasCol {
		mode = "session"
	}

	// ====== MODE: session ======
	if mode == "session" {
		var ann, col *sessRow
		if hasAnn {
			r, er := ctl.fetchSess(c, *req.AssessmentAnnounceSessionID)
			if er != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "Sesi announce tidak ditemukan")
			}
			if r.Deleted != nil || r.MasjidID != mid {
				return helper.JsonError(c, fiber.StatusForbidden, "Sesi announce bukan milik masjid Anda / sudah dihapus")
			}
			ann = r
		}
		if hasCol {
			r, er := ctl.fetchSess(c, *req.AssessmentCollectSessionID)
			if er != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "Sesi collect tidak ditemukan")
			}
			if r.Deleted != nil || r.MasjidID != mid {
				return helper.JsonError(c, fiber.StatusForbidden, "Sesi collect bukan milik masjid Anda / sudah dihapus")
			}
			col = r
		}
		if ann != nil && col != nil && pickTime(col).Before(pickTime(ann)) {
			return helper.JsonError(c, fiber.StatusBadRequest, "collect_session harus sama atau setelah announce_session")
		}

		row := req.ToModel()
		row.AssessmentSubmissionMode = "session"

		// Turunkan start/due dari sesi bila belum diisi
		if row.AssessmentStartAt == nil && ann != nil {
			if ann.StartsAt != nil {
				row.AssessmentStartAt = ann.StartsAt
			} else if ann.Date != nil {
				row.AssessmentStartAt = ann.Date
			}
		}
		if row.AssessmentDueAt == nil && col != nil {
			if col.StartsAt != nil {
				row.AssessmentDueAt = col.StartsAt
			} else if col.Date != nil {
				row.AssessmentDueAt = col.Date
			}
		}

		// ===== auto-title & auto-desc jika kosong =====
		row.AssessmentTitle = autofillTitle(row.AssessmentTitle, csstName, func() *string {
			if ann != nil {
				return ann.Title
			}
			return nil
		}())
		row.AssessmentDescription = autofillDesc(row.AssessmentDescription, ann, col)

		// Snapshots
		if csstSnap != nil {
			row.AssessmentCSSTSnapshot = csstSnap
		}
		if ann != nil {
			row.AssessmentAnnounceSessionSnapshot = makeSessionSnap(ann)
		}
		if col != nil {
			row.AssessmentCollectSessionSnapshot = makeSessionSnap(col)
		}

		if err := ctl.DB.WithContext(c.Context()).Create(&row).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat assessment")
		}
		return helper.JsonCreated(c, "Assessment (mode session) berhasil dibuat", dto.FromModelAssesment(row))
	}

	// ===== MODE: date =====
	if req.AssessmentStartAt != nil && req.AssessmentDueAt != nil &&
		req.AssessmentDueAt.Before(*req.AssessmentStartAt) {
		return helper.JsonError(c, fiber.StatusBadRequest, "assessment_due_at harus setelah atau sama dengan assessment_start_at")
	}

	row := req.ToModel()
	row.AssessmentSubmissionMode = "date"

	// Auto-title (pakai csstName bila title kosong)
	row.AssessmentTitle = autofillTitle(row.AssessmentTitle, csstName, nil)

	// Snapshot CSST bila ada
	if csstSnap != nil {
		row.AssessmentCSSTSnapshot = csstSnap
	}

	if err := ctl.DB.WithContext(c.Context()).Create(&row).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat assessment")
	}
	return helper.JsonCreated(c, "Assessment (mode date) berhasil dibuat", dto.FromModelAssesment(row))
}

// PATCH /assessments/:id
func (ctl *AssessmentController) Patch(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	id, err := helper.ParseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "assessment_id tidak valid")
	}

	var req dto.PatchAssessmentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	req.Normalize()
	if err := ctl.Validator.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// 🔒 resolve & authorize
	mid, err := resolveMasjidForDKMOrTeacher(c)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var existing model.AssessmentModel
	if err := ctl.DB.WithContext(c.Context()).
		Where(`
			assessment_id = ?
			AND assessment_masjid_id = ?
			AND assessment_deleted_at IS NULL
		`, id, mid).
		First(&existing).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// validasi guru bila diubah
	// before: if err := ctl.assertTeacherBelongsToMasjid(*c, mid, req.AssessmentCreatedByTeacherID); err != nil {
	if err := ctl.assertTeacherBelongsToMasjid(c, mid, req.AssessmentCreatedByTeacherID); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// ==== (Opsional) update CSST snapshot bila CSST diubah ====
	if req.AssessmentClassSectionSubjectTeacherID != nil {
		if *req.AssessmentClassSectionSubjectTeacherID == uuid.Nil {
			existing.AssessmentClassSectionSubjectTeacherID = nil
			existing.AssessmentCSSTSnapshot = datatypes.JSONMap{}
		} else {
			cs, er := snapshot.ValidateAndSnapshotCSST(ctl.DB.WithContext(c.Context()), mid, *req.AssessmentClassSectionSubjectTeacherID)
			if er != nil {
				if fe, ok := er.(*fiber.Error); ok {
					return helper.JsonError(c, fe.Code, fe.Message)
				}
				return helper.JsonError(c, fiber.StatusBadRequest, er.Error())
			}
			existing.AssessmentClassSectionSubjectTeacherID = req.AssessmentClassSectionSubjectTeacherID
			if jb := snapshot.ToJSON(cs); len(jb) > 0 {
				var m map[string]any
				_ = json.Unmarshal(jb, &m)
				existing.AssessmentCSSTSnapshot = datatypes.JSONMap(m)
			}
		}
	}

	// ==== Hitung FINAL session IDs (memperhatikan niat PATCH untuk clear) ====
	finalAnnID := existing.AssessmentAnnounceSessionID
	if req.AssessmentAnnounceSessionID != nil {
		if *req.AssessmentAnnounceSessionID == uuid.Nil {
			finalAnnID = nil
		} else {
			v := *req.AssessmentAnnounceSessionID
			finalAnnID = &v
		}
	}
	finalColID := existing.AssessmentCollectSessionID
	if req.AssessmentCollectSessionID != nil {
		if *req.AssessmentCollectSessionID == uuid.Nil {
			finalColID = nil
		} else {
			v := *req.AssessmentCollectSessionID
			finalColID = &v
		}
	}

	// Tentukan mode hasil
	nextMode := "date"
	if finalAnnID != nil || finalColID != nil {
		nextMode = "session"
	}

	// ===== MODE: session =====
	if nextMode == "session" {
		var ann, col *sessRow
		if finalAnnID != nil {
			r, er := ctl.fetchSess(c, *finalAnnID)
			if er != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "Sesi announce tidak ditemukan")
			}
			if r.Deleted != nil || r.MasjidID != mid {
				return helper.JsonError(c, fiber.StatusForbidden, "Sesi announce bukan milik masjid Anda / sudah dihapus")
			}
			ann = r
		}
		if finalColID != nil {
			r, er := ctl.fetchSess(c, *finalColID)
			if er != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "Sesi collect tidak ditemukan")
			}
			if r.Deleted != nil || r.MasjidID != mid {
				return helper.JsonError(c, fiber.StatusForbidden, "Sesi collect bukan milik masjid Anda / sudah dihapus")
			}
			col = r
		}
		if ann != nil && col != nil && pickTime(col).Before(pickTime(ann)) {
			return helper.JsonError(c, fiber.StatusBadRequest, "collect_session harus sama atau setelah announce_session")
		}

		// Terapkan PATCH ke existing (tanpa simpan dulu)
		req.Apply(&existing)

		// Set mode & session ids final
		existing.AssessmentSubmissionMode = "session"
		existing.AssessmentAnnounceSessionID = finalAnnID
		existing.AssessmentCollectSessionID = finalColID

		// Jika user TIDAK kirim start/due di PATCH, turunkan dari sesi (agar konsisten)
		if req.AssessmentStartAt == nil && existing.AssessmentStartAt == nil && ann != nil {
			if ann.StartsAt != nil {
				existing.AssessmentStartAt = ann.StartsAt
			} else if ann.Date != nil {
				existing.AssessmentStartAt = ann.Date
			}
		}
		if req.AssessmentDueAt == nil && existing.AssessmentDueAt == nil && col != nil {
			if col.StartsAt != nil {
				existing.AssessmentDueAt = col.StartsAt
			} else if col.Date != nil {
				existing.AssessmentDueAt = col.Date
			}
		}

		// Snapshots sesi
		if ann != nil {
			existing.AssessmentAnnounceSessionSnapshot = makeSessionSnap(ann)
		} else {
			existing.AssessmentAnnounceSessionSnapshot = datatypes.JSONMap{}
		}
		if col != nil {
			existing.AssessmentCollectSessionSnapshot = makeSessionSnap(col)
		} else {
			existing.AssessmentCollectSessionSnapshot = datatypes.JSONMap{}
		}

		existing.AssessmentUpdatedAt = time.Now()
		if err := ctl.DB.WithContext(c.Context()).Save(&existing).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui assessment")
		}
		return helper.JsonUpdated(c, "Assessment (mode session) diperbarui", dto.FromModelAssesment(existing))
	}

	// ===== MODE: date =====
	// Validasi kombinasi waktu
	switch {
	case req.AssessmentStartAt != nil && req.AssessmentDueAt != nil:
		if req.AssessmentDueAt.Before(*req.AssessmentStartAt) {
			return helper.JsonError(c, fiber.StatusBadRequest, "assessment_due_at harus setelah atau sama dengan assessment_start_at")
		}
	case req.AssessmentStartAt != nil && req.AssessmentDueAt == nil:
		if existing.AssessmentDueAt != nil && existing.AssessmentDueAt.Before(*req.AssessmentStartAt) {
			return helper.JsonError(c, fiber.StatusBadRequest, "Tanggal due saat ini lebih awal dari start baru")
		}
	case req.AssessmentStartAt == nil && req.AssessmentDueAt != nil:
		if existing.AssessmentStartAt != nil && req.AssessmentDueAt.Before(*existing.AssessmentStartAt) {
			return helper.JsonError(c, fiber.StatusBadRequest, "assessment_due_at tidak boleh sebelum assessment_start_at")
		}
	}

	// Terapkan PATCH & set mode/date
	req.Apply(&existing)
	existing.AssessmentSubmissionMode = "date"

	// Jika pindah ke date → bersihkan session IDs & snapshots bila user clear
	existing.AssessmentAnnounceSessionID = finalAnnID // akan nil jika user kirim UUID nil
	existing.AssessmentCollectSessionID = finalColID
	if finalAnnID == nil {
		existing.AssessmentAnnounceSessionSnapshot = datatypes.JSONMap{}
	}
	if finalColID == nil {
		existing.AssessmentCollectSessionSnapshot = datatypes.JSONMap{}
	}

	existing.AssessmentUpdatedAt = time.Now()
	if err := ctl.DB.WithContext(c.Context()).Save(&existing).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui assessment")
	}
	return helper.JsonUpdated(c, "Assessment (mode date) diperbarui", dto.FromModelAssesment(existing))
}

// DELETE /assessments/:id (soft delete)
func (ctl *AssessmentController) Delete(c *fiber.Ctx) error {
	// Pastikan helper slug→id bisa akses DB dari context
	c.Locals("DB", ctl.DB)

	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "assessment_id tidak valid")
	}

	// 🔒 resolve & authorize
	mid, err := resolveMasjidForDKMOrTeacher(c)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var row model.AssessmentModel
	if err := ctl.DB.WithContext(c.Context()).
		Where(`
			assessment_id = ?
			AND assessment_masjid_id = ?
			AND assessment_deleted_at IS NULL
		`, id, mid).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	if err := ctl.DB.WithContext(c.Context()).Delete(&row).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus assessment")
	}

	return helper.JsonDeleted(c, "Assessment dihapus", fiber.Map{
		"assessment_id": id,
	})
}

/* ========================================================
   Helpers (local)
======================================================== */

func atoiOr(def int, s string) int {
	if s == "" {
		return def
	}
	n := 0
	sign := 1
	for i := 0; i < len(s); i++ {
		if i == 0 && s[i] == '-' {
			sign = -1
			continue
		}
		if s[i] < '0' || s[i] > '9' {
			return def
		}
		n = n*10 + int(s[i]-'0')
	}
	n *= sign
	if n <= 0 {
		return def
	}
	return n
}

func eqTrue(s string) bool {
	v := strings.TrimSpace(strings.ToLower(s))
	return v == "1" || v == "true"
}
