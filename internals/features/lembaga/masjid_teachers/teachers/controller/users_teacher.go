package controller

import (
	"errors"
	"strings"
	"time"

	"masjidku_backend/internals/features/lembaga/masjid_teachers/teachers/dto"
	"masjidku_backend/internals/features/lembaga/masjid_teachers/teachers/model"
	helper "masjidku_backend/internals/helpers"

	// <— SESUAIKAN path helper-nya
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

	// Cek duplikasi profil
	var count int64
	if err := ctl.DB.Model(&model.UsersTeacherModel{}).
		Where("users_teacher_user_id = ? AND users_teacher_deleted_at IS NULL", req.UserID).
		Count(&count).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}
	if count > 0 {
		return helper.JsonError(c, fiber.StatusConflict, "User sudah memiliki profil pengajar")
	}

	m := model.UsersTeacherModel{
		UserID:          req.UserID,
		Field:           strings.TrimSpace(req.Field),
		ShortBio:        strings.TrimSpace(req.ShortBio),
		Greeting:        strings.TrimSpace(req.Greeting),
		Education:       strings.TrimSpace(req.Education),
		Activity:        strings.TrimSpace(req.Activity),
		ExperienceYears: req.ExperienceYears,
		Specialties:     req.Specialties,
		Certificates:    req.Certificates,
		Links:           req.Links,
		IsVerified:      false,
		IsActive:        true,
	}
	if req.IsVerified != nil {
		m.IsVerified = *req.IsVerified
	}
	if req.IsActive != nil {
		m.IsActive = *req.IsActive
	}

	if err := ctl.DB.WithContext(c.Context()).Create(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat profil pengajar")
	}

	resp := dto.ToUsersTeacherResponse(m, ctl.getFullName(m.UserID))
	return helper.JsonCreated(c, "Profil pengajar berhasil dibuat", resp)
}

/* =========================
   GET BY ID
========================= */

func (ctl *UsersTeacherController) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var m model.UsersTeacherModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("users_teacher_id = ?", id).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	resp := dto.ToUsersTeacherResponse(m, ctl.getFullName(m.UserID))
	return helper.JsonOK(c, "OK", resp)
}

/* =========================
   GET BY USER ID
========================= */

func (ctl *UsersTeacherController) GetByUserID(c *fiber.Ctx) error {
	userID, err := uuid.Parse(strings.TrimSpace(c.Params("user_id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "User ID tidak valid")
	}

	var m model.UsersTeacherModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("users_teacher_user_id = ?", userID).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	resp := dto.ToUsersTeacherResponse(m, ctl.getFullName(m.UserID))
	return helper.JsonOK(c, "OK", resp)
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

	tx := ctl.DB.WithContext(c.Context()).Model(&model.UsersTeacherModel{})

	if q.Active != nil {
		tx = tx.Where("users_teacher_is_active = ?", *q.Active)
	}
	if q.Verified != nil {
		tx = tx.Where("users_teacher_is_verified = ?", *q.Verified)
	}
	if s := strings.TrimSpace(q.Q); s != "" {
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

	var rows []model.UsersTeacherModel
	if err := tx.Order("users_teacher_created_at DESC").
		Limit(q.Limit).Offset(q.Offset).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	resps := make([]dto.UsersTeacherResponse, 0, len(rows))
	for _, m := range rows {
		resps = append(resps, dto.ToUsersTeacherResponse(m, ctl.getFullName(m.UserID)))
	}

	pagination := fiber.Map{
		"total":        total,
		"limit":        q.Limit,
		"offset":       q.Offset,
		"next_offset":  q.Offset + q.Limit,
		"prev_offset":  func() int { if q.Offset-q.Limit < 0 { return 0 }; return q.Offset - q.Limit }(),
		"returned":     len(resps),
		"server_time":  time.Now().Format(time.RFC3339),
	}

	return helper.JsonList(c, resps, pagination)
}

/* =========================
   UPDATE (partial)
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

	var m model.UsersTeacherModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("users_teacher_id = ?", id).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	// apply partial
	if req.Field != nil {
		m.Field = strings.TrimSpace(*req.Field)
	}
	if req.ShortBio != nil {
		m.ShortBio = strings.TrimSpace(*req.ShortBio)
	}
	if req.Greeting != nil {
		m.Greeting = strings.TrimSpace(*req.Greeting)
	}
	if req.Education != nil {
		m.Education = strings.TrimSpace(*req.Education)
	}
	if req.Activity != nil {
		m.Activity = strings.TrimSpace(*req.Activity)
	}
	if req.ExperienceYears != nil {
		m.ExperienceYears = req.ExperienceYears
	}
	if req.Specialties != nil {
		m.Specialties = *req.Specialties
	}
	if req.Certificates != nil {
		m.Certificates = *req.Certificates
	}
	if req.Links != nil {
		m.Links = *req.Links
	}
	if req.IsVerified != nil {
		m.IsVerified = *req.IsVerified
	}
	if req.IsActive != nil {
		m.IsActive = *req.IsActive
	}

	if err := ctl.DB.WithContext(c.Context()).Save(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui data")
	}

	resp := dto.ToUsersTeacherResponse(m, ctl.getFullName(m.UserID))
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
		Delete(&model.UsersTeacherModel{}).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus data")
	}

	// kirim body (200) agar konsisten format response
	return helper.JsonDeleted(c, "Profil pengajar berhasil dihapus", fiber.Map{"users_teacher_id": id})
}
