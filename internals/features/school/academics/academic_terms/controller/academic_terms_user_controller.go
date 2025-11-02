// file: internals/features/lembaga/academics/academic_terms/controller/controller.go
package controller

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"schoolku_backend/internals/features/school/academics/academic_terms/dto"
	"schoolku_backend/internals/features/school/academics/academic_terms/model"
	helper "schoolku_backend/internals/helpers"
	helperAuth "schoolku_backend/internals/helpers/auth"
)

/* ================= Controller & Constructor ================= */

var fallbackValidator = validator.New()

/* ================= Handlers ================= */

// GET /api/a/academic-terms
// GET /api/u/:school_id/academic-terms
func (ctl *AcademicTermController) List(c *fiber.Ctx) error {
	// ===== School context (PUBLIC) =====
	mc, err := helperAuth.ResolveSchoolContext(c)
	if err != nil {
		return err
	}
	var schoolID uuid.UUID
	if mc.ID != uuid.Nil {
		schoolID = mc.ID
	} else if s := strings.TrimSpace(mc.Slug); s != "" {
		id, er := helperAuth.GetSchoolIDBySlug(c, s)
		if er != nil {
			if errors.Is(er, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, fiber.StatusNotFound, "School (slug) tidak ditemukan")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal resolve school dari slug")
		}
		schoolID = id
	} else {
		return helperAuth.ErrSchoolContextMissing
	}

	var q dto.AcademicTermFilterDTO
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid query: "+err.Error())
	}
	q.Normalize()

	v := ctl.Validator
	if v == nil {
		v = fallbackValidator
	}
	if err := v.Struct(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Validation failed: "+err.Error())
	}

	// ==== Pagination (helper) ====
	rawQ := string(c.Request().URI().QueryString())
	httpReq := &http.Request{URL: &url.URL{RawQuery: rawQ}}
	p := helper.ParseWith(httpReq, "created_at", "desc", helper.AdminOpts)

	// map sort_by -> kolom sebenarnya (singular)
	allowedSort := map[string]string{
		"created_at": "academic_term_created_at",
		"updated_at": "academic_term_updated_at",
		"start_date": "academic_term_start_date",
		"end_date":   "academic_term_end_date",
		"name":       "academic_term_name",
		"year":       "academic_term_academic_year",
		"angkatan":   "academic_term_angkatan",
		"code":       "academic_term_code",
		"slug":       "academic_term_slug",
	}
	orderClause, err := p.SafeOrderClause(allowedSort, "created_at")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "sort_by tidak valid")
	}
	orderExpr := strings.TrimPrefix(orderClause, "ORDER BY ")

	dbq := ctl.DB.Model(&model.AcademicTermModel{}).
		Where("academic_term_school_id = ? AND academic_term_deleted_at IS NULL", schoolID)

	// filter ID (robust)
	if rawID, has := c.Queries()["id"]; has {
		cleaned := strings.Trim(rawID, `"' {}`)
		low := strings.ToLower(strings.TrimSpace(cleaned))
		if cleaned != "" && low != "null" && low != "undefined" {
			if termID, err := uuid.Parse(cleaned); err == nil {
				dbq = dbq.Where("academic_term_id = ?", termID)
			}
		}
	}
	if q.Year != nil && strings.TrimSpace(*q.Year) != "" {
		dbq = dbq.Where("academic_term_academic_year = ?", strings.TrimSpace(*q.Year))
	}
	if q.Name != nil && strings.TrimSpace(*q.Name) != "" {
		dbq = dbq.Where("academic_term_name = ?", strings.TrimSpace(*q.Name))
	}
	if q.Code != nil && strings.TrimSpace(*q.Code) != "" {
		dbq = dbq.Where("academic_term_code = ?", strings.TrimSpace(*q.Code))
	}
	if q.Slug != nil && strings.TrimSpace(*q.Slug) != "" {
		dbq = dbq.Where("academic_term_slug = ?", strings.TrimSpace(*q.Slug))
	}
	if q.Active != nil {
		dbq = dbq.Where("academic_term_is_active = ?", *q.Active)
	}
	if q.Angkatan != nil {
		dbq = dbq.Where("academic_term_angkatan = ?", *q.Angkatan)
	}

	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Count failed: "+err.Error())
	}

	var list []model.AcademicTermModel
	if err := dbq.
		Order(orderExpr).
		Limit(p.Limit()).
		Offset(p.Offset()).
		Find(&list).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Query failed: "+err.Error())
	}

	meta := helper.BuildMeta(total, p)
	return helper.JsonList(c, dto.FromModels(list), meta)
}

