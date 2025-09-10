// file: internals/features/school/attendance_assesment/submissions/controller/submission_controller.go
package controller

import (
	"math"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	dto "masjidku_backend/internals/features/school/submissions_assesments/submissions/dto"
	model "masjidku_backend/internals/features/school/submissions_assesments/submissions/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
)

type SubmissionController struct {
	DB        *gorm.DB
	Validator *validator.Validate
}

func NewSubmissionController(db *gorm.DB) *SubmissionController {
	return &SubmissionController{
		DB:        db,
		Validator: validator.New(),
	}
}

// ===== Contoh registrasi route =====
// func RegisterSubmissionRoutes(app *fiber.App, db *gorm.DB) {
// 	ctrl := NewSubmissionController(db)
// 	g := app.Group("/api/a/submissions")
// 	g.Get("/", ctrl.List)
// 	g.Get("/:id", ctrl.GetByID)
// 	g.Post("/", ctrl.Create)
// 	g.Patch("/:id", ctrl.Patch)
// 	g.Patch("/:id/grade", ctrl.Grade)
// 	g.Delete("/:id", ctrl.Delete)
// }

// ============ Helpers ============
func (ctrl *SubmissionController) tenantMasjidID(c *fiber.Ctx) (uuid.UUID, error) {
	adminMasjidID, _ := helperAuth.GetMasjidIDFromToken(c)
	teacherMasjidID, _ := helperAuth.GetTeacherMasjidIDFromToken(c)

	switch {
	case adminMasjidID != uuid.Nil:
		return adminMasjidID, nil
	case teacherMasjidID != uuid.Nil:
		return teacherMasjidID, nil
	default:
		return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "Hanya admin atau guru yang diizinkan")
	}
}

func clampPage(n int) int {
	if n <= 0 {
		return 1
	}
	return n
}
func clampPerPage(n int) int {
	if n <= 0 {
		return 20
	}
	if n > 200 {
		return 200
	}
	return n
}

func applyFilters(q *gorm.DB, f *dto.ListSubmissionsQuery) *gorm.DB {
	if f == nil {
		return q
	}
	if f.MasjidID != nil {
		q = q.Where("submissions_masjid_id = ?", *f.MasjidID)
	}
	if f.AssessmentID != nil {
		q = q.Where("submissions_assessment_id = ?", *f.AssessmentID)
	}
	if f.StudentID != nil {
		q = q.Where("submissions_student_id = ?", *f.StudentID)
	}
	if f.Status != nil {
		q = q.Where("submissions_status = ?", *f.Status)
	}
	if f.SubmittedFrom != nil {
		q = q.Where("submissions_submitted_at >= ?", *f.SubmittedFrom)
	}
	if f.SubmittedTo != nil {
		q = q.Where("submissions_submitted_at < ?", *f.SubmittedTo)
	}
	return q
}

func applySort(q *gorm.DB, sort string) *gorm.DB {
	switch strings.TrimSpace(sort) {
	case "created_at":
		return q.Order("submissions_created_at ASC")
	case "desc_created_at", "":
		return q.Order("submissions_created_at DESC")
	case "submitted_at":
		return q.Order("submissions_submitted_at ASC NULLS LAST")
	case "desc_submitted_at":
		return q.Order("submissions_submitted_at DESC NULLS LAST")
	case "score":
		return q.Order("submissions_score ASC NULLS LAST")
	case "desc_score":
		return q.Order("submissions_score DESC NULLS LAST")
	default:
		return q.Order("submissions_created_at DESC")
	}
}

// ============ Handlers ============

