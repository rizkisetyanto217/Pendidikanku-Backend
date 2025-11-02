// file: internals/features/lembaga/classes/subjects/main/controller/class_subject_list_controller.go
package controller

import (
	"errors"
	"strings"

	csDTO "schoolku_backend/internals/features/school/academics/subjects/dto"
	csModel "schoolku_backend/internals/features/school/academics/subjects/model"
	helper "schoolku_backend/internals/helpers"
	helperAuth "schoolku_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

/*
=========================================================

	LIST (Sederhana)
	GET /admin/class-subjects

	Query (mengikuti DTO ListClassSubjectQuery):
	  - q                 : cari pada desc (ILIKE)
	  - is_active         : bool
	  - parent_id         : UUID
	  - subject_id        : UUID
	  - with_deleted      : bool (default false)
	  - order_by          : order_index|created_at|updated_at (default: created_at)
	  - sort              : asc|desc (default: asc)
	  - limit (1..200), offset

=========================================================
*/
func (h *ClassSubjectController) List(c *fiber.Ctx) error {
	// ===== School context (PUBLIC): no role check =====
	mc, err := helperAuth.ResolveSchoolContext(c)
	if err != nil {
		return err
	}
	var schoolID uuid.UUID
	switch {
	case mc.ID != uuid.Nil:
		schoolID = mc.ID
	case strings.TrimSpace(mc.Slug) != "":
		id, er := helperAuth.GetSchoolIDBySlug(c, strings.TrimSpace(mc.Slug))
		if er != nil {
			if errors.Is(er, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusNotFound, "School (slug) tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal resolve school dari slug")
		}
		schoolID = id
	default:
		return helperAuth.ErrSchoolContextMissing
	}

	// ===== Parse & guard pagination
	var q csDTO.ListClassSubjectQuery
	if err := c.QueryParser(&q); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Query tidak valid")
	}
	limit, offset := 20, 0
	if q.Limit != nil && *q.Limit > 0 && *q.Limit <= 200 {
		limit = *q.Limit
	}
	if q.Offset != nil && *q.Offset >= 0 {
		offset = *q.Offset
	}

	// ===== Base query (single-tenant via context) =====
	tx := h.DB.Model(&csModel.ClassSubjectModel{}).
		Where("class_subject_school_id = ?", schoolID)

	// ===== Soft delete (default exclude)
	if q.WithDeleted == nil || !*q.WithDeleted {
		tx = tx.Where("class_subject_deleted_at IS NULL")
	}

	// ===== Filter aktif
	if q.IsActive != nil {
		tx = tx.Where("class_subject_is_active = ?", *q.IsActive)
	}

	// ===== Filter by parent_id / subject_id (langsung dari DTO)
	if q.ParentID != nil {
		tx = tx.Where("class_subject_parent_id = ?", *q.ParentID)
	}
	if q.SubjectID != nil {
		tx = tx.Where("class_subject_subject_id = ?", *q.SubjectID)
	}

	// ===== Search di desc
	if q.Q != nil && strings.TrimSpace(*q.Q) != "" {
		kw := "%" + strings.ToLower(strings.TrimSpace(*q.Q)) + "%"
		tx = tx.Where("LOWER(COALESCE(class_subject_desc,'')) LIKE ?", kw)
	}

	// ===== Sorting whitelist
	orderBy := "class_subject_created_at"
	if q.OrderBy != nil {
		switch strings.ToLower(strings.TrimSpace(*q.OrderBy)) {
		case "order_index":
			orderBy = "class_subject_order_index"
		case "created_at":
			orderBy = "class_subject_created_at"
		case "updated_at":
			orderBy = "class_subject_updated_at"
		}
	}
	sort := "ASC"
	if q.Sort != nil && strings.ToLower(strings.TrimSpace(*q.Sort)) == "desc" {
		sort = "DESC"
	}

	// ===== Total (sebelum limit/offset)
	var total int64
	if err := tx.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghitung total data")
	}

	// ===== Data
	var rows []csModel.ClassSubjectModel
	if err := tx.
		Order(orderBy + " " + sort).
		Limit(limit).
		Offset(offset).
		Find(&rows).Error; err != nil { // ⬅️ tanpa Select(), ambil semua kolom termasuk snapshots
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	return helper.JsonList(
		c,
		csDTO.FromClassSubjectModels(rows), // DTO akan mengisi snapshot dari model
		csDTO.Pagination{Limit: limit, Offset: offset, Total: int(total)},
	)

}
