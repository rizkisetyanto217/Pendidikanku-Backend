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

	dto "masjidku_backend/internals/features/school/sessions/sessions/dto"
	model "masjidku_backend/internals/features/school/sessions/sessions/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
	helperOSS "masjidku_backend/internals/helpers/oss"
)

// =========================================================
// Controller
// =========================================================

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

// ----------------- Attendance resolvers -----------------

// Ambil attendance_id dari query/path (baca beberapa alias)
// Ambil attendance_id dari query/path; boleh kosong -> uuid.Nil
func (ctl *UserAttendanceUrlController) resolveAttendanceIDOptional(c *fiber.Ctx) (uuid.UUID, bool, error) {
	cands := []string{
		strings.TrimSpace(c.Query("attendance_id")),
		strings.TrimSpace(c.Query("user_attendance_id")),
		strings.TrimSpace(c.Query("id")),
		strings.TrimSpace(c.Params("attendance_id")),
		strings.TrimSpace(c.Params("id")),
	}
	for _, raw := range cands {
		if raw == "" { continue }
		id, err := uuid.Parse(raw)
		if err != nil || id == uuid.Nil {
			return uuid.Nil, false, fiber.NewError(fiber.StatusBadRequest, "attendance_id tidak valid: "+raw)
		}
		return id, true, nil
	}
	return uuid.Nil, false, nil // tidak ada -> opsional
}


// Ambil attendance_id dari form (multipart) dengan beberapa alias field
func (ctl *UserAttendanceUrlController) resolveAttendanceIDFromForm(c *fiber.Ctx) (uuid.UUID, error) {
	cands := []string{
		strings.TrimSpace(c.FormValue("user_attendance_urls_attendance_id")),
		strings.TrimSpace(c.FormValue("attendance_id")),
		strings.TrimSpace(c.FormValue("user_attendance_id")),
	}
	var raw string
	for _, s := range cands {
		if s != "" { raw = s; break }
	}
	if raw == "" {
		return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "attendance_id (form) wajib (user_attendance_urls_attendance_id/attendance_id/user_attendance_id)")
	}
	id, err := uuid.Parse(raw)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "attendance_id (form) tidak valid: "+raw)
	}
	return id, nil
}

// Guard: pastikan attendance milik masjid & belum dihapus
func (ctl *UserAttendanceUrlController) guardAttendanceTenant(c *fiber.Ctx, attID, masjidID uuid.UUID) error {
	var ok bool
	if err := ctl.DB.WithContext(c.Context()).
		Raw(`SELECT EXISTS(
				SELECT 1 FROM user_attendance
				WHERE user_attendance_id = ? 
				  AND user_attendance_masjid_id = ?
				  AND user_attendance_deleted_at IS NULL
			)`, attID, masjidID).
		Scan(&ok).Error; err != nil {
		log.Println("[ERROR] guard attendance:", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal memeriksa attendance")
	}
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "Attendance tidak ditemukan / beda masjid")
	}
	return nil
}

// =========================================================
// LIST
// GET /api/a/user-attendance-urls?attendance_id=:attId&limit=20&offset=0
// =========================================================

