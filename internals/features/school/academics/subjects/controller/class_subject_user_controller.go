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
  - limit/per_page, page/offset

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
				return helper.JsonError(c, fiber.StatusNotFound, "School (slug) tidak ditemukan")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal resolve school dari slug")
		}
		schoolID = id
	default:
		return helperAuth.ErrSchoolContextMissing
	}

	// ===== Parse query DTO (toleran) =====
	var q csDTO.ListClassSubjectQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}

	// ===== Paging (jsonresponse helper; dukung page/per_page & limit/offset) =====
	// default per_page = 20, max = 200 sesuai kebutuhan endpoint ini
	p := helper.ResolvePaging(c, 20, 200)
	limit, offset := p.Limit, p.Offset

	// ===== Base query (tenant-safe) =====
	tx := h.DB.Model(&csModel.ClassSubjectModel{}).
		Where("class_subject_school_id = ?", schoolID)

	// ===== Soft delete (default exclude)
	withDeleted := q.WithDeleted != nil && *q.WithDeleted
	if !withDeleted {
		tx = tx.Where("class_subject_deleted_at IS NULL")
	}

	// ===== Filters =====
	if q.IsActive != nil {
		tx = tx.Where("class_subject_is_active = ?", *q.IsActive)
	}
	if q.ParentID != nil {
		tx = tx.Where("class_subject_parent_id = ?", *q.ParentID)
	}
	if q.SubjectID != nil {
		tx = tx.Where("class_subject_subject_id = ?", *q.SubjectID)
	}
	if q.Q != nil && strings.TrimSpace(*q.Q) != "" {
		kw := "%" + strings.ToLower(strings.TrimSpace(*q.Q)) + "%"
		tx = tx.Where("LOWER(COALESCE(class_subject_desc,'')) LIKE ?", kw)
	}

	// ===== Sorting whitelist =====
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

	// ===== Count sebelum limit/offset =====
	var total int64
	if err := tx.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung total data")
	}

	// ===== Data =====
	var rows []csModel.ClassSubjectModel
	if err := tx.
		Order(orderBy + " " + sort).
		Limit(limit).
		Offset(offset).
		Find(&rows).Error; err != nil { // ambil semua kolom termasuk snapshots
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// ===== Pagination meta (jsonresponse) =====
	pg := helper.BuildPaginationFromOffset(total, offset, limit)

	// ===== Response standar (jsonresponse) =====
	return helper.JsonList(c, "ok", csDTO.FromClassSubjectModels(rows), pg)
}
