// file: internals/features/classes/openings/controller/class_term_opening_controller.go
package controller

import (
	"strings"
	"time"

	"masjidku_backend/internals/features/school/academics/academic_terms/dto"
	"masjidku_backend/internals/features/school/academics/academic_terms/model"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	helper "masjidku_backend/internals/helpers"
)

type ClassTermOpeningController struct {
	DB       *gorm.DB
	Validate *validator.Validate
}

func NewClassTermOpeningController(db *gorm.DB) *ClassTermOpeningController {
	return &ClassTermOpeningController{
		DB:       db,
		Validate: validator.New(),
	}
}

// ============================
// Helpers
// ============================

func parseUUIDParam(c *fiber.Ctx, key string) (uuid.UUID, error) {
	idStr := c.Params(key)
	return uuid.Parse(idStr)
}

func toResponse(m model.ClassTermOpeningModel) dto.ClassTermOpeningResponse {
	return dto.ClassTermOpeningResponse{
		ClassTermOpeningsID:                    m.ClassTermOpeningsID,
		ClassTermOpeningsMasjidID:              m.ClassTermOpeningsMasjidID,
		ClassTermOpeningsClassID:               m.ClassTermOpeningsClassID,
		ClassTermOpeningsTermID:                m.ClassTermOpeningsTermID,
		ClassTermOpeningsIsOpen:                m.ClassTermOpeningsIsOpen,
		ClassTermOpeningsRegistrationOpensAt:   m.ClassTermOpeningsRegistrationOpensAt,
		ClassTermOpeningsRegistrationClosesAt:  m.ClassTermOpeningsRegistrationClosesAt,
		ClassTermOpeningsQuotaTotal:            m.ClassTermOpeningsQuotaTotal,
		ClassTermOpeningsQuotaTaken:            m.ClassTermOpeningsQuotaTaken,
		ClassTermOpeningsFeeOverrideMonthlyIDR: m.ClassTermOpeningsFeeOverrideMonthlyIDR,
		ClassTermOpeningsNotes:                 m.ClassTermOpeningsNotes,
		ClassTermOpeningsCreatedAt:             m.ClassTermOpeningsCreatedAt,
		ClassTermOpeningsUpdatedAt:             m.ClassTermOpeningsUpdatedAt,
		ClassTermOpeningsDeletedAt:             m.ClassTermOpeningsDeletedAt,
	}
}

func applyUpdate(dst *model.ClassTermOpeningModel, req dto.UpdateClassTermOpeningRequest) {
	now := time.Now()
	dst.ClassTermOpeningsUpdatedAt = &now

	if req.ClassTermOpeningsIsOpen != nil {
		dst.ClassTermOpeningsIsOpen = *req.ClassTermOpeningsIsOpen
	}
	if req.ClassTermOpeningsRegistrationOpensAt != nil {
		dst.ClassTermOpeningsRegistrationOpensAt = req.ClassTermOpeningsRegistrationOpensAt
	}
	if req.ClassTermOpeningsRegistrationClosesAt != nil {
		dst.ClassTermOpeningsRegistrationClosesAt = req.ClassTermOpeningsRegistrationClosesAt
	}
	if req.ClassTermOpeningsQuotaTotal != nil {
		dst.ClassTermOpeningsQuotaTotal = req.ClassTermOpeningsQuotaTotal
	}
	if req.ClassTermOpeningsFeeOverrideMonthlyIDR != nil {
		dst.ClassTermOpeningsFeeOverrideMonthlyIDR = req.ClassTermOpeningsFeeOverrideMonthlyIDR
	}
	if req.ClassTermOpeningsNotes != nil {
		dst.ClassTermOpeningsNotes = req.ClassTermOpeningsNotes
	}
}

// ============================
// CRUD
// ============================


// POST /api/a/class-term-openings
func (ct *ClassTermOpeningController) CreateClassTermOpening(c *fiber.Ctx) error {
	var req dto.CreateClassTermOpeningRequest

	// 1) Bind body
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body: "+err.Error())
	}

	// 2) Ambil masjid_id dari token (prefer: teacher -> dkm -> union -> admin)
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err // sudah fiber.Error dari helper
	}

	// 3) (Optional strict) Tolak jika body mengirim masjid_id yang beda dengan token
	if req.ClassTermOpeningsMasjidID != uuid.Nil && req.ClassTermOpeningsMasjidID != masjidID {
		return fiber.NewError(fiber.StatusForbidden, "masjid_id di body tidak sesuai dengan token")
	}

	// 4) Override masjid_id agar anti-spoof & lolos validator "required"
	req.ClassTermOpeningsMasjidID = masjidID

	// 5) Validate DTO
	if err := ct.Validate.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// 6) Map DTO -> Model & set default
	now := time.Now()
	m := model.ClassTermOpeningModel{
		ClassTermOpeningsMasjidID:              req.ClassTermOpeningsMasjidID,
		ClassTermOpeningsClassID:               req.ClassTermOpeningsClassID,
		ClassTermOpeningsTermID:                req.ClassTermOpeningsTermID,
		ClassTermOpeningsIsOpen:                true, // default:true, override di bawah kalau dikirim
		ClassTermOpeningsRegistrationOpensAt:   req.ClassTermOpeningsRegistrationOpensAt,
		ClassTermOpeningsRegistrationClosesAt:  req.ClassTermOpeningsRegistrationClosesAt,
		ClassTermOpeningsQuotaTotal:            req.ClassTermOpeningsQuotaTotal,
		ClassTermOpeningsQuotaTaken:            0,
		ClassTermOpeningsFeeOverrideMonthlyIDR: req.ClassTermOpeningsFeeOverrideMonthlyIDR,
		ClassTermOpeningsNotes:                 req.ClassTermOpeningsNotes,
		ClassTermOpeningsCreatedAt:             now,
	}
	if req.ClassTermOpeningsIsOpen != nil {
		m.ClassTermOpeningsIsOpen = *req.ClassTermOpeningsIsOpen
	}

	// 7) Insert
	if err := ct.DB.Create(&m).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "create failed: "+err.Error())
	}

	// 8) Response
	return c.Status(fiber.StatusCreated).JSON(toResponse(m))
}



