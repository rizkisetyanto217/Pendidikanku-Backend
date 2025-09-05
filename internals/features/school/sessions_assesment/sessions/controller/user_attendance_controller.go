// internals/features/school/attendance_assesment/user_result/user_attendance/controller/user_attendance_controller.go
package controller

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"time"

	attDTO "masjidku_backend/internals/features/school/sessions_assesment/sessions/dto"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	attModel "masjidku_backend/internals/features/school/sessions_assesment/sessions/model"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserAttendanceController struct {
	DB        *gorm.DB
	Validator *validator.Validate
}

func NewUserAttendanceController(db *gorm.DB) *UserAttendanceController {
	return &UserAttendanceController{
		DB:        db,
		Validator: validator.New(),
	}
}


// letakkan di file controller yang sama
func isDuplicateKey(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	// umumnya driver menuliskan salah satu dari ini
	return strings.Contains(s, "duplicate key") ||
		strings.Contains(s, "violates unique constraint") ||
		strings.Contains(s, "unique constraint") ||
		strings.Contains(s, "sqlstate 23505")
}


const dateLayout = "2006-01-02"

// ===============================
// Helpers
// ===============================

// Pastikan session milik masjid ini (tenant-safe)
func (ctl *UserAttendanceController) ensureSessionBelongsToMasjid(c *fiber.Ctx, sessionID, masjidID uuid.UUID) error {
	var count int64
	if err := ctl.DB.WithContext(c.Context()).
		Table("class_attendance_sessions").
		Where("class_attendance_sessions_id = ? AND class_attendance_sessions_masjid_id = ? AND class_attendance_sessions_deleted_at IS NULL",
			sessionID, masjidID).
		Count(&count).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if count == 0 {
		return fiber.NewError(fiber.StatusForbidden, "Session tidak ditemukan/diizinkan untuk masjid ini")
	}
	return nil
}

// Build list query with filters/sort (tenant-aware)
func (ctl *UserAttendanceController) buildListQuery(c *fiber.Ctx, q attDTO.ListUserAttendanceQuery, masjidID uuid.UUID) (*gorm.DB, error) {
	tx := ctl.DB.WithContext(c.Context()).Model(&attModel.UserAttendanceModel{}).
		Where("user_attendance_masjid_id = ?", masjidID)

	if q.SessionID != nil {
		tx = tx.Where("user_attendance_session_id = ?", *q.SessionID)
	}
	if q.UserID != nil {
		tx = tx.Where("user_attendance_user_id = ?", *q.UserID)
	}
	if q.Status != nil && strings.TrimSpace(*q.Status) != "" {
		s := strings.ToLower(strings.TrimSpace(*q.Status))
		switch s {
		case "present", "absent", "excused", "late":
			tx = tx.Where("user_attendance_status = ?", s)
		default:
			return nil, helper.JsonError(c, fiber.StatusBadRequest, "status tidak valid (present/absent/excused/late)")
		}
	}
	if q.CreatedFrom != nil && strings.TrimSpace(*q.CreatedFrom) != "" {
		t, err := time.Parse(dateLayout, strings.TrimSpace(*q.CreatedFrom))
		if err != nil {
			return nil, helper.JsonError(c, fiber.StatusBadRequest, "created_from invalid format, expected YYYY-MM-DD")
		}
		tx = tx.Where("user_attendance_created_at >= ?", t)
	}
	if q.CreatedTo != nil && strings.TrimSpace(*q.CreatedTo) != "" {
		t, err := time.Parse(dateLayout, strings.TrimSpace(*q.CreatedTo))
		if err != nil {
			return nil, helper.JsonError(c, fiber.StatusBadRequest, "created_to invalid format, expected YYYY-MM-DD")
		}
		tx = tx.Where("user_attendance_created_at < ?", t.Add(24*time.Hour))
	}

	order := "user_attendance_created_at DESC"
	if q.Sort != nil {
		switch strings.ToLower(strings.TrimSpace(*q.Sort)) {
		case "created_at_asc":
			order = "user_attendance_created_at ASC"
		case "created_at_desc":
			order = "user_attendance_created_at DESC"
		}
	}
	tx = tx.Order(order)
	return tx, nil
}

// ===============================
// Handlers
// ===============================