// POST /
// POST /
func (ctrl *SubmissionController) Create(c *fiber.Ctx) error {
	var body dto.CreateSubmissionRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}

	// ===== Ambil masjid & student dari token (user/student) =====
	masjidID, err := helperAuth.GetActiveMasjidIDFromToken(c)
	if err != nil || masjidID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Masjid aktif tidak ditemukan di token")
	}
	studentID, err := helperAuth.GetMasjidStudentIDForMasjid(c, masjidID)
	if err != nil || studentID == uuid.Nil {
		// Tidak punya student record di masjid tsb
		return helper.JsonError(c, fiber.StatusForbidden, "Hanya siswa terdaftar yang diizinkan membuat submission")
	}

	// Paksa/override masjid & student dari token agar aman
	body.SubmissionMasjidID = masjidID
	body.SubmissionStudentID = studentID

	// ===== Validasi payload setelah di-override =====
	if err := ctrl.Validator.Struct(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Status default = submitted (bisa override kalau dikirim)
	status := model.SubmissionStatusSubmitted
	if body.SubmissionStatus != nil {
		status = *body.SubmissionStatus
	}

	sub := &model.Submission{
		SubmissionMasjidID:     body.SubmissionMasjidID,
		SubmissionAssessmentID: body.SubmissionAssessmentID, // wajib dikirim client
		SubmissionStudentID:    body.SubmissionStudentID,

		SubmissionText:        body.SubmissionText,
		SubmissionStatus:      status,
		SubmissionSubmittedAt: body.SubmissionSubmittedAt,
		SubmissionIsLate:      body.SubmissionIsLate,
	}

	// Auto isi submitted_at jika status submitted/resubmitted tapi belum ada waktu
	if (sub.SubmissionStatus == model.SubmissionStatusSubmitted || sub.SubmissionStatus == model.SubmissionStatusResubmitted) &&
		sub.SubmissionSubmittedAt == nil {
		now := time.Now()
		sub.SubmissionSubmittedAt = &now
	}

	if err := ctrl.DB.Create(sub).Error; err != nil {
		le := strings.ToLower(err.Error())
		if strings.Contains(le, "duplicate key") || strings.Contains(le, "unique constraint") {
			return helper.JsonError(c, fiber.StatusConflict, "Submission untuk assessment & student ini sudah ada")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonCreated(c, "Submission berhasil dibuat", dto.FromModel(sub))
}


// GET /
// GET /
func (ctrl *SubmissionController) List(c *fiber.Ctx) error {
	var q dto.ListSubmissionsQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}
	if err := ctrl.Validator.Struct(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// ← HANYA set default tenant bila memang ada (admin/guru),
	//    kalau tidak ada (uuid.Nil) BIARKAN kosong agar tidak mem-filter ke Nil.
	if tenantMasjidID, _ := ctrl.tenantMasjidID(c); tenantMasjidID != uuid.Nil && q.MasjidID == nil {
		q.MasjidID = &tenantMasjidID
	}

	// ❌ HAPUS blok yang memaksa q.MasjidID dan Forbidden:
	// if q.MasjidID == nil { q.MasjidID = &tenantMasjidID } else if *q.MasjidID != tenantMasjidID { ... }

	page := clampPage(q.Page)
	perPage := clampPerPage(q.PerPage)

	var total int64
	dbq := ctrl.DB.Model(&model.Submission{})
	dbq = applyFilters(dbq, &q)

	if err := dbq.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	var rows []model.Submission
	dbq = applySort(dbq, q.Sort)
	if err := dbq.Offset((page - 1) * perPage).Limit(perPage).Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	out := make([]dto.SubmissionResponse, 0, len(rows))
	for i := range rows {
		out = append(out, dto.FromModel(&rows[i]))
	}

	pagination := fiber.Map{
		"page":        page,
		"per_page":    perPage,
		"total":       total,
		"total_pages": int(math.Ceil(float64(total) / float64(perPage))),
	}

	return helper.JsonList(c, out, pagination)
}


// GET /:id
func (ctrl *SubmissionController) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	tenantMasjidID, err := ctrl.tenantMasjidID(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Hanya admin atau guru yang diizinkan")
	}

	var sub model.Submission
	if err := ctrl.DB.
		First(&sub, "submissions_id = ? AND submissions_masjid_id = ?", id, tenantMasjidID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Submission tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonOK(c, "OK", dto.FromModel(&sub))
}

// PATCH /:id
func (ctrl *SubmissionController) Patch(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	tenantMasjidID, err := ctrl.tenantMasjidID(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Hanya admin atau guru yang diizinkan")
	}

	var sub model.Submission
	if err := ctrl.DB.
		First(&sub, "submissions_id = ? AND submissions_masjid_id = ?", id, tenantMasjidID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Submission tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	var body dto.PatchSubmissionRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	// Validasi enum & range
	if err := ctrl.Validator.Struct(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	if body.SubmissionStatus != nil && body.SubmissionStatus.ShouldUpdate() && !body.SubmissionStatus.IsNull() {
		switch *body.SubmissionStatus.Value {
		case model.SubmissionStatusDraft, model.SubmissionStatusSubmitted, model.SubmissionStatusResubmitted,
			model.SubmissionStatusGraded, model.SubmissionStatusReturned:
		default:
			return helper.JsonError(c, fiber.StatusBadRequest, "submissions_status invalid")
		}
	}
	if body.SubmissionScore != nil && body.SubmissionScore.ShouldUpdate() && !body.SubmissionScore.IsNull() {
		if *body.SubmissionScore.Value < 0 || *body.SubmissionScore.Value > 100 {
			return helper.JsonError(c, fiber.StatusBadRequest, "submissions_score harus 0..100")
		}
	}

	updates := body.ToUpdates()

	// Auto: jika status jadi submitted/resubmitted dan submitted_at kosong di payload & sebelumnya kosong -> isi now
	if v, ok := updates["submissions_status"]; ok {
		if st, ok2 := v.(model.SubmissionStatus); ok2 {
			if (st == model.SubmissionStatusSubmitted || st == model.SubmissionStatusResubmitted) &&
				updates["submissions_submitted_at"] == nil && sub.SubmissionSubmittedAt == nil {
				updates["submissions_submitted_at"] = time.Now()
			}
		}
	}

	if len(updates) > 0 {
		if err := ctrl.DB.Model(&sub).Updates(updates).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
		}
	}

	if err := ctrl.DB.First(&sub, "submissions_id = ?", id).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonUpdated(c, "Submission diperbarui", dto.FromModel(&sub))
}

// PATCH /:id/grade
func (ctrl *SubmissionController) Grade(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	tenantMasjidID, err := ctrl.tenantMasjidID(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Hanya admin atau guru yang diizinkan")
	}

	var sub model.Submission
	if err := ctrl.DB.
		First(&sub, "submissions_id = ? AND submissions_masjid_id = ?", id, tenantMasjidID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Submission tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	var body dto.GradeSubmissionRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctrl.Validator.Struct(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	if body.SubmissionScore != nil && body.SubmissionScore.ShouldUpdate() && !body.SubmissionScore.IsNull() {
		if *body.SubmissionScore.Value < 0 || *body.SubmissionScore.Value > 100 {
			return helper.JsonError(c, fiber.StatusBadRequest, "submissions_score harus 0..100")
		}
	}

	updates := body.ToUpdates()

	// Otomatis status=graded jika ada perubahan score/feedback/graded_by/graded_at
	if _, ok := updates["submissions_score"]; ok {
		updates["submissions_status"] = model.SubmissionStatusGraded
	} else if _, ok := updates["submissions_feedback"]; ok {
		updates["submissions_status"] = model.SubmissionStatusGraded
	} else if _, ok := updates["submissions_graded_by_teacher_id"]; ok {
		updates["submissions_status"] = model.SubmissionStatusGraded
	} else if _, ok := updates["submissions_graded_at"]; ok {
		updates["submissions_status"] = model.SubmissionStatusGraded
	}

	// graded_at default now bila ada grading field namun graded_at tidak diberikan
	if updates["submissions_graded_at"] == nil &&
		(updates["submissions_score"] != nil || updates["submissions_feedback"] != nil || updates["submissions_graded_by_teacher_id"] != nil) {
		updates["submissions_graded_at"] = time.Now()
	}

	if len(updates) == 0 {
		return helper.JsonOK(c, "OK", dto.FromModel(&sub))
	}

	if err := ctrl.DB.Model(&sub).Updates(updates).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	if err := ctrl.DB.First(&sub, "submissions_id = ?", id).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonUpdated(c, "Submission dinilai", dto.FromModel(&sub))
}

// DELETE /:id (soft delete)
func (ctrl *SubmissionController) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	tenantMasjidID, err := ctrl.tenantMasjidID(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Hanya admin atau guru yang diizinkan")
	}

	var sub model.Submission
	if err := ctrl.DB.Select("submissions_id").
		First(&sub, "submissions_id = ? AND submissions_masjid_id = ?", id, tenantMasjidID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Submission tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	if err := ctrl.DB.Delete(&sub).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Gunakan 200 agar bisa kirim body
	return helper.JsonDeleted(c, "Submission dihapus", nil)
}
