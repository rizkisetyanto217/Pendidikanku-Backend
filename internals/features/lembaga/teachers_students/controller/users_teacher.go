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

type UsersTeacherController struct {
	DB       *gorm.DB
	Validate *validator.Validate
}

func NewUsersTeacherController(db *gorm.DB, validate *validator.Validate) *UsersTeacherController {
	return &UsersTeacherController{DB: db, Validate: validate}
}

/* =========================
   Helpers (private)
========================= */

func (ctl *UsersTeacherController) getFullName(userID uuid.UUID) string {
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

func (ctl *UsersTeacherController) Create(c *fiber.Ctx) error {
	var req dto.CreateUsersTeacherRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.Validate.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Normalize whitespace di string-field (sebelum ToModel -> NULL-kan jika kosong)
	req.UsersTeacherField = strings.TrimSpace(req.UsersTeacherField)
	req.UsersTeacherShortBio = strings.TrimSpace(req.UsersTeacherShortBio)
	req.UsersTeacherGreeting = strings.TrimSpace(req.UsersTeacherGreeting)
	req.UsersTeacherEducation = strings.TrimSpace(req.UsersTeacherEducation)
	req.UsersTeacherActivity = strings.TrimSpace(req.UsersTeacherActivity)

	// Cek duplikasi profil (unique per user)
	var count int64
	if err := ctl.DB.Model(&model.UserTeacher{}).
		Where("users_teacher_user_id = ? AND users_teacher_deleted_at IS NULL", req.UsersTeacherUserID).
		Count(&count).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}
	if count > 0 {
		return helper.JsonError(c, fiber.StatusConflict, "User sudah memiliki profil pengajar")
	}

	// Map DTO -> Model
	m := req.ToModel()

	if err := ctl.DB.WithContext(c.Context()).Create(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat profil pengajar")
	}

	resp := dto.ToUsersTeacherResponse(m, ctl.getFullName(m.UsersTeacherUserID))
	return helper.JsonCreated(c, "Profil pengajar berhasil dibuat", resp)
}

/* =========================
   LIST (q/active/verified + paging)
========================= */

func (ctl *UsersTeacherController) List(c *fiber.Ctx) error {
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
		tx = tx.Where("users_teacher_is_active = ?", *q.Active)
	}
	if q.Verified != nil {
		tx = tx.Where("users_teacher_is_verified = ?", *q.Verified)
	}
	if s := strings.TrimSpace(q.Q); s != "" {
		// gunakan FTS + fallback ILIKE
		tx = tx.Where(`
			users_teacher_search @@ plainto_tsquery('simple', ?) OR
			users_teacher_field ILIKE ? OR
			users_teacher_short_bio ILIKE ? OR
			users_teacher_education ILIKE ?
		`, s, "%"+s+"%", "%"+s+"%", "%"+s+"%")
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	var rows []model.UserTeacher
	if err := tx.Order("users_teacher_created_at DESC").
		Limit(q.Limit).Offset(q.Offset).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	resps := make([]dto.UsersTeacherResponse, 0, len(rows))
	for _, m := range rows {
		resps = append(resps, dto.ToUsersTeacherResponse(m, ctl.getFullName(m.UsersTeacherUserID)))
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

func (ctl *UsersTeacherController) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var req dto.UpdateUsersTeacherRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.Validate.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var m model.UserTeacher
	if err := ctl.DB.WithContext(c.Context()).
		Where("users_teacher_id = ?", id).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	// Normalize whitespace pada string-pointer sebelum ApplyPatch
	req.UsersTeacherField = trimPtr(req.UsersTeacherField)
	req.UsersTeacherShortBio = trimPtr(req.UsersTeacherShortBio)
	req.UsersTeacherGreeting = trimPtr(req.UsersTeacherGreeting)
	req.UsersTeacherEducation = trimPtr(req.UsersTeacherEducation)
	req.UsersTeacherActivity = trimPtr(req.UsersTeacherActivity)

	// Terapkan patch (termasuk __clear → NULL)
	req.ApplyPatch(&m)

	if err := ctl.DB.WithContext(c.Context()).Save(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui data")
	}

	resp := dto.ToUsersTeacherResponse(m, ctl.getFullName(m.UsersTeacherUserID))
	return helper.JsonUpdated(c, "Profil pengajar berhasil diperbarui", resp)
}

/* =========================
   DELETE (soft)
========================= */

func (ctl *UsersTeacherController) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	if err := ctl.DB.WithContext(c.Context()).
		Where("users_teacher_id = ?", id).
		Delete(&model.UserTeacher{}).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus data")
	}

	return helper.JsonDeleted(c, "Profil pengajar berhasil dihapus", fiber.Map{"users_teacher_id": id})
}
