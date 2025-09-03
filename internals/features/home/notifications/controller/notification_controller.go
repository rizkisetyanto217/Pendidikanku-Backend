package controller

import (
	"log"
	"strconv"

	"masjidku_backend/internals/features/home/notifications/dto"
	"masjidku_backend/internals/features/home/notifications/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type NotificationController struct {
	DB *gorm.DB
}

func NewNotificationController(db *gorm.DB) *NotificationController {
	return &NotificationController{DB: db}
}

// 🟢 POST /api/a/notifications
func (ctrl *NotificationController) CreateNotification(c *fiber.Ctx) error {
	var req dto.NotificationRequest
	if err := c.BodyParser(&req); err != nil {
		log.Printf("[ERROR] Gagal parsing body: %v", err)
		return helper.JsonError(c, fiber.StatusBadRequest, "Permintaan tidak valid")
	}

	newNotif := req.ToModel()
	if err := ctrl.DB.Create(newNotif).Error; err != nil {
		log.Printf("[ERROR] Gagal menyimpan notifikasi: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan notifikasi")
	}

	return helper.JsonCreated(c, "Notifikasi berhasil dikirim", dto.ToNotificationResponse(newNotif))
}

// 🟢 POST /api/a/notifications/by-masjid  (body { "masjid_id": "..." }) + pagination
func (ctrl *NotificationController) GetNotificationsByMasjid(c *fiber.Ctx) error {
	type MasjidPayload struct {
		MasjidID uuid.UUID `json:"masjid_id"`
	}
	var payload MasjidPayload
	if err := c.BodyParser(&payload); err != nil || payload.MasjidID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
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

	// count
	var total int64
	if err := ctrl.DB.Model(&model.NotificationModel{}).
		Where("notification_masjid_id = ?", payload.MasjidID).
		Count(&total).Error; err != nil {
		log.Printf("[ERROR] Count notifs by masjid: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	// data
	var notifs []model.NotificationModel
	if err := ctrl.DB.
		Where("notification_masjid_id = ?", payload.MasjidID).
		Order("notification_created_at DESC").
		Limit(limit).Offset(offset).
		Find(&notifs).Error; err != nil {
		log.Printf("[ERROR] Gagal mengambil data notifikasi: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	pagination := fiber.Map{
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": int((total + int64(limit) - 1) / int64(limit)),
		"has_next":    int64(page*limit) < total,
		"has_prev":    page > 1,
		"masjid_id":   payload.MasjidID,
	}

	return helper.JsonList(c, dto.ToNotificationResponseList(notifs), pagination)
}

// 🟢 GET /api/a/notifications  (+ pagination)
func (ctrl *NotificationController) GetAllNotifications(c *fiber.Ctx) error {
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

	// count
	var total int64
	if err := ctrl.DB.Model(&model.NotificationModel{}).Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	// data
	var notifs []model.NotificationModel
	if err := ctrl.DB.
		Order("notification_created_at DESC").
		Limit(limit).Offset(offset).
		Find(&notifs).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	pagination := fiber.Map{
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": int((total + int64(limit) - 1) / int64(limit)),
		"has_next":    int64(page*limit) < total,
		"has_prev":    page > 1,
	}
	return helper.JsonList(c, dto.ToNotificationResponseList(notifs), pagination)
}

// 🟢 GET /api/u/notifications  (+ pagination)
// (Jika nanti ada kolom user/receiver, tinggal tambahkan filter WHERE sesuai user_id dari token)
func (ctrl *NotificationController) GetAllNotificationsForUser(c *fiber.Ctx) error {
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

	// count
	var total int64
	if err := ctrl.DB.Model(&model.NotificationModel{}).Count(&total).Error; err != nil {
		log.Printf("[ERROR] Count notifications for user: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung notifikasi")
	}

	// data
	var notifs []model.NotificationModel
	if err := ctrl.DB.
		Order("notification_created_at DESC").
		Limit(limit).Offset(offset).
		Find(&notifs).Error; err != nil {
		log.Printf("[ERROR] Gagal mengambil notifikasi untuk user: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil notifikasi")
	}

	pagination := fiber.Map{
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": int((total + int64(limit) - 1) / int64(limit)),
		"has_next":    int64(page*limit) < total,
		"has_prev":    page > 1,
	}
	return helper.JsonList(c, dto.ToNotificationResponseList(notifs), pagination)
}


// 🛑 DELETE /api/a/notifications/:id[?hard=true]
func (ctrl *NotificationController) DeleteNotification(c *fiber.Ctx) error {
	// param id
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Format ID tidak valid")
	}

	// ambil data (alive only)
	var notif model.NotificationModel
	if err := ctrl.DB.
		Where("notification_id = ?", id).
		First(&notif).Error; err != nil {
		// GORM default exclude soft-deleted — kalau mau menampilkan pesan sama
		// untuk yang sudah terhapus, cukup tindak-nyatakan not found
		return helper.JsonError(c, fiber.StatusNotFound, "Notifikasi tidak ditemukan")
	}

	// otorisasi tenant (jika notifikasi terikat masjid)
	// global notification (notification_masjid_id == NULL) bebas dari scope ini;
	// kalau ingin dibatasi, tambahkan pengecekan role di sini.
	if notif.NotificationMasjidID != nil {
		masjidID, terr := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
		if terr != nil || *notif.NotificationMasjidID != masjidID {
			return helper.JsonError(c, fiber.StatusForbidden, "Tidak punya akses untuk menghapus notifikasi ini")
		}
	}

	// soft delete (default) atau hard delete (opsional)
	hard := c.Query("hard") == "true"
	if hard {
		if err := ctrl.DB.Unscoped().Delete(&notif).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus notifikasi (hard)")
		}
		return helper.JsonOK(c, "Notifikasi berhasil dihapus permanen", nil)
	}

	if err := ctrl.DB.Delete(&notif).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus notifikasi")
	}
	return helper.JsonOK(c, "Notifikasi berhasil dihapus", nil)
}