// POST /user-attendance
// POST /user-attendance
func (ctl *UserAttendanceController) Create(c *fiber.Ctx) error {
	// prefer TEACHER -> DKM -> MASJID_IDS -> ADMIN
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	body := bytes.TrimSpace(c.Body())
	if len(body) == 0 {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload kosong")
	}

	// =========================
	// BULK: jika payload berupa array JSON
	// =========================
	if body[0] == '[' {
		var reqs []attDTO.CreateUserAttendanceRequest
		if err := json.Unmarshal(body, &reqs); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid (array)")
		}
		if len(reqs) == 0 {
			return helper.JsonError(c, fiber.StatusBadRequest, "Items kosong")
		}

		// Validasi dan kumpulkan session unik
		sessions := make(map[uuid.UUID]struct{})
		for i := range reqs {
			if err := ctl.Validator.Struct(&reqs[i]); err != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
			}
			sessions[reqs[i].UserAttendanceSessionID] = struct{}{}
		}

		// Tenant guard per session unik
		for sid := range sessions {
			if err := ctl.ensureSessionBelongsToMasjid(c, sid, masjidID); err != nil {
				var fe *fiber.Error
				if errors.As(err, &fe) {
					return helper.JsonError(c, fe.Code, fe.Message)
				}
				return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
			}
		}

		// Build models
		models := make([]*attModel.UserAttendanceModel, 0, len(reqs))
		for i := range reqs {
			models = append(models, reqs[i].ToModel(masjidID))
		}

		// Insert in transaction (all-or-nothing)
		tx := ctl.DB.WithContext(c.Context()).Begin()
		if err := tx.CreateInBatches(models, 200).Error; err != nil {
			tx.Rollback()
			if isDuplicateKey(err) {
				return helper.JsonError(c, fiber.StatusConflict, "Beberapa kehadiran sudah tercatat (duplikat)")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
		}
		if err := tx.Commit().Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
		}

		// Build responses
		// sebelumnya: out := make([]attDTO.UserAttendanceResponse, 0, len(models))
		out := make([]*attDTO.UserAttendanceResponse, 0, len(models))
		for _, m := range models {
			out = append(out, attDTO.NewUserAttendanceResponse(m)) // New... mengembalikan *Response
		}
		return helper.JsonCreated(c, "Berhasil membuat kehadiran (bulk)", fiber.Map{
			"created": len(out),
			"items":   out,
		})

	}

	// =========================
	// SINGLE: payload object JSON (perilaku lama)
	// =========================
	var req attDTO.CreateUserAttendanceRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.Validator.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// tenant guard by session
	if err := ctl.ensureSessionBelongsToMasjid(c, req.UserAttendanceSessionID, masjidID); err != nil {
		var fe *fiber.Error
		if errors.As(err, &fe) {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	m := req.ToModel(masjidID)
	if err := ctl.DB.WithContext(c.Context()).Create(m).Error; err != nil {
		if isDuplicateKey(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Kehadiran sudah tercatat (duplikat)")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonCreated(c, "Berhasil membuat kehadiran", attDTO.NewUserAttendanceResponse(m))
}


// GET /user-attendance/:id
func (ctl *UserAttendanceController) GetByID(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var m attModel.UserAttendanceModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("user_attendance_id = ? AND user_attendance_masjid_id = ?", id, masjidID).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonOK(c, "OK", attDTO.NewUserAttendanceResponse(&m))
}

// GET /user-attendance
func (ctl *UserAttendanceController) List(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	var q attDTO.ListUserAttendanceQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}

	tx, err := ctl.buildListQuery(c, q, masjidID)
	if err != nil {
		// buildListQuery sudah return helperonse JSON error
		return err
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	limit := q.Limit
	offset := q.Offset
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	var rows []attModel.UserAttendanceModel
	if err := tx.Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonList(c, attDTO.FromUserAttendanceModels(rows), fiber.Map{
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// PATCH /user-attendance/:id
func (ctl *UserAttendanceController) Update(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var m attModel.UserAttendanceModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("user_attendance_id = ? AND user_attendance_masjid_id = ?", id, masjidID).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	var req attDTO.UpdateUserAttendanceRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.Validator.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Jika session diganti, cek kepemilikan tenant
	if req.UserAttendanceSessionID != nil {
		if err := ctl.ensureSessionBelongsToMasjid(c, *req.UserAttendanceSessionID, masjidID); err != nil {
			var fe *fiber.Error
			if errors.As(err, &fe) {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
		}
	}

	// Apply changes & save
	req.ApplyToModel(&m)
	if err := ctl.DB.WithContext(c.Context()).
		Model(&m).
		Select(
			"user_attendance_session_id",
			"user_attendance_user_id",
			"user_attendance_status",
			"user_attendance_user_note",
			"user_attendance_teacher_note",
			"user_attendance_updated_at",
		).
		Updates(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonUpdated(c, "Berhasil mengubah kehadiran", attDTO.NewUserAttendanceResponse(&m))
}

// DELETE /user-attendance/:id  (soft delete)
func (ctl *UserAttendanceController) Delete(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	if err := ctl.DB.WithContext(c.Context()).
		Where("user_attendance_id = ? AND user_attendance_masjid_id = ?", id, masjidID).
		Delete(&attModel.UserAttendanceModel{}).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// 200 OK dengan body (mengikuti helper JsonDeleted)
	return helper.JsonDeleted(c, "Berhasil menghapus kehadiran", fiber.Map{"id": id})
}
