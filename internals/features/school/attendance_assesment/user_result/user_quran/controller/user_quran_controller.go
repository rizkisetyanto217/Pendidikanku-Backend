// internals/features/lembaga/user_quran_records/controller/user_quran_record_controller.go
package controller

import (
	"errors"
	"strings"
	"time"

	"masjidku_backend/internals/features/school/attendance_assesment/user_result/user_quran/dto"
	model "masjidku_backend/internals/features/school/attendance_assesment/user_result/user_quran/model"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserQuranRecordController struct {
	DB        *gorm.DB
	Validator *validator.Validate
}

// NewUserQuranRecordController: panggil di wiring DI kamu
func NewUserQuranRecordController(db *gorm.DB) *UserQuranRecordController {
	return &UserQuranRecordController{
		DB:        db,
		Validator: validator.New(),
	}
}

const dateLayout = "2006-01-02"

// ===============================
// Helpers
// ===============================

// NOTE: ganti dengan helper asli kamu:
//   helper.GetMasjidIDFromToken(c)
func getMasjidIDFromToken(c *fiber.Ctx) (uuid.UUID, error) {
	// Placeholder untuk konsistensi contoh
	// Ambil dari Locals/Claims sesuai implementasi kamu
	if v := c.Locals("masjid_id"); v != nil {
		if s, ok := v.(string); ok {
			return uuid.Parse(s)
		}
	}
	return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
}

func parseUUIDParam(c *fiber.Ctx, name string) (uuid.UUID, error) {
	idStr := strings.TrimSpace(c.Params(name))
	return uuid.Parse(idStr)
}

func buildListQuery(tx *gorm.DB, q dto.ListUserQuranRecordQuery) (*gorm.DB, error) {
	// Filters
	if q.UserID != nil {
		tx = tx.Where("user_quran_records_user_id = ?", *q.UserID)
	}
	if q.SessionID != nil {
		tx = tx.Where("user_quran_records_session_id = ?", *q.SessionID)
	}
	if q.TeacherUser != nil {
		tx = tx.Where("user_quran_records_teacher_user_id = ?", *q.TeacherUser)
	}
	if q.SourceKind != nil && strings.TrimSpace(*q.SourceKind) != "" {
		tx = tx.Where("user_quran_records_source_kind = ?", strings.TrimSpace(*q.SourceKind))
	}
	// is_next: pakai IS NOT DISTINCT FROM biar NULL-aware
	if q.IsNext != nil {
		tx = tx.Where("user_quran_records_is_next IS NOT DISTINCT FROM ?", *q.IsNext)
	}
	// score range
	if q.ScoreMin != nil {
		tx = tx.Where("user_quran_records_score >= ?", *q.ScoreMin)
	}
	if q.ScoreMax != nil {
		tx = tx.Where("user_quran_records_score <= ?", *q.ScoreMax)
	}
	// created range
	if q.CreatedFrom != nil && strings.TrimSpace(*q.CreatedFrom) != "" {
		t, err := time.Parse(dateLayout, strings.TrimSpace(*q.CreatedFrom))
		if err != nil {
			return nil, fiber.NewError(fiber.StatusBadRequest, "created_from invalid format, expected YYYY-MM-DD")
		}
		tx = tx.Where("user_quran_records_created_at >= ?", t)
	}
	if q.CreatedTo != nil && strings.TrimSpace(*q.CreatedTo) != "" {
		t, err := time.Parse(dateLayout, strings.TrimSpace(*q.CreatedTo))
		if err != nil {
			return nil, fiber.NewError(fiber.StatusBadRequest, "created_to invalid format, expected YYYY-MM-DD")
		}
		tx = tx.Where("user_quran_records_created_at < ?", t.Add(24*time.Hour))
	}
	// search Q
	if q.Q != nil && strings.TrimSpace(*q.Q) != "" {
		needle := strings.TrimSpace(*q.Q)
		tx = tx.Where("user_quran_records_scope ILIKE ?", "%"+needle+"%")
	}

	// Sorting
	order := "user_quran_records_created_at DESC"
	if q.Sort != nil {
		switch strings.ToLower(strings.TrimSpace(*q.Sort)) {
		case "created_at_asc":
			order = "user_quran_records_created_at ASC"
		case "created_at_desc":
			order = "user_quran_records_created_at DESC"
		case "score_asc":
			order = "user_quran_records_score ASC NULLS LAST, user_quran_records_created_at DESC"
		case "score_desc":
			order = "user_quran_records_score DESC NULLS LAST, user_quran_records_created_at DESC"
		case "is_next_first":
			order = "user_quran_records_is_next DESC NULLS LAST, user_quran_records_created_at DESC"
		case "relevance":
			// optional, butuh q.Q; kalau kosong fallback default
			if q.Q != nil && strings.TrimSpace(*q.Q) != "" {
				needle := strings.TrimSpace(*q.Q)
				ord := "similarity(user_quran_records_scope, ?) DESC, user_quran_records_created_at DESC"
				tx = tx.Clauses(clause.Expr{SQL: "ORDER BY " + ord, Vars: []interface{}{needle}})
			}
		}
	}
	// apply order bila belum ada ORDER BY custom
	if _, ok := tx.Statement.Clauses["ORDER BY"]; !ok {
		tx = tx.Order(order)
	}

	return tx, nil
}

// ===============================
// Routes (contoh)
// ===============================
//
// POST   /api/a/user-quran-records
// GET    /api/a/user-quran-records
// GET    /api/a/user-quran-records/:id
// PATCH  /api/a/user-quran-records/:id
// DELETE /api/a/user-quran-records/:id
//
// NOTE: pastikan middleware auth ngisi masjid_id di Locals / token.

// ===============================
// Handlers
// ===============================

func (ctl *UserQuranRecordController) Create(c *fiber.Ctx) error {
	masjidID, err := getMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	var req dto.CreateUserQuranRecordRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.Validator.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	m := req.ToModel(masjidID)
	if err := ctl.DB.WithContext(c.Context()).Create(m).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"data": dto.NewUserQuranRecordResponse(m),
	})
}