func (ctl *UserAttendanceUrlController) ListByAttendance(c *fiber.Ctx) error {
	// ambil masjid_id prefer teacher
	mid, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil || mid == uuid.Nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}
	// authorize: anggota masjid (semua role)
	if err := helperAuth.EnsureMemberMasjid(c, mid); err != nil { return err }

	attID, hasAttID, err := ctl.resolveAttendanceIDOptional(c)
	if err != nil { return err }

	limit := clampInt(parseInt(c.Query("limit", "20"), 20), 1, 100)
	offset := maxInt(parseInt(c.Query("offset", "0"), 0), 0)

	// Base query
	tx := ctl.DB.WithContext(c.Context()).
		Where("user_attendance_urls_masjid_id = ? AND user_attendance_urls_deleted_at IS NULL", mid)

	// Jika ada attendance_id, guard & filter
	if hasAttID {
		if err := ctl.guardAttendanceTenant(c, attID, mid); err != nil {
			return err
		}
		tx = tx.Where("user_attendance_urls_attendance_id = ?", attID)
	}

	var rows []model.UserAttendanceURLModel
	if err := tx.Order("user_attendance_urls_created_at DESC").
		Limit(limit).Offset(offset).
		Find(&rows).Error; err != nil {
		log.Println("[ERROR] list user_attendance_urls:", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	out := make([]dto.UserAttendanceURLResponse, 0, len(rows))
	for _, r := range rows { out = append(out, dto.ToUserAttendanceURLResponse(r)) }
	return c.JSON(out)
}


// =========================================================
// CREATE - MULTIPART (upload file)
// POST /api/a/user-attendance-urls/multipart
// Form:
// - file (required) atau user_attendance_urls_href (file)
// - user_attendance_urls_attendance_id / attendance_id / user_attendance_id (uuid, required)
// - user_attendance_type_id (uuid, optional)
// - user_attendance_urls_label (optional)
// - user_attendance_urls_uploader_teacher_id (uuid, optional)
// - user_attendance_urls_uploader_student_id (uuid, optional)
// =========================================================
func (ctl *UserAttendanceUrlController) CreateMultipart(c *fiber.Ctx) error {
	// ambil masjid_id prefer teacher
	mid, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil || mid == uuid.Nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}
	// authorize: anggota masjid (semua role)
	if err := helperAuth.EnsureMemberMasjid(c, mid); err != nil { return err }

	attID, err := ctl.resolveAttendanceIDFromForm(c)
	if err != nil { return err }
	if err := ctl.guardAttendanceTenant(c, attID, mid); err != nil { return err }

	// optional fields
	var labelPtr *string
	if lbl := strings.TrimSpace(c.FormValue("user_attendance_urls_label")); lbl != "" {
		labelPtr = &lbl
	}
	var uploaderTeacherID *uuid.UUID
	if s := strings.TrimSpace(c.FormValue("user_attendance_urls_uploader_teacher_id")); s != "" {
		if v, e := uuid.Parse(s); e == nil { uploaderTeacherID = &v }
	}
	var uploaderStudentID *uuid.UUID
	if s := strings.TrimSpace(c.FormValue("user_attendance_urls_uploader_student_id")); s != "" {
		if v, e := uuid.Parse(s); e == nil { uploaderStudentID = &v }
	}
	var typeID *uuid.UUID
	if s := strings.TrimSpace(c.FormValue("user_attendance_type_id")); s != "" {
		if v, e := uuid.Parse(s); e == nil { typeID = &v }
	}

	// ambil file: support dua nama field
	fh, ferr := c.FormFile("user_attendance_urls_href")
	if ferr != nil || fh == nil {
		if alt, e2 := c.FormFile("file"); e2 == nil && alt != nil {
			fh, ferr = alt, nil
		}
	}
	if ferr != nil || fh == nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "File tidak ditemukan (pakai field 'file' atau 'user_attendance_urls_href')")
	}

	// init OSS
	svc, err := helperOSS.NewOSSServiceFromEnv("")
	if err != nil {
		log.Println("[OSS] init error:", err)
		return helper.JsonError(c, fiber.StatusBadGateway, "Gagal inisialisasi storage")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// upload apa adanya (bukan re-encode), ke: masjids/{masjid_id}/files/attendance/{attendance_id}/<file>
	dir := fmt.Sprintf("masjids/%s/files/attendance/%s", mid.String(), attID.String())
	key, _, upErr := svc.UploadFromFormFileToDir(ctx, dir, fh)
	if upErr != nil {
		// helper UploadFromFormFileToDir sudah return error detail
		return helper.JsonError(c, fiber.StatusBadGateway, "Gagal upload file")
	}
	publicURL := svc.PublicURL(key)

	// build & validate DTO
	req := dto.CreateUserAttendanceURLRequest{
		UserAttendanceURLsAttendanceID:      attID,
		UserAttendanceTypeID:                typeID,
		UserAttendanceURLsLabel:             labelPtr,
		UserAttendanceURLsHref:              publicURL,
		UserAttendanceURLsUploaderTeacherID: uploaderTeacherID,
		UserAttendanceURLsUploaderStudentID: uploaderStudentID,
	}
	if err := ctl.Validator.Struct(req); err != nil {
		_ = helperOSS.DeleteByPublicURLENV(publicURL, 10*time.Second) // cleanup
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// persist
	m := dto.NewUserAttendanceURLModelFromCreate(req, mid)
	if err := ctl.DB.WithContext(c.Context()).Create(&m).Error; err != nil {
		_ = helperOSS.DeleteByPublicURLENV(publicURL, 10*time.Second) // rollback file
		// nama index unik di migration: uq_uau_attendance_href_alive
		if isUniqueViolation(err, "uq_uau_attendance_href_alive") || isUniqueViolation(err, "uq_uau_attendance_href") {
			return helper.JsonError(c, http.StatusConflict, "URL sudah terdaftar untuk attendance tersebut")
		}
		log.Println("[ERROR] create(multipart) user_attendance_urls:", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat data")
	}

	// response standar
	return helper.JsonCreated(c, "Berhasil membuat URL lampiran attendance", dto.ToUserAttendanceURLResponse(m))
}



// =========================================================
// UPDATE - JSON / MULTIPART
// PATCH /api/a/user-attendance-urls/:id
// =========================================================
func (ctl *UserAttendanceUrlController) Update(c *fiber.Ctx) error {


		// ambil masjid_id prefer teacher
	mid, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil || mid == uuid.Nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}
	// authorize: anggota masjid (semua role)
	if err := helperAuth.EnsureMemberMasjid(c, mid); err != nil { return err }


	id, perr := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if perr != nil { return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid") }

	var m model.UserAttendanceURLModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("user_attendance_urls_id = ? AND user_attendance_urls_masjid_id = ? AND user_attendance_urls_deleted_at IS NULL",
			id, mid).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(http.StatusNotFound, "Data tidak ditemukan")
		}
		log.Println("[ERROR] fetch user_attendance_urls:", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	ct := strings.ToLower(strings.TrimSpace(c.Get(fiber.HeaderContentType)))
	isMultipart := strings.HasPrefix(ct, fiber.MIMEMultipartForm)

	if isMultipart {
		fh, ferr := c.FormFile("file")
		if ferr != nil || fh == nil {
			return fiber.NewError(fiber.StatusBadRequest, "File tidak ditemukan (field 'file')")
		}
		trashDays := 7
		if v := strings.TrimSpace(c.Query("trash_days")); v != "" {
			if d, e := strconv.Atoi(v); e == nil && d > 0 && d <= 365 { trashDays = d }
		}

		oldURL := m.UserAttendanceURLsHref
		svc, err := helperOSS.NewOSSServiceFromEnv("")
		if err != nil {
			log.Println("[OSS] init error:", err)
			return fiber.NewError(fiber.StatusBadGateway, "Gagal inisialisasi storage")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		slot := fmt.Sprintf("attendance/%s", m.UserAttendanceURLsAttendanceID.String())
		newURL, upErr := helperOSS.UploadImageToOSS(ctx, svc, mid, slot, fh)
		if upErr != nil { return upErr }

		now := time.Now()
		delAt := now.Add(time.Duration(trashDays) * 24 * time.Hour)
		m.UserAttendanceURLsTrashURL = strPtr(oldURL)
		m.UserAttendanceURLsDeletePendingUntil = &delAt
		m.UserAttendanceURLsHref = newURL
		m.UserAttendanceURLsUpdatedAt = now

		if err := ctl.DB.WithContext(c.Context()).Save(&m).Error; err != nil {
			_ = helperOSS.DeleteByPublicURLENV(newURL, 10*time.Second)
			if isUniqueViolation(err, "uq_uau_attendance_href") {
				return fiber.NewError(http.StatusConflict, "URL sudah terdaftar untuk attendance tersebut")
			}
			log.Println("[ERROR] update(multipart) user_attendance_urls:", err)
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui data")
		}
		return c.JSON(dto.ToUserAttendanceURLResponse(m))
	}

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

// =========================================================
// DELETE (soft)
// =========================================================
func (ctl *UserAttendanceUrlController) SoftDelete(c *fiber.Ctx) error {

	// ambil masjid_id prefer teacher
	mid, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil || mid == uuid.Nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}
	// authorize: anggota masjid (semua role)
	if err := helperAuth.EnsureMemberMasjid(c, mid); err != nil { return err }

	id, perr := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if perr != nil { return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid") }

	var m model.UserAttendanceURLModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("user_attendance_urls_id = ? AND user_attendance_urls_masjid_id = ? AND user_attendance_urls_deleted_at IS NULL",
			id, mid).
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

// =========================================================
// Helpers
// =========================================================

func isUniqueViolation(err error, constraintHint string) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) { return true }
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate key value") ||
		strings.Contains(msg, "unique constraint") ||
		(constraintHint != "" && strings.Contains(msg, strings.ToLower(constraintHint)))
}

func parseInt(s string, def int) int { if v, err := strconv.Atoi(s); err == nil { return v }; return def }
func clampInt(v, lo, hi int) int { if v < lo { return lo }; if v > hi { return hi }; return v }
func maxInt(a, b int) int { if a > b { return a }; return b }
func strPtr(s string) *string { return &s }