/* ================= Extra: Search By Year (dengan openings & class) ================= */

type AcademicTermWithOpeningsResponse struct {
	dto.AcademicTermResponseDTO
	Openings []dto.OpeningWithClass `json:"openings"`
}

// GET /api/u/:school_id/academic-terms/search?year=2026&angkatan=10&per_page=20&page=1
func (ctl *AcademicTermController) SearchByYear(c *fiber.Ctx) error {
	yearQ := strings.TrimSpace(c.Query("year"))

	// ===== School context (PUBLIC) =====
	mc, err := helperAuth.ResolveSchoolContext(c)
	if err != nil {
		return err
	}
	var schoolID uuid.UUID
	if mc.ID != uuid.Nil {
		schoolID = mc.ID
	} else if s := strings.TrimSpace(mc.Slug); s != "" {
		id, er := helperAuth.GetSchoolIDBySlug(c, s)
		if er != nil {
			if errors.Is(er, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, fiber.StatusNotFound, "School (slug) tidak ditemukan")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal resolve school dari slug")
		}
		schoolID = id
	} else {
		return helperAuth.ErrSchoolContextMissing
	}

	// ==== Pagination (helper) ====
	rawQ := string(c.Request().URI().QueryString())
	httpReq := &http.Request{URL: &url.URL{RawQuery: rawQ}}
	p := helper.ParseWith(httpReq, "start_date", "desc", helper.AdminOpts)

	allowedSort := map[string]string{
		"start_date": "academic_term_start_date",
		"end_date":   "academic_term_end_date",
		"created_at": "academic_term_created_at",
		"updated_at": "academic_term_updated_at",
		"name":       "academic_term_name",
		"year":       "academic_term_academic_year",
		"angkatan":   "academic_term_angkatan",
		"code":       "academic_term_code",
		"slug":       "academic_term_slug",
	}
	orderClause, err := p.SafeOrderClause(allowedSort, "start_date")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "sort_by tidak valid")
	}
	orderExpr := strings.TrimPrefix(orderClause, "ORDER BY ")

	// filter angkatan (opsional)
	var angkatan *int
	if s := strings.TrimSpace(c.Query("angkatan")); s != "" {
		v, convErr := strconv.Atoi(s)
		if convErr != nil || v < 0 {
			return fiber.NewError(fiber.StatusBadRequest, "angkatan harus berupa angka >= 0")
		}
		angkatan = &v
	}

	dbq := ctl.DB.Model(&model.AcademicTermModel{}).
		Where("academic_term_school_id = ? AND academic_term_deleted_at IS NULL", schoolID)
	if yearQ != "" {
		dbq = dbq.Where("academic_term_academic_year ILIKE ?", "%"+yearQ+"%")
	}
	if angkatan != nil {
		dbq = dbq.Where("academic_term_angkatan = ?", *angkatan)
	}

	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Count failed: "+err.Error())
	}

	var list []model.AcademicTermModel
	if err := dbq.
		Order(orderExpr).
		Limit(p.Limit()).
		Offset(p.Offset()).
		Find(&list).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Query failed: "+err.Error())
	}

	// kumpulkan term_ids
	termIDs := make([]uuid.UUID, 0, len(list))
	for _, t := range list {
		termIDs = append(termIDs, t.AcademicTermID)
	}

	// map term_id -> openings[]
	openingMap := make(map[uuid.UUID][]dto.OpeningWithClass, len(list))
	if len(termIDs) > 0 {
		type row struct {
			// opening
			ClassTermOpeningsID                    uuid.UUID
			ClassTermOpeningsSchoolID              uuid.UUID
			ClassTermOpeningsClassID               uuid.UUID
			ClassTermOpeningsTermID                uuid.UUID
			ClassTermOpeningsIsOpen                bool
			ClassTermOpeningsRegistrationOpensAt   *time.Time
			ClassTermOpeningsRegistrationClosesAt  *time.Time
			ClassTermOpeningsQuotaTotal            *int
			ClassTermOpeningsQuotaTaken            int
			ClassTermOpeningsFeeOverrideMonthlyIDR *int
			ClassTermOpeningsNotes                 *string
			ClassTermOpeningsCreatedAt             time.Time
			ClassTermOpeningsUpdatedAt             *time.Time
			ClassTermOpeningsDeletedAt             *time.Time
			// class
			ClassID          uuid.UUID
			ClassSchoolID    *uuid.UUID
			ClassName        string
			ClassSlug        string
			ClassDescription *string
			ClassLevel       *string
			ClassImageURL    *string
			ClassIsActive    bool
		}

		var rows []row
		if err := ctl.DB.
			Table("class_term_openings AS o").
			Select(`
				o.class_term_openings_id, o.class_term_openings_school_id, o.class_term_openings_class_id,
				o.class_term_openings_term_id, o.class_term_openings_is_open,
				o.class_term_openings_registration_opens_at, o.class_term_openings_registration_closes_at,
				o.class_term_openings_quota_total, o.class_term_openings_quota_taken,
				o.class_term_openings_fee_override_monthly_idr, o.class_term_openings_notes,
				o.class_term_openings_created_at, o.class_term_openings_updated_at, o.class_term_openings_deleted_at,
				c.class_id, c.class_school_id, c.class_name, c.class_slug, c.class_description,
				c.class_level, c.class_image_url, c.class_is_active
			`).
			Joins("JOIN classes c ON c.class_id = o.class_term_openings_class_id AND c.class_deleted_at IS NULL").
			Where("o.class_term_openings_term_id IN (?)", termIDs).
			Where("o.class_term_openings_school_id = ?", schoolID).
			Where("o.class_term_openings_deleted_at IS NULL").
			Order("o.class_term_openings_created_at DESC").
			Find(&rows).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Query openings failed: "+err.Error())
		}

		for _, r := range rows {
			item := dto.OpeningWithClass{
				ClassTermOpeningsID:                    r.ClassTermOpeningsID,
				ClassTermOpeningsSchoolID:              r.ClassTermOpeningsSchoolID,
				ClassTermOpeningsClassID:               r.ClassTermOpeningsClassID,
				ClassTermOpeningsTermID:                r.ClassTermOpeningsTermID,
				ClassTermOpeningsIsOpen:                r.ClassTermOpeningsIsOpen,
				ClassTermOpeningsRegistrationOpensAt:   r.ClassTermOpeningsRegistrationOpensAt,
				ClassTermOpeningsRegistrationClosesAt:  r.ClassTermOpeningsRegistrationClosesAt,
				ClassTermOpeningsQuotaTotal:            r.ClassTermOpeningsQuotaTotal,
				ClassTermOpeningsQuotaTaken:            r.ClassTermOpeningsQuotaTaken,
				ClassTermOpeningsFeeOverrideMonthlyIDR: r.ClassTermOpeningsFeeOverrideMonthlyIDR,
				ClassTermOpeningsNotes:                 r.ClassTermOpeningsNotes,
				ClassTermOpeningsCreatedAt:             r.ClassTermOpeningsCreatedAt,
				ClassTermOpeningsUpdatedAt:             r.ClassTermOpeningsUpdatedAt,
				ClassTermOpeningsDeletedAt:             r.ClassTermOpeningsDeletedAt,
			}
			item.Class.ClassID = r.ClassID
			item.Class.ClassSchoolID = r.ClassSchoolID
			item.Class.ClassName = r.ClassName
			item.Class.ClassSlug = r.ClassSlug
			item.Class.ClassDescription = r.ClassDescription
			item.Class.ClassLevel = r.ClassLevel
			item.Class.ClassImageURL = r.ClassImageURL
			item.Class.ClassIsActive = r.ClassIsActive

			openingMap[r.ClassTermOpeningsTermID] = append(openingMap[r.ClassTermOpeningsTermID], item)
		}
	}

	termsDTO := dto.FromModels(list)
	out := make([]AcademicTermWithOpeningsResponse, 0, len(termsDTO))
	for i, t := range list {
		out = append(out, AcademicTermWithOpeningsResponse{
			AcademicTermResponseDTO: termsDTO[i],
			Openings:                openingMap[t.AcademicTermID],
		})
	}

	meta := helper.BuildMeta(total, p)
	return helper.JsonList(c, out, meta)
}