func (ctl *UserQuranRecordController) GetByID(c *fiber.Ctx) error {
	masjidID, err := getMasjidIDFromToken(c)
	if err != nil {
		return err
	}
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	var m model.UserQuranRecordModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("user_quran_records_id = ? AND user_quran_records_masjid_id = ?", id, masjidID).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"data": dto.NewUserQuranRecordResponse(&m)})
}

func (ctl *UserQuranRecordController) List(c *fiber.Ctx) error {
	masjidID, err := getMasjidIDFromToken(c)
	if err != nil {
		return err
	}
	var q dto.ListUserQuranRecordQuery
	if err := c.QueryParser(&q); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Query tidak valid")
	}

	tx := ctl.DB.WithContext(c.Context()).Model(&model.UserQuranRecordModel{}).
		Where("user_quran_records_masjid_id = ?", masjidID)

	tx, err = buildListQuery(tx, q)
	if err != nil {
		// err sudah dalam bentuk fiber error (400) bila parsing salah
		return err
	}

	// Count total
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	limit := q.Limit
	offset := q.Offset
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	var rows []model.UserQuranRecordModel
	if err := tx.Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{
		"data":   dto.FromUserQuranRecordModels(rows),
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (ctl *UserQuranRecordController) Update(c *fiber.Ctx) error {
	masjidID, err := getMasjidIDFromToken(c)
	if err != nil {
		return err
	}
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	// Ambil existing (tenant-safe)
	var m model.UserQuranRecordModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("user_quran_records_id = ? AND user_quran_records_masjid_id = ?", id, masjidID).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	var req dto.UpdateUserQuranRecordRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.Validator.Struct(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Terapkan perubahan ke model
	req.ApplyToModel(&m)

	// Simpan (partial update)
	if err := ctl.DB.WithContext(c.Context()).
		Model(&m).
		Select(
			"user_quran_records_user_id",
			"user_quran_records_session_id",
			"user_quran_records_teacher_user_id",
			"user_quran_records_source_kind",
			"user_quran_records_scope",
			"user_quran_records_user_note",
			"user_quran_records_teacher_note",
			"user_quran_records_score",
			"user_quran_records_is_next",
			"user_quran_records_updated_at", // autoUpdateTime / trigger
		).
		Updates(&m).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{"data": dto.NewUserQuranRecordResponse(&m)})
}

func (ctl *UserQuranRecordController) Delete(c *fiber.Ctx) error {
	masjidID, err := getMasjidIDFromToken(c)
	if err != nil {
		return err
	}
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	// Soft delete tenant-safe
	if err := ctl.DB.WithContext(c.Context()).
		Where("user_quran_records_id = ? AND user_quran_records_masjid_id = ?", id, masjidID).
		Delete(&model.UserQuranRecordModel{}).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.SendStatus(fiber.StatusNoContent)
}
