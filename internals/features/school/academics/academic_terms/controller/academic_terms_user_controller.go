// file: internals/features/lembaga/academics/academic_terms/controller/controller.go
package controller

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"masjidku_backend/internals/features/school/academics/academic_terms/dto"
	"masjidku_backend/internals/features/school/academics/academic_terms/model"
	helperAuth "masjidku_backend/internals/helpers/auth"
	helper "masjidku_backend/internals/helpers"
)

/* ================= Controller & Constructor ================= */


var fallbackValidator = validator.New()

/* ================= Handlers ================= */

// GET /api/a/academic-terms/:id
// GetByID (scoped ke masjid di token)
func (ctl *AcademicTermController) GetByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid id")
	}

	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	var ent model.AcademicTermModel
	if err := ctl.DB.
		Where("academic_terms_id = ? AND academic_terms_masjid_id = ? AND academic_terms_deleted_at IS NULL", id, masjidID).
		First(&ent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Record not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Query failed: "+err.Error())
	}

	return helper.JsonOK(c, "Academic term fetched successfully", dto.FromModel(ent))
}

// GET /api/a/academic-terms
// List (multi-tenant via token) + Filter + Pagination + Sorting
func (ctl *AcademicTermController) List(c *fiber.Ctx) error {
	var q dto.AcademicTermFilterDTO
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid query: "+err.Error())
	}
	q.Normalize()

	// validate
	v := ctl.Validator
	if v == nil {
		v = fallbackValidator
	}
	if err := v.Struct(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Validation failed: "+err.Error())
	}

	// multi-tenant
	masjidIDs, err := helperAuth.GetMasjidIDsFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	// paging default
	page := q.Page
	if page == 0 {
		page = 1
	}
	pageSize := q.PageSize
	if pageSize == 0 {
		pageSize = 20
	}

	dbq := ctl.DB.Model(&model.AcademicTermModel{}).
		Where("academic_terms_masjid_id IN (?) AND academic_terms_deleted_at IS NULL", masjidIDs)

	// filter by ID (robust: abaikan nilai kosong/invalid, jangan 400)
	if rawID, has := c.Queries()["id"]; has {
		cleaned := strings.TrimSpace(rawID)
		cleaned = strings.Trim(cleaned, `"' {}`) // bersihkan kutip/kurung nyasar
		low := strings.ToLower(cleaned)
		if cleaned != "" && low != "null" && low != "undefined" {
			if termID, err := uuid.Parse(cleaned); err == nil {
				dbq = dbq.Where("academic_terms_id = ?", termID)
			}
			// else: biarkan, jangan return 400 agar "get all" tetap jalan
		}
	}

	// filters lain
	if q.Year != nil && strings.TrimSpace(*q.Year) != "" {
		dbq = dbq.Where("academic_terms_academic_year = ?", strings.TrimSpace(*q.Year))
	}
	if q.Name != nil && strings.TrimSpace(*q.Name) != "" {
		dbq = dbq.Where("academic_terms_name = ?", strings.TrimSpace(*q.Name))
	}
	if q.Active != nil {
		dbq = dbq.Where("academic_terms_is_active = ?", *q.Active)
	}

	// sorting
	sortBy := "academic_terms_created_at"
	if q.SortBy != nil {
		switch *q.SortBy {
		case "created_at":
			sortBy = "academic_terms_created_at"
		case "updated_at":
			sortBy = "academic_terms_updated_at"
		case "start_date":
			sortBy = "academic_terms_start_date"
		case "end_date":
			sortBy = "academic_terms_end_date"
		case "name":
			sortBy = "academic_terms_name"
		case "year":
			sortBy = "academic_terms_academic_year"
		}
	}
	sortDir := "desc"
	if q.SortDir != nil && (*q.SortDir == "asc" || *q.SortDir == "desc") {
		sortDir = *q.SortDir
	}

	// total
	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Count failed: "+err.Error())
	}

	// data
	var list []model.AcademicTermModel
	if err := dbq.
		Order(sortBy + " " + sortDir).
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&list).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Query failed: "+err.Error())
	}

	return helper.JsonList(c, dto.FromModels(list), fiber.Map{
		"total": total, "page": page, "page_size": pageSize,
	})
}

/* ================= Extra: Search By Year (dengan openings & class) ================= */