// GET /api/a/class-term-openings
// Query optional: masjid_id, class_id, term_id, is_open, include_deleted, page, limit, sort (e.g. "created_at desc")
func (ct *ClassTermOpeningController) GetAllClassTermOpenings(c *fiber.Ctx) error {
	db := ct.DB.Model(&model.ClassTermOpeningModel{})

	// filters
	if v := c.Query("masjid_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			db = db.Where("class_term_openings_masjid_id = ?", id)
		} else {
			return fiber.NewError(fiber.StatusBadRequest, "invalid masjid_id")
		}
	}
	if v := c.Query("class_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			db = db.Where("class_term_openings_class_id = ?", id)
		} else {
			return fiber.NewError(fiber.StatusBadRequest, "invalid class_id")
		}
	}
	if v := c.Query("term_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			db = db.Where("class_term_openings_term_id = ?", id)
		} else {
			return fiber.NewError(fiber.StatusBadRequest, "invalid term_id")
		}
	}
	if v := c.Query("is_open"); v != "" {
		low := strings.ToLower(v)
		if low == "true" || low == "1" {
			db = db.Where("class_term_openings_is_open = true")
		} else if low == "false" || low == "0" {
			db = db.Where("class_term_openings_is_open = false")
		} else {
			return fiber.NewError(fiber.StatusBadRequest, "invalid is_open")
		}
	}
	includeDeleted := strings.ToLower(c.Query("include_deleted")) == "true"

	// pagination
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)
	if page < 1 {
		page = 1
	}
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	offset := (page - 1) * limit

	// count
	var total int64
	countDB := db
	if !includeDeleted {
		countDB = countDB.Where("class_term_openings_deleted_at IS NULL")
	}
	if err := countDB.Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "count failed: "+err.Error())
	}

	// sorting
	sort := c.Query("sort", "class_term_openings_created_at desc")
	if sort != "" {
		db = db.Order(sort)
	}

	if !includeDeleted {
		db = db.Where("class_term_openings_deleted_at IS NULL")
	}

	var rows []model.ClassTermOpeningModel
	if err := db.Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "query failed: "+err.Error())
	}

	res := make([]dto.ClassTermOpeningResponse, 0, len(rows))
	for _, r := range rows {
		res = append(res, toResponse(r))
	}

	return c.JSON(fiber.Map{
		"data":       res,
		"pagination": fiber.Map{"page": page, "limit": limit, "total": total},
	})
}

// GET /api/a/class-term-openings/:id
func (ct *ClassTermOpeningController) GetClassTermOpeningByID(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	var m model.ClassTermOpeningModel
	if err := ct.DB.First(&m, "class_term_openings_id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "query failed: "+err.Error())
	}
	return c.JSON(toResponse(m))
}


// PUT /api/a/class-term-openings/:id
func (ct *ClassTermOpeningController) UpdateClassTermOpening(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	// Ambil masjid_id dari token (prefer teacher -> dkm -> union -> admin)
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err // sudah fiber.Error
	}

	var req dto.UpdateClassTermOpeningRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body: "+err.Error())
	}

	// Ambil record hanya milik masjid dari token
	var m model.ClassTermOpeningModel
	if err := ct.DB.First(&m,
		"class_term_openings_id = ? AND class_term_openings_masjid_id = ?",
		id, masjidID,
	).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "query failed: "+err.Error())
	}

	applyUpdate(&m, req)

	if err := ct.DB.Save(&m).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "update failed: "+err.Error())
	}
	return c.JSON(toResponse(m))
}

// DELETE /api/a/class-term-openings/:id  (soft delete, scoped masjid dari token)
func (ct *ClassTermOpeningController) DeleteClassTermOpening(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	// Ambil masjid_id dari token (prefer teacher -> dkm -> union -> admin)
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err // sudah fiber.Error
	}

	// Ambil record hanya milik masjid dari token
	var m model.ClassTermOpeningModel
	if err := ct.DB.First(&m,
		"class_term_openings_id = ? AND class_term_openings_masjid_id = ?",
		id, masjidID,
	).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "query failed: "+err.Error())
	}

	now := time.Now()
	m.ClassTermOpeningsDeletedAt = &now
	m.ClassTermOpeningsUpdatedAt = &now

	if err := ct.DB.Save(&m).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "delete failed: "+err.Error())
	}
	return c.Status(fiber.StatusNoContent).Send(nil)
}


// Optional: RESTORE (kalau butuh)
// POST /api/a/class-term-openings/:id/restore
func (ct *ClassTermOpeningController) RestoreClassTermOpening(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	var m model.ClassTermOpeningModel
	if err := ct.DB.First(&m, "class_term_openings_id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "query failed: "+err.Error())
	}

	m.ClassTermOpeningsDeletedAt = nil
	now := time.Now()
	m.ClassTermOpeningsUpdatedAt = &now

	if err := ct.DB.Save(&m).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "restore failed: "+err.Error())
	}
	return c.JSON(toResponse(m))
}