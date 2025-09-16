package controller

import (
	"errors"
	"strings"
	"time"

	"masjidku_backend/internals/features/lembaga/teachers_students/dto"
	"masjidku_backend/internals/features/lembaga/teachers_students/model"
	helper "masjidku_backend/internals/helpers"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserTeacherController struct {
	DB       *gorm.DB
	Validate *validator.Validate
}

func NewUserTeacherController(db *gorm.DB, validate *validator.Validate) *UserTeacherController {
	return &UserTeacherController{DB: db, Validate: validate}
}

/* =========================
   Helpers (private)
========================= */

func (ctl *UserTeacherController) getFullName(userID uuid.UUID) string {
	var fullName string
	_ = ctl.DB.Raw(
		"SELECT full_name FROM users WHERE id = ? AND deleted_at IS NULL",
		userID,
	).Scan(&fullName).Error
	return fullName
}

func clampLimit(limit, def, max int) int {
	if limit <= 0 {
		return def
	}
	if limit > max {
		return max
	}
	return limit
}

func trimPtr(p *string) *string {
	if p == nil {
		return nil
	}
	s := strings.TrimSpace(*p)
	return &s
}

/* =========================
   CREATE
========================= */

func (ctl *UserTeacherController) Create(c *fiber.Ctx) error {
	var req dto.CreateUserTeacherRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.Validate.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Normalize whitespace di string-field
	req.UserTeacherField = strings.TrimSpace(req.UserTeacherField)
	req.UserTeacherShortBio = strings.TrimSpace(req.UserTeacherShortBio)
	req.UserTeacherLongBio = strings.TrimSpace(req.UserTeacherLongBio)
	req.UserTeacherGreeting = strings.TrimSpace(req.UserTeacherGreeting)
	req.UserTeacherEducation = strings.TrimSpace(req.UserTeacherEducation)
	req.UserTeacherActivity = strings.TrimSpace(req.UserTeacherActivity)

	// Cek duplikasi profil (unik per user)
	var exists int64
	if err := ctl.DB.Model(&model.UserTeacher{}).
		Where("user_teacher_user_id = ?", req.UserTeacherUserID).
		Count(&exists).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}
	if exists > 0 {
		return helper.JsonError(c, fiber.StatusConflict, "User sudah memiliki profil pengajar")
	}

	// Map DTO -> Model
	m := req.ToModel()

	if err := ctl.DB.WithContext(c.Context()).Create(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat profil pengajar")
	}

	resp := dto.ToUserTeacherResponse(m, ctl.getFullName(m.UserTeacherUserID))
	return helper.JsonCreated(c, "Profil pengajar berhasil dibuat", resp)
}

/* =========================
   LIST (q/active/verified + paging)
========================= */

func (ctl *UserTeacherController) List(c *fiber.Ctx) error {
	type Query struct {
		Q        string `query:"q"`
		Active   *bool  `query:"active"`
		Verified *bool  `query:"verified"`
		Limit    int    `query:"limit"`
		Offset   int    `query:"offset"`
	}
	var q Query
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}
	q.Limit = clampLimit(q.Limit, 20, 100)
	if q.Offset < 0 {
		q.Offset = 0
	}

	tx := ctl.DB.WithContext(c.Context()).Model(&model.UserTeacher{})

	if q.Active != nil {
		tx = tx.Where("user_teacher_is_active = ?", *q.Active)
	}
	if q.Verified != nil {
		tx = tx.Where("user_teacher_is_verified = ?", *q.Verified)
	}
	if s := strings.TrimSpace(q.Q); s != "" {
		// FTS + fallback ILIKE
		tx = tx.Where(`
			user_teacher_search @@ plainto_tsquery('simple', ?) OR
			user_teacher_field ILIKE ? OR
			user_teacher_short_bio ILIKE ? OR
			user_teacher_education ILIKE ?
		`, s, "%"+s+"%", "%"+s+"%", "%"+s+"%")
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	var rows []model.UserTeacher
	if err := tx.Order("user_teacher_created_at DESC").
		Limit(q.Limit).Offset(q.Offset).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	resps := make([]dto.UserTeacherResponse, 0, len(rows))
	for _, m := range rows {
		resps = append(resps, dto.ToUserTeacherResponse(m, ctl.getFullName(m.UserTeacherUserID)))
	}

	pagination := fiber.Map{
		"total":       total,
		"limit":       q.Limit,
		"offset":      q.Offset,
		"next_offset": q.Offset + q.Limit,
		"prev_offset": func() int {
			if q.Offset-q.Limit < 0 {
				return 0
			}
			return q.Offset - q.Limit
		}(),
		"returned":    len(resps),
		"server_time": time.Now().Format(time.RFC3339),
	}

	return helper.JsonList(c, resps, pagination)
}

/* =========================
   UPDATE (PATCH)
========================= */

func (ctl *UserTeacherController) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var req dto.UpdateUserTeacherRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.Validate.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var m model.UserTeacher
	if err := ctl.DB.WithContext(c.Context()).
		Where("user_teacher_id = ?", id).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	// Normalize whitespace pada string-pointer sebelum ApplyPatch
	req.UserTeacherField = trimPtr(req.UserTeacherField)
	req.UserTeacherShortBio = trimPtr(req.UserTeacherShortBio)
	req.UserTeacherLongBio = trimPtr(req.UserTeacherLongBio)
	req.UserTeacherGreeting = trimPtr(req.UserTeacherGreeting)
	req.UserTeacherEducation = trimPtr(req.UserTeacherEducation)
	req.UserTeacherActivity = trimPtr(req.UserTeacherActivity)

	// Terapkan patch (termasuk __clear → NULL)
	req.ApplyPatch(&m)

	// Save: karena kita pakai pointer utk field opsional, nilai yang tidak diubah tidak ter-overwrite
	if err := ctl.DB.WithContext(c.Context()).Save(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui data")
	}

	resp := dto.ToUserTeacherResponse(m, ctl.getFullName(m.UserTeacherUserID))
	return helper.JsonUpdated(c, "Profil pengajar berhasil diperbarui", resp)
}

/* =========================
   DELETE (soft)
========================= */

func (ctl *UserTeacherController) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	if err := ctl.DB.WithContext(c.Context()).
		Where("user_teacher_id = ?", id).
		Delete(&model.UserTeacher{}).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus data")
	}

	return helper.JsonDeleted(c, "Profil pengajar berhasil dihapus", fiber.Map{"user_teacher_id": id})
}
