// file: internals/features/school/attendance_assesment/attendance_sessions/controller/user_attendance_urls_controller.go
package controller

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	dto "masjidku_backend/internals/features/school/sessions_assesment/sessions/dto"
	model "masjidku_backend/internals/features/school/sessions_assesment/sessions/model"
	helperAuth "masjidku_backend/internals/helpers/auth"

	helperOSS "masjidku_backend/internals/helpers/oss" // alias sama saja; jelas bahwa UploadImageToOSS dkk dari file oss_file.go
)

// Struct & Constructor sesuai permintaan
type UserAttendanceUrlController struct {
	DB        *gorm.DB
	Validator *validator.Validate
}

func NewUserAttendanceUrlController(db *gorm.DB) *UserAttendanceUrlController {
	return &UserAttendanceUrlController{
		DB:        db,
		Validator: validator.New(),
	}
}

/*
Routes (contoh):
g := app.Group("/api/a/user-attendance-urls")
g.Post("/", ctl.CreateJSON)
g.Post("/multipart", ctl.CreateMultipart)
g.Patch("/:id", ctl.Update)      // JSON atau Multipart
g.Get("/:id", ctl.GetByID)
g.Get("/", ctl.ListByAttendance) // ?attendance_id=...&limit=20&offset=0
g.Delete("/:id", ctl.SoftDelete) // soft delete
*/