// Struktur join untuk response search
type OpeningWithClass struct {
	// opening
	ClassTermOpeningsID                    uuid.UUID  `json:"class_term_openings_id"                      gorm:"column:class_term_openings_id"`
	ClassTermOpeningsMasjidID              uuid.UUID  `json:"class_term_openings_masjid_id"               gorm:"column:class_term_openings_masjid_id"`
	ClassTermOpeningsClassID               uuid.UUID  `json:"class_term_openings_class_id"                gorm:"column:class_term_openings_class_id"`
	ClassTermOpeningsTermID                uuid.UUID  `json:"class_term_openings_term_id"                 gorm:"column:class_term_openings_term_id"`
	ClassTermOpeningsIsOpen                bool       `json:"class_term_openings_is_open"                 gorm:"column:class_term_openings_is_open"`
	ClassTermOpeningsRegistrationOpensAt   *time.Time `json:"class_term_openings_registration_opens_at"   gorm:"column:class_term_openings_registration_opens_at"`
	ClassTermOpeningsRegistrationClosesAt  *time.Time `json:"class_term_openings_registration_closes_at"  gorm:"column:class_term_openings_registration_closes_at"`
	ClassTermOpeningsQuotaTotal            *int       `json:"class_term_openings_quota_total"             gorm:"column:class_term_openings_quota_total"`
	ClassTermOpeningsQuotaTaken            int        `json:"class_term_openings_quota_taken"             gorm:"column:class_term_openings_quota_taken"`
	ClassTermOpeningsFeeOverrideMonthlyIDR *int       `json:"class_term_openings_fee_override_monthly_idr" gorm:"column:class_term_openings_fee_override_monthly_idr"`
	ClassTermOpeningsNotes                 *string    `json:"class_term_openings_notes"                   gorm:"column:class_term_openings_notes"`
	ClassTermOpeningsCreatedAt             time.Time  `json:"class_term_openings_created_at"              gorm:"column:class_term_openings_created_at"`
	ClassTermOpeningsUpdatedAt             *time.Time `json:"class_term_openings_updated_at"              gorm:"column:class_term_openings_updated_at"`
	ClassTermOpeningsDeletedAt             *time.Time `json:"class_term_openings_deleted_at"              gorm:"column:class_term_openings_deleted_at"`

	// class (subset)
	Class struct {
		ClassID            uuid.UUID  `json:"class_id"              gorm:"column:class_id"`
		ClassMasjidID      *uuid.UUID `json:"class_masjid_id"       gorm:"column:class_masjid_id"`
		ClassName          string     `json:"class_name"            gorm:"column:class_name"`
		ClassSlug          string     `json:"class_slug"            gorm:"column:class_slug"`
		ClassDescription   *string    `json:"class_description"     gorm:"column:class_description"`
		ClassLevel         *string    `json:"class_level"           gorm:"column:class_level"`
		ClassImageURL      *string    `json:"class_image_url"       gorm:"column:class_image_url"`
		ClassIsActive      bool       `json:"class_is_active"       gorm:"column:class_is_active"`
	} `json:"class"`
}

type AcademicTermWithOpeningsResponse struct {
	dto.AcademicTermResponseDTO
	Openings []OpeningWithClass `json:"openings"`
}

// GET /api/a/academic-terms/search?year=2026&id=<masjid_id>&page_size=20&page=1
func (ctl *AcademicTermController) SearchByYear(c *fiber.Ctx) error {
	yearQ := strings.TrimSpace(c.Query("year"))
	masjidIDParam := strings.TrimSpace(c.Query("id")) // dimaknai sebagai masjid_id

	// multi-tenant via token
	masjidIDs, err := helperAuth.GetMasjidIDsFromToken(c)
	if err != nil {
		return err
	}

	// jika masjid_id diberikan, validasi & limit ke masjid tsb (authorization)
	if masjidIDParam != "" {
		wantID, err := uuid.Parse(masjidIDParam)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid 'id' (must be uuid masjid_id)")
		}
		authorized := false
		for _, id := range masjidIDs {
			if id == wantID {
				authorized = true
				break
			}
		}
		if !authorized {
			return fiber.NewError(fiber.StatusForbidden, "masjid_id tidak diizinkan untuk user ini")
		}
		masjidIDs = []uuid.UUID{wantID}
	}

	// paginasi sederhana (default)
	page := 1
	pageSize := 20
	if v := c.Query("page"); v != "" {
		if n, _ := strconv.Atoi(v); n > 0 {
			page = n
		}
	}
	if v := c.Query("page_size"); v != "" {
		if n, _ := strconv.Atoi(v); n > 0 && n <= 200 {
			pageSize = n
		}
	}

	dbq := ctl.DB.Model(&model.AcademicTermModel{}).
		Where("academic_terms_masjid_id IN (?) AND academic_terms_deleted_at IS NULL", masjidIDs)

	// filter
	if yearQ != "" {
		dbq = dbq.Where("academic_terms_academic_year ILIKE ?", "%"+yearQ+"%")
	}

	// count
	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Count failed: "+err.Error())
	}

	// ambil terms
	var list []model.AcademicTermModel
	if err := dbq.
		Order("academic_terms_start_date DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&list).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Query failed: "+err.Error())
	}

	// kumpulkan term_ids
	termIDs := make([]uuid.UUID, 0, len(list))
	for _, t := range list {
		termIDs = append(termIDs, t.AcademicTermsID)
	}

	// map term_id -> openings[]
	openingMap := make(map[uuid.UUID][]OpeningWithClass, len(list))
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
			ClassID            uuid.UUID
			ClassMasjidID      *uuid.UUID
			ClassName          string
			ClassSlug          string
			ClassDescription   *string
			ClassLevel         *string
			ClassImageURL      *string
			ClassIsActive      bool
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
				c.class_level, c.class_image_url, c.class_fee_monthly_idr, c.class_is_active
			`).
			Joins("JOIN classes c ON c.class_id = o.class_term_openings_class_id AND c.class_deleted_at IS NULL").
			Where("o.class_term_openings_term_id IN (?)", termIDs).
			Where("o.class_term_openings_masjid_id IN (?)", masjidIDs).
			Where("o.class_term_openings_deleted_at IS NULL").
			Order("o.class_term_openings_created_at DESC").
			Find(&rows).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Query openings failed: "+err.Error())
		}

		for _, r := range rows {
			item := OpeningWithClass{
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

	return helper.JsonList(c, out, fiber.Map{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"query":     yearQ,
		"masjid_id": masjidIDParam,
	})
}
