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

	"masjidku_backend/internals/features/school/academics/academic_terms/dto"
	"masjidku_backend/internals/features/school/academics/academic_terms/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
)

/* ================= Controller & Constructor ================= */

var fallbackValidator = validator.New()

/* ================= Handlers ================= */

// GET /api/a/academic-terms
// List (multi-tenant via token) + Filter + Pagination + Sorting
// GET /api/u/:masjid_id/academic-terms  (atau kirim via header/query/subdomain)
func (ctl *AcademicTermController) List(c *fiber.Ctx) error {
	// ===== Masjid context (PUBLIC) =====
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	var masjidID uuid.UUID
	if mc.ID != uuid.Nil {
		masjidID = mc.ID
	} else if s := strings.TrimSpace(mc.Slug); s != "" {
		id, er := helperAuth.GetMasjidIDBySlug(c, s)
		if er != nil {
			if errors.Is(er, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, fiber.StatusNotFound, "Masjid (slug) tidak ditemukan")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal resolve masjid dari slug")
		}
		masjidID = id
	} else {
		return helperAuth.ErrMasjidContextMissing
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

	allowedSort := map[string]string{
		"created_at": "academic_terms_created_at",
		"updated_at": "academic_terms_updated_at",
		"start_date": "academic_terms_start_date",
		"end_date":   "academic_terms_end_date",
		"name":       "academic_terms_name",
		"year":       "academic_terms_academic_year",
		"angkatan":   "academic_terms_angkatan",
	}
	orderClause, err := p.SafeOrderClause(allowedSort, "created_at")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "sort_by tidak valid")
	}
	orderExpr := strings.TrimPrefix(orderClause, "ORDER BY ")

	dbq := ctl.DB.Model(&model.AcademicTermModel{}).
		Where("academic_terms_masjid_id = ? AND academic_terms_deleted_at IS NULL", masjidID)

	// filter ID (robust)
	if rawID, has := c.Queries()["id"]; has {
		cleaned := strings.Trim(rawID, `"' {}`)
		low := strings.ToLower(strings.TrimSpace(cleaned))
		if cleaned != "" && low != "null" && low != "undefined" {
			if termID, err := uuid.Parse(cleaned); err == nil {
				dbq = dbq.Where("academic_terms_id = ?", termID)
			}
		}
	}
	if q.Year != nil && strings.TrimSpace(*q.Year) != "" {
		dbq = dbq.Where("academic_terms_academic_year = ?", strings.TrimSpace(*q.Year))
	}
	if q.Name != nil && strings.TrimSpace(*q.Name) != "" {
		dbq = dbq.Where("academic_terms_name = ?", strings.TrimSpace(*q.Name))
	}
	if q.Active != nil {
		dbq = dbq.Where("academic_terms_is_active = ?", *q.Active)
	}
	if q.Angkatan != nil {
		dbq = dbq.Where("academic_terms_angkatan = ?", *q.Angkatan)
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

// Struktur join untuk response search

type AcademicTermWithOpeningsResponse struct {
	dto.AcademicTermResponseDTO
	Openings []dto.OpeningWithClass `json:"openings"`
}

// GET /api/u/:masjid_id/academic-terms/search?year=2026&angkatan=10&per_page=20&page=1
// (atau kirim konteks via header X-Active-Masjid-ID / ?masjid_id= / subdomain)
func (ctl *AcademicTermController) SearchByYear(c *fiber.Ctx) error {
	yearQ := strings.TrimSpace(c.Query("year"))

	// ===== Masjid context (PUBLIC) =====
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	var masjidID uuid.UUID
	if mc.ID != uuid.Nil {
		masjidID = mc.ID
	} else if s := strings.TrimSpace(mc.Slug); s != "" {
		id, er := helperAuth.GetMasjidIDBySlug(c, s)
		if er != nil {
			if errors.Is(er, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, fiber.StatusNotFound, "Masjid (slug) tidak ditemukan")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal resolve masjid dari slug")
		}
		masjidID = id
	} else {
		return helperAuth.ErrMasjidContextMissing
	}

	// ==== Pagination (helper) ====
	rawQ := string(c.Request().URI().QueryString())
	httpReq := &http.Request{URL: &url.URL{RawQuery: rawQ}}
	p := helper.ParseWith(httpReq, "start_date", "desc", helper.AdminOpts)

	allowedSort := map[string]string{
		"start_date": "academic_terms_start_date",
		"end_date":   "academic_terms_end_date",
		"created_at": "academic_terms_created_at",
		"updated_at": "academic_terms_updated_at",
		"name":       "academic_terms_name",
		"year":       "academic_terms_academic_year",
		"angkatan":   "academic_terms_angkatan",
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
		Where("academic_terms_masjid_id = ? AND academic_terms_deleted_at IS NULL", masjidID)
	if yearQ != "" {
		dbq = dbq.Where("academic_terms_academic_year ILIKE ?", "%"+yearQ+"%")
	}
	if angkatan != nil {
		dbq = dbq.Where("academic_terms_angkatan = ?", *angkatan)
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
		termIDs = append(termIDs, t.AcademicTermsID)
	}

	// map term_id -> openings[]
	openingMap := make(map[uuid.UUID][]dto.OpeningWithClass, len(list))
	if len(termIDs) > 0 {
		type row struct {
			// opening
			ClassTermOpeningsID                    uuid.UUID
			ClassTermOpeningsMasjidID              uuid.UUID
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
			ClassMasjidID    *uuid.UUID
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
				o.class_term_openings_id, o.class_term_openings_masjid_id, o.class_term_openings_class_id,
				o.class_term_openings_term_id, o.class_term_openings_is_open,
				o.class_term_openings_registration_opens_at, o.class_term_openings_registration_closes_at,
				o.class_term_openings_quota_total, o.class_term_openings_quota_taken,
				o.class_term_openings_fee_override_monthly_idr, o.class_term_openings_notes,
				o.class_term_openings_created_at, o.class_term_openings_updated_at, o.class_term_openings_deleted_at,
				c.class_id, c.class_masjid_id, c.class_name, c.class_slug, c.class_description,
				c.class_level, c.class_image_url, c.class_is_active
			`).
			Joins("JOIN classes c ON c.class_id = o.class_term_openings_class_id AND c.class_deleted_at IS NULL").
			Where("o.class_term_openings_term_id IN (?)", termIDs).
			Where("o.class_term_openings_masjid_id = ?", masjidID).
			Where("o.class_term_openings_deleted_at IS NULL").
			Order("o.class_term_openings_created_at DESC").
			Find(&rows).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Query openings failed: "+err.Error())
		}

		for _, r := range rows {
			item := dto.OpeningWithClass{
				ClassTermOpeningsID:                    r.ClassTermOpeningsID,
				ClassTermOpeningsMasjidID:              r.ClassTermOpeningsMasjidID,
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
			item.Class.ClassMasjidID = r.ClassMasjidID
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
			Openings:                openingMap[t.AcademicTermsID],
		})
	}

	meta := helper.BuildMeta(total, p)
	return helper.JsonList(c, out, meta)
}