// =============================
// CREATE - JSON
// =============================
// POST /api/a/user-attendance-urls
func (ctl *UserAttendanceUrlController) CreateJSON(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	var req dto.CreateUserAttendanceURLRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.Validator.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	m := dto.NewUserAttendanceURLModelFromCreate(req, masjidID)

	if err := ctl.DB.WithContext(c.Context()).Create(&m).Error; err != nil {
		if isUniqueViolation(err, "uq_uau_attendance_href") {
			return fiber.NewError(http.StatusConflict, "URL sudah terdaftar untuk attendance tersebut")
		}
		log.Println("[ERROR] create user_attendance_urls:", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat data")
	}

	return c.Status(http.StatusCreated).JSON(dto.ToUserAttendanceURLResponse(m))
}

// =============================
// CREATE - MULTIPART (upload file)
// =============================
// POST /api/a/user-attendance-urls/multipart
// Form fields:
// - file (required)
// - user_attendance_urls_attendance_id (uuid, required)
// - user_attendance_urls_label (optional)
// - user_attendance_urls_uploader_teacher_id (uuid, optional)
// - user_attendance_urls_uploader_user_id (uuid, optional)
func (ctl *UserAttendanceUrlController) CreateMultipart(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	attIDStr := strings.TrimSpace(c.FormValue("user_attendance_urls_attendance_id"))
	attID, err := uuid.Parse(attIDStr)
	if err != nil || attID == uuid.Nil {
		return fiber.NewError(fiber.StatusBadRequest, "attendance_id tidak valid")
	}

	var (
		labelPtr *string
	)
	if lbl := strings.TrimSpace(c.FormValue("user_attendance_urls_label")); lbl != "" {
		labelPtr = &lbl
	}

	var uploaderTeacherID *uuid.UUID
	if s := strings.TrimSpace(c.FormValue("user_attendance_urls_uploader_teacher_id")); s != "" {
		if v, e := uuid.Parse(s); e == nil {
			uploaderTeacherID = &v
		}
	}
	var uploaderUserID *uuid.UUID
	if s := strings.TrimSpace(c.FormValue("user_attendance_urls_uploader_user_id")); s != "" {
		if v, e := uuid.Parse(s); e == nil {
			uploaderUserID = &v
		}
	}

	// file wajib
	fh, err := c.FormFile("file")
	if err != nil || fh == nil {
		return fiber.NewError(fiber.StatusBadRequest, "File tidak ditemukan (field 'file')")
	}

	// Init OSS service (root prefix)
	svc, err := helperOSS.NewOSSServiceFromEnv("")
	if err != nil {
		log.Println("[OSS] init error:", err)
		return fiber.NewError(fiber.StatusBadGateway, "Gagal inisialisasi storage")
	}
	// Upload sebagai WEBP ke path: masjids/{masjid}/images/attendance/{attendance_id}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	slot := fmt.Sprintf("attendance/%s", attID.String())
	publicURL, upErr := helperOSS.UploadImageToOSS(ctx, svc, masjidID, slot, fh)
	if upErr != nil {
		return upErr // sudah fiber.Error di helper untuk beberapa kasus
	}

	req := dto.CreateUserAttendanceURLRequest{
		UserAttendanceURLsAttendanceID:      attID,
		UserAttendanceURLsLabel:             labelPtr,
		UserAttendanceURLsHref:              publicURL,
		UserAttendanceURLsUploaderTeacherID: uploaderTeacherID,
		UserAttendanceURLsUploaderUserID:    uploaderUserID,
	}
	if err := ctl.Validator.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	m := dto.NewUserAttendanceURLModelFromCreate(req, masjidID)
	if err := ctl.DB.WithContext(c.Context()).Create(&m).Error; err != nil {
		// jika gagal DB, hapus object yang tadi diupload agar tidak orphan (best effort)
		_ = helperOSS.DeleteByPublicURLENV(publicURL, 10*time.Second)

		if isUniqueViolation(err, "uq_uau_attendance_href") {
			return fiber.NewError(http.StatusConflict, "URL sudah terdaftar untuk attendance tersebut")
		}
		log.Println("[ERROR] create(multipart) user_attendance_urls:", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat data")
	}

	return c.Status(http.StatusCreated).JSON(dto.ToUserAttendanceURLResponse(m))
}

// =============================
// UPDATE - JSON / MULTIPART
// =============================
// PATCH /api/a/user-attendance-urls/:id
// JSON: gunakan body seperti Update DTO
// Multipart: kirim 'file' untuk ganti file; opsional label/href lain akan diabaikan
// Query opsional: ?trash_days=7 (default 7)
func (ctl *UserAttendanceUrlController) Update(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}
	id, perr := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if perr != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	// Ambil existing
	var m model.UserAttendanceURLModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("user_attendance_urls_id = ? AND user_attendance_urls_masjid_id = ? AND user_attendance_urls_deleted_at IS NULL",
			id, masjidID).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(http.StatusNotFound, "Data tidak ditemukan")
		}
		log.Println("[ERROR] fetch user_attendance_urls:", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	ct := strings.ToLower(strings.TrimSpace(c.Get(fiber.HeaderContentType)))
	isMultipart := strings.HasPrefix(ct, fiber.MIMEMultipartForm)

	// ====== MODE MULTIPART: ganti file ======
	if isMultipart {
		fh, ferr := c.FormFile("file")
		if ferr != nil || fh == nil {
			return fiber.NewError(fiber.StatusBadRequest, "File tidak ditemukan (field 'file')")
		}

		trashDays := 7
		if v := strings.TrimSpace(c.Query("trash_days")); v != "" {
			if d, e := strconv.Atoi(v); e == nil && d > 0 && d <= 365 {
				trashDays = d
			}
		}

		oldURL := m.UserAttendanceURLsHref

		// Upload baru
		svc, err := helperOSS.NewOSSServiceFromEnv("")
		if err != nil {
			log.Println("[OSS] init error:", err)
			return fiber.NewError(fiber.StatusBadGateway, "Gagal inisialisasi storage")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		slot := fmt.Sprintf("attendance/%s", m.UserAttendanceURLsAttendanceID.String())
		newURL, upErr := helperOSS.UploadImageToOSS(ctx, svc, masjidID, slot, fh)
		if upErr != nil {
			return upErr
		}

		// Set field model
		now := time.Now()
		delAt := now.Add(time.Duration(trashDays) * 24 * time.Hour)
		m.UserAttendanceURLsTrashURL = strPtr(oldURL)
		m.UserAttendanceURLsDeletePendingUntil = &delAt
		m.UserAttendanceURLsHref = newURL
		m.UserAttendanceURLsUpdatedAt = now

		if err := ctl.DB.WithContext(c.Context()).Save(&m).Error; err != nil {
			// rollback upload baru agar tidak orphan
			_ = helperOSS.DeleteByPublicURLENV(newURL, 10*time.Second)

			if isUniqueViolation(err, "uq_uau_attendance_href") {
				return fiber.NewError(http.StatusConflict, "URL sudah terdaftar untuk attendance tersebut")
			}
			log.Println("[ERROR] update(multipart) user_attendance_urls:", err)
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui data")
		}

		return c.JSON(dto.ToUserAttendanceURLResponse(m))
	}

	// ====== MODE JSON: partial fields ======
	var req dto.UpdateUserAttendanceURLRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.Validator.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	dto.ApplyUpdateToUserAttendanceURLModel(&m, req)

	if err := ctl.DB.WithContext(c.Context()).Save(&m).Error; err != nil {
		if isUniqueViolation(err, "uq_uau_attendance_href") {
			return fiber.NewError(http.StatusConflict, "URL sudah terdaftar untuk attendance tersebut")
		}
		log.Println("[ERROR] update(json) user_attendance_urls:", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui data")
	}

	return c.JSON(dto.ToUserAttendanceURLResponse(m))
}

// GET /api/a/user-attendance-urls/:id
func (ctl *UserAttendanceUrlController) GetByID(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}
	id, perr := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if perr != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	var m model.UserAttendanceURLModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("user_attendance_urls_id = ? AND user_attendance_urls_masjid_id = ? AND user_attendance_urls_deleted_at IS NULL",
			id, masjidID).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(http.StatusNotFound, "Data tidak ditemukan")
		}
		log.Println("[ERROR] get user_attendance_urls by id:", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	return c.JSON(dto.ToUserAttendanceURLResponse(m))
}

// GET /api/a/user-attendance-urls?attendance_id=:attId&limit=20&offset=0
func (ctl *UserAttendanceUrlController) ListByAttendance(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}
	attIDStr := strings.TrimSpace(c.Query("attendance_id"))
	attID, err := uuid.Parse(attIDStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "attendance_id tidak valid")
	}
	limit := clampInt(parseInt(c.Query("limit", "20"), 20), 1, 100)
	offset := maxInt(parseInt(c.Query("offset", "0"), 0), 0)

	var rows []model.UserAttendanceURLModel
	if err := ctl.DB.WithContext(c.Context()).
		Where(`
			user_attendance_urls_masjid_id = ?
			AND user_attendance_urls_attendance_id = ?
			AND user_attendance_urls_deleted_at IS NULL
		`, masjidID, attID).
		Order("user_attendance_urls_created_at DESC").
		Limit(limit).Offset(offset).
		Find(&rows).Error; err != nil {
		log.Println("[ERROR] list user_attendance_urls:", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	out := make([]dto.UserAttendanceURLResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, dto.ToUserAttendanceURLResponse(r))
	}
	return c.JSON(out)
}

// DELETE /api/a/user-attendance-urls/:id  (soft delete)
func (ctl *UserAttendanceUrlController) SoftDelete(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}
	id, perr := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if perr != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	var m model.UserAttendanceURLModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("user_attendance_urls_id = ? AND user_attendance_urls_masjid_id = ? AND user_attendance_urls_deleted_at IS NULL",
			id, masjidID).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(http.StatusNotFound, "Data tidak ditemukan")
		}
		log.Println("[ERROR] fetch before soft delete:", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	now := time.Now()
	if err := ctl.DB.WithContext(c.Context()).
		Model(&m).
		Update("user_attendance_urls_deleted_at", now).Error; err != nil {
		log.Println("[ERROR] soft delete user_attendance_urls:", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus data")
	}

	return c.SendStatus(http.StatusNoContent)
}

// ================ Helpers ================
func isUniqueViolation(err error, constraintHint string) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "duplicate key value") ||
		strings.Contains(msg, "unique constraint") ||
		(constraintHint != "" && strings.Contains(msg, strings.ToLower(constraintHint))) {
		return true
	}
	return false
}

func parseInt(s string, def int) int {
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	return def
}
func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func strPtr(s string) *string { return &s }
