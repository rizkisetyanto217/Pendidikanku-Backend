package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	clsDTO "masjidku_backend/internals/features/school/academics/rooms/dto"
	clsModel "masjidku_backend/internals/features/school/academics/rooms/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
)

// GET /admin/class-room-virtual-links
// Query:
//
//	masjid_id, room_id, active, q
//	sort_by: created_at|updated_at|label (default: created_at)
//	order: asc|desc
//	limit, page
func (h *ClassRoomVirtualLinkController) List(c *fiber.Ctx) error {
	// Pagination & sorting (shared helper)
	req, _ := http.NewRequest("GET", "http://local"+c.OriginalURL(), nil)
	p := helper.ParseWith(req, "created_at", "desc", helper.AdminOpts)

	orderClause, err := p.SafeOrderClause(map[string]string{
		"created_at": "class_room_virtual_link_created_at",
		"updated_at": "class_room_virtual_link_updated_at",
		"label":      "LOWER(class_room_virtual_link_label)",
	}, "created_at")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "sort_by tidak dikenal")
	}
	orderExpr := strings.TrimPrefix(strings.TrimSpace(orderClause), "ORDER BY ")

	// ==== Masjid context (PUBLIC) ====
	mc, _ := helperAuth.ResolveMasjidContext(c) // jangan gagal dulu; kita masih cek ?masjid_id
	var masjidID uuid.UUID

	// 1) Prioritas: ?masjid_id=
	if v := strings.TrimSpace(c.Query("masjid_id")); v != "" {
		uid, e := uuid.Parse(v)
		if e != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "masjid_id tidak valid")
		}
		masjidID = uid
	}

	// 2) Kalau masih kosong, coba dari context (path/header/cookie/query/host)
	if masjidID == uuid.Nil {
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
		}
	}

	// 3) Wajib punya konteks masjid untuk list
	if masjidID == uuid.Nil {
		return helperAuth.ErrMasjidContextMissing
	}

	// ==== Base query (tenant-safe) ====
	q := h.DB.Model(&clsModel.ClassRoomVirtualLinkModel{}).
		Where("class_room_virtual_link_masjid_id = ?", masjidID)

	// Filters
	if v := strings.TrimSpace(c.Query("room_id")); v != "" {
		if uid, e := uuid.Parse(v); e == nil {
			q = q.Where("class_room_virtual_link_room_id = ?", uid)
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "room_id tidak valid")
		}
	}
	if v := c.Query("active"); v != "" {
		if b, e := strconv.ParseBool(v); e == nil {
			q = q.Where("class_room_virtual_link_is_active = ?", b)
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "active harus boolean")
		}
	}
	if kw := strings.TrimSpace(c.Query("q")); kw != "" {
		like := "%" + kw + "%"
		q = q.Where(
			"class_room_virtual_link_label ILIKE ? OR class_room_virtual_link_join_url ILIKE ?",
			like, like,
		)
	}

	// Count
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	// Data
	var rows []clsModel.ClassRoomVirtualLinkModel
	if err := q.Order(orderExpr).
		Limit(p.Limit()).
		Offset(p.Offset()).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	items := make([]clsDTO.ClassRoomVirtualLinkResponse, 0, len(rows))
	for i := range rows {
		items = append(items, clsDTO.FromModelClassRoomVirtualLink(&rows[i]))
	}
	meta := helper.BuildMeta(total, p)
	return helper.JsonList(c, items, meta)
}
