package controller

import (
	"errors"
	"masjidku_backend/internals/features/lembaga/academics/academic_terms/dto"
	"masjidku_backend/internals/features/lembaga/academics/academic_terms/model"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	helper "masjidku_backend/internals/helpers"
)

// -----------------------------
// GetByID (scoped ke masjid di token)
// -----------------------------
// -----------------------------
// GetByID (scoped ke masjid di token)
// -----------------------------
func (ctl *AcademicTermController) GetByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid id")
	}

	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
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


// -----------------------------
// List (multi-tenant via token) + Filter + Pagination + Sorting
// -----------------------------
func (ctl *AcademicTermController) List(c *fiber.Ctx) error {
	var q dto.AcademicTermFilterDTO
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid query: "+err.Error())
	}
	if err := ctl.Validator.Struct(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Validation failed: "+err.Error())
	}

	// Dapatkan SEMUA masjid yang boleh diakses dari token
	masjidIDs, err := helper.GetMasjidIDsFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

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

	if q.Year != nil && strings.TrimSpace(*q.Year) != "" {
		dbq = dbq.Where("academic_terms_academic_year = ?", strings.TrimSpace(*q.Year))
	}
	if q.Name != nil && strings.TrimSpace(*q.Name) != "" {
		dbq = dbq.Where("academic_terms_name = ?", strings.TrimSpace(*q.Name))
	}
	if q.Active != nil {
		dbq = dbq.Where("academic_terms_is_active = ?", *q.Active)
	}

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

	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Count failed: "+err.Error())
	}

	var list []model.AcademicTermModel
	if err := dbq.
		Order(sortBy + " " + sortDir).
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&list).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Query failed: "+err.Error())
	}

	// Gunakan helper.JsonList untuk response list + pagination
	pagination := fiber.Map{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}
	return helper.JsonList(c, dto.FromModels(list), pagination)
}








type OpeningWithClass struct {
	// --- opening (mirror kolom, seperti response DTO openings) ---
	ClassTermOpeningsID                    uuid.UUID  `json:"class_term_openings_id"                     gorm:"column:class_term_openings_id"`
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

	// --- class (subset penting; tambah kalau perlu) ---
	Class struct {
		ClassID            uuid.UUID  `json:"class_id"               gorm:"column:class_id"`
		ClassMasjidID      *uuid.UUID `json:"class_masjid_id"        gorm:"column:class_masjid_id"`
		ClassName          string     `json:"class_name"             gorm:"column:class_name"`
		ClassSlug          string     `json:"class_slug"             gorm:"column:class_slug"`
		ClassDescription   *string    `json:"class_description"      gorm:"column:class_description"`
		ClassLevel         *string    `json:"class_level"            gorm:"column:class_level"`
		ClassImageURL      *string    `json:"class_image_url"        gorm:"column:class_image_url"`
		ClassFeeMonthlyIDR *int       `json:"class_fee_monthly_idr"  gorm:"column:class_fee_monthly_idr"`
		ClassIsActive      bool       `json:"class_is_active"        gorm:"column:class_is_active"`
	} `json:"class"`
}

type AcademicTermWithOpeningsResponse struct {
	dto.AcademicTermResponseDTO
	Openings []OpeningWithClass `json:"openings"`
}

// GET /academic-terms/search?year=2026&id=<uuid>&page_size=20&page=1

// GET /academic-terms/search?year=2026&masjid_id=<uuid>&page_size=20&page=1
func (ctl *AcademicTermController) SearchByYear(c *fiber.Ctx) error {
	yearQ := strings.TrimSpace(c.Query("year"))
	masjidIDParam := strings.TrimSpace(c.Query("id")) // <-- sekarang dimaknai sebagai masjid_id

	// multi-tenant via token
	masjidIDs, err := helper.GetMasjidIDsFromToken(c)
	if err != nil {
		return err
	}

	// jika masjid_id diberikan, validasi & limit ke masjid tersebut (dengan otorisasi)
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
		// batasi scope hanya ke masjid itu
		masjidIDs = []uuid.UUID{wantID}
	}

	// paginasi sederhana (fallback default)
	page := 1
	pageSize := 20
	if v := c.Query("page"); v != "" {
		if n, _ := strconv.Atoi(v); n > 0 { page = n }
	}
	if v := c.Query("page_size"); v != "" {
		if n, _ := strconv.Atoi(v); n > 0 && n <= 200 { pageSize = n }
	}

	dbq := ctl.DB.Model(&model.AcademicTermModel{}).
		Where("academic_terms_masjid_id IN (?) AND academic_terms_deleted_at IS NULL", masjidIDs)

	// --- APPLY FILTERS ---
	if yearQ != "" {
		dbq = dbq.Where("academic_terms_academic_year ILIKE ?", "%"+yearQ+"%")
	}

	// count
	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Count failed: "+err.Error())
	}

	// fetch terms
	var list []model.AcademicTermModel
	if err := dbq.
		Order("academic_terms_start_date DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&list).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Query failed: "+err.Error())
	}

	// kumpulkan term_ids untuk fetch openings
	termIDs := make([]uuid.UUID, 0, len(list))
	for _, t := range list {
		termIDs = append(termIDs, t.AcademicTermsID)
	}

	// map term_id -> openings[]
	openingMap := make(map[uuid.UUID][]OpeningWithClass, len(list))
	if len(termIDs) > 0 {
		type row struct {
			// opening cols
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
			// class cols
			ClassID            uuid.UUID
			ClassMasjidID      *uuid.UUID
			ClassName          string
			ClassSlug          string
			ClassDescription   *string
			ClassLevel         *string
			ClassImageURL      *string
			ClassFeeMonthlyIDR *int
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
			Where("o.class_term_openings_masjid_id IN (?)", masjidIDs). // sudah discope
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
			item.Class.ClassFeeMonthlyIDR = r.ClassFeeMonthlyIDR
			item.Class.ClassIsActive = r.ClassIsActive

			openingMap[r.ClassTermOpeningsTermID] = append(openingMap[r.ClassTermOpeningsTermID], item)
		}
	}

	// susun response
	termsDTO := dto.FromModels(list)
	out := make([]AcademicTermWithOpeningsResponse, 0, len(termsDTO))
	for i, t := range list {
		out = append(out, AcademicTermWithOpeningsResponse{
			AcademicTermResponseDTO: termsDTO[i],
			Openings:                openingMap[t.AcademicTermsID],
		})
	}

	// gunakan helper.JsonList untuk respons sukses standar
	return helper.JsonList(c, out, fiber.Map{
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
		"query":      yearQ,
		"masjid_id":  masjidIDParam, // hanya diisi kalau user kirim param
	})
}
