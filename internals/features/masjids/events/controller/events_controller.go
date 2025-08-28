package controller

import (
	"log"
	"strconv"
	"strings"

	"masjidku_backend/internals/features/masjids/events/dto"
	"masjidku_backend/internals/features/masjids/events/model"
	helper "masjidku_backend/internals/helpers"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EventController struct {
	DB *gorm.DB
}

func NewEventController(db *gorm.DB) *EventController {
	return &EventController{DB: db}
}

// 🟢 POST /api/a/events
func (ctrl *EventController) CreateEvent(c *fiber.Ctx) error {
	var req dto.EventRequest
	if err := c.BodyParser(&req); err != nil {
		log.Printf("[ERROR] Body parser gagal: %v", err)
		return helper.JsonError(c, fiber.StatusBadRequest, "Permintaan tidak valid")
	}

	newEvent := req.ToModel()
	if err := ctrl.DB.Create(newEvent).Error; err != nil {
		log.Printf("[ERROR] Gagal menyimpan event: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan event")
	}

	return helper.JsonCreated(c, "Event berhasil ditambahkan", dto.ToEventResponse(newEvent))
}

// 🟢 GET /api/u/events/id/:id
func (ctrl *EventController) GetEventByID(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Event ID tidak boleh kosong")
	}

	var ev model.EventModel
	if err := ctrl.DB.Where("event_id = ?", id).First(&ev).Error; err != nil {
		log.Printf("[ERROR] Event dengan ID '%s' tidak ditemukan: %v", id, err)
		return helper.JsonError(c, fiber.StatusNotFound, "Event tidak ditemukan")
	}

	return helper.JsonOK(c, "Event berhasil ditemukan", dto.ToEventResponse(&ev))
}

// 🟡 PATCH /api/a/events/:id
func (ctrl *EventController) UpdateEvent(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Event ID tidak boleh kosong")
	}

	// Ambil record lama
	var ev model.EventModel
	if err := ctrl.DB.Where("event_id = ?", id).First(&ev).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "Event tidak ditemukan")
	}

	// Parse body
	var req dto.EventUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Permintaan tidak valid")
	}

	updates := map[string]interface{}{}

	// Jika judul diupdate → slug ikut diupdate
	if req.EventTitle != nil {
		updates["event_title"] = *req.EventTitle
		updates["event_slug"] = dto.GenerateSlug(*req.EventTitle)
	}
	if req.EventDescription != nil {
		updates["event_description"] = *req.EventDescription
	}
	if req.EventLocation != nil {
		updates["event_location"] = *req.EventLocation
	}
	if req.EventMasjidID != nil {
		// (Opsional) precheck FK biar error jadi 404/400, bukan 500
		var cnt int64
		if err := ctrl.DB.Table("masjids").
			Where("masjid_id = ?", *req.EventMasjidID).
			Count(&cnt).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memeriksa masjid")
		}
		if cnt == 0 {
			return helper.JsonError(c, fiber.StatusNotFound, "Masjid tidak ditemukan")
		}
		updates["event_masjid_id"] = *req.EventMasjidID
	}

	if len(updates) == 0 {
		return helper.JsonError(c, fiber.StatusBadRequest, "Tidak ada field yang diupdate")
	}

	// Lakukan update
	if err := ctrl.DB.Model(&ev).Updates(updates).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui event")
	}

	// Reload untuk response terbaru
	if err := ctrl.DB.Where("event_id = ?", id).First(&ev).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memuat data event terbaru")
	}

	return helper.JsonUpdated(c, "Event berhasil diperbarui", dto.ToEventResponse(&ev))
}

// 🟢 POST /api/a/events/by-masjid   (body: { "masjid_id": "..." }) + pagination
func (ctrl *EventController) GetEventsByMasjid(c *fiber.Ctx) error {
	type Request struct {
		MasjidID string `json:"masjid_id"`
	}
	var body Request
	if err := c.BodyParser(&body); err != nil || body.MasjidID == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Masjid ID tidak valid")
	}

	// pagination
	page, _ := strconv.Atoi(c.Query("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	// hitung total
	var total int64
	if err := ctrl.DB.Model(&model.EventModel{}).
		Where("event_masjid_id = ?", body.MasjidID).
		Count(&total).Error; err != nil {
		log.Printf("[ERROR] Count events by masjid: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung event")
	}

	// ambil data
	var events []model.EventModel
	if err := ctrl.DB.
		Where("event_masjid_id = ?", body.MasjidID).
		Order("event_created_at DESC").
		Limit(limit).Offset(offset).
		Find(&events).Error; err != nil {
		log.Printf("[ERROR] Gagal mengambil data event: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil event")
	}

	pagination := fiber.Map{
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": int((total + int64(limit) - 1) / int64(limit)),
		"has_next":    int64(page*limit) < total,
		"has_prev":    page > 1,
		"masjid_id":   body.MasjidID,
	}

	return helper.JsonList(c, dto.ToEventResponseList(events), pagination)
}

// 🟢 GET /api/a/events/all  (atau /api/u/events/all) + pagination
func (ctrl *EventController) GetAllEvents(c *fiber.Ctx) error {
	// pagination
	page, _ := strconv.Atoi(c.Query("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	// hitung total
	var total int64
	if err := ctrl.DB.Model(&model.EventModel{}).Count(&total).Error; err != nil {
		log.Printf("[ERROR] Count all events: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung event")
	}

	// data
	var events []model.EventModel
	if err := ctrl.DB.
		Order("event_created_at DESC").
		Limit(limit).Offset(offset).
		Find(&events).Error; err != nil {
		log.Printf("[ERROR] Gagal mengambil semua event: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data event")
	}

	pagination := fiber.Map{
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": int((total + int64(limit) - 1) / int64(limit)),
		"has_next":    int64(page*limit) < total,
		"has_prev":    page > 1,
	}

	return helper.JsonList(c, dto.ToEventResponseList(events), pagination)
}

// 🟢 GET /api/u/events/:slug
func (ctrl *EventController) GetEventBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Slug tidak boleh kosong")
	}

	var event model.EventModel
	if err := ctrl.DB.Where("event_slug = ?", slug).First(&event).Error; err != nil {
		log.Printf("[ERROR] Event dengan slug '%s' tidak ditemukan: %v", slug, err)
		return helper.JsonError(c, fiber.StatusNotFound, "Event tidak ditemukan")
	}

	return helper.JsonOK(c, "Event berhasil ditemukan", dto.ToEventResponse(&event))
}


// DELETE /api/a/events/:id[?hard=true]
func (ctrl *EventController) DeleteEvent(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if strings.TrimSpace(idStr) == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Event ID tidak boleh kosong")
	}
	evID, err := uuid.Parse(idStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Event ID tidak valid")
	}

	// Scope tenant
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err // sudah dalam bentuk fiber.Error dari helper
	}

	// Cek keberadaan event (hanya baris 'hidup' kalau pakai soft delete)
	var ev model.EventModel
	if err := ctrl.DB.
		Where("event_id = ? AND event_masjid_id = ?", evID, masjidID).
		First(&ev).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "Event tidak ditemukan atau bukan milik masjid ini")
	}

	hard := strings.EqualFold(c.Query("hard"), "true")

	// Jalankan dalam transaksi
	tx := ctrl.DB.Begin()
	if tx.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memulai transaksi")
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if hard {
		// HARD DELETE: akan memicu ON DELETE CASCADE ke event_sessions & user_event_registrations
		if err := tx.Unscoped().Delete(&ev).Error; err != nil {
			tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus event (hard)")
		}
	} else {
		// SOFT DELETE: tandai event_deleted_at; optional → soft delete anak-anaknya juga
		if err := tx.Delete(&ev).Error; err != nil {
			tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus event")
		}

		// (Opsional tapi direkomendasikan) Soft delete anak-anak agar konsisten.
		// Perhatikan: model anak harus juga pakai gorm.DeletedAt kolom *_deleted_at.
		if err := tx.
			Table("event_sessions").
			Where("event_session_event_id = ? AND event_session_masjid_id = ? AND event_session_deleted_at IS NULL", evID, masjidID).
			Update("event_session_deleted_at", gorm.Expr("now()")).Error; err != nil {
			tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menandai sesi event sebagai terhapus")
		}

		if err := tx.
			Table("user_event_registrations").
			Where(`user_event_registration_event_session_id IN (
                SELECT event_session_id FROM event_sessions
                WHERE event_session_event_id = ? AND event_session_masjid_id = ? AND event_session_deleted_at IS NOT NULL
            ) AND user_event_registration_deleted_at IS NULL`, evID, masjidID).
			Update("user_event_registration_deleted_at", gorm.Expr("now()")).Error; err != nil {
			tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menandai registrasi sebagai terhapus")
		}
	}

	if err := tx.Commit().Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan perubahan")
	}

	msg := "Event berhasil dihapus"
	if hard {
		msg = "Event berhasil dihapus permanen (cascade)"
	}
	return helper.JsonOK(c, msg, nil)
}
