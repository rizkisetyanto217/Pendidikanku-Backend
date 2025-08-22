package controller

import (
	"log"
	"strconv"
	"time"

	"masjidku_backend/internals/features/masjids/events/dto"
	"masjidku_backend/internals/features/masjids/events/model"
	helper "masjidku_backend/internals/helpers"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EventSessionController struct {
	DB *gorm.DB
}

func NewEventSessionController(db *gorm.DB) *EventSessionController {
	return &EventSessionController{DB: db}
}

// 🟢 POST /api/a/event-sessions
func (ctrl *EventSessionController) CreateEventSession(c *fiber.Ctx) error {
	uidRaw := c.Locals("user_id")
	uidStr, ok := uidRaw.(string)
	if !ok || uidStr == "" {
		log.Println("[ERROR] user_id tidak ditemukan di token")
		return helper.JsonError(c, fiber.StatusUnauthorized, "User tidak terautentikasi")
	}
	userID, err := uuid.Parse(uidStr)
	if err != nil {
		log.Printf("[ERROR] parse user_id: %v", err)
		return helper.JsonError(c, fiber.StatusUnauthorized, "User ID tidak valid")
	}

	var req dto.EventSessionRequest
	if err := c.BodyParser(&req); err != nil {
		log.Printf("[ERROR] body parser: %v", err)
		return helper.JsonError(c, fiber.StatusBadRequest, "Permintaan tidak valid")
	}

	session := req.ToModel()
	session.EventSessionCreatedBy = &userID

	if err := ctrl.DB.Create(session).Error; err != nil {
		log.Printf("[ERROR] create session: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan event session")
	}

	return helper.JsonCreated(c, "Event session berhasil dibuat", dto.ToEventSessionResponse(session))
}

// 🟢 GET /api/u/event-sessions/by-event/:event_id?page=&limit=
func (ctrl *EventSessionController) GetEventSessionsByEvent(c *fiber.Ctx) error {
	eventID := c.Params("event_id")
	if eventID == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Event ID tidak boleh kosong")
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
	if err := ctrl.DB.Model(&model.EventSessionModel{}).
		Where("event_session_event_id = ?", eventID).
		Count(&total).Error; err != nil {
		log.Printf("[ERROR] count sessions: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung event sessions")
	}

	// data
	var sessions []model.EventSessionModel
	if err := ctrl.DB.
		Where("event_session_event_id = ?", eventID).
		Order("event_session_start_time ASC").
		Limit(limit).Offset(offset).
		Find(&sessions).Error; err != nil {
		log.Printf("[ERROR] get sessions: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil event sessions")
	}

	// map + status
	now := time.Now()
	res := make([]dto.EventSessionResponse, 0, len(sessions))
	for i := range sessions {
		status := "upcoming"
		if now.After(sessions[i].EventSessionStartTime) && now.Before(sessions[i].EventSessionEndTime) {
			status = "ongoing"
		} else if now.After(sessions[i].EventSessionEndTime) {
			status = "completed"
		}
		item := dto.ToEventSessionResponse(&sessions[i])
		item.EventSessionStatus = status
		res = append(res, *item)
	}

	pagination := fiber.Map{
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": int((total + int64(limit) - 1) / int64(limit)),
		"has_next":    int64(page*limit) < total,
		"has_prev":    page > 1,
		"event_id":    eventID,
	}
	return helper.JsonList(c, res, pagination)
}

// 🟢 GET /api/u/event-sessions/all?page=&limit=
func (ctrl *EventSessionController) GetAllEventSessions(c *fiber.Ctx) error {
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
	if err := ctrl.DB.Model(&model.EventSessionModel{}).Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung event sessions")
	}

	// data
	var sessions []model.EventSessionModel
	if err := ctrl.DB.
		Order("event_session_start_time DESC").
		Limit(limit).Offset(offset).
		Find(&sessions).Error; err != nil {
		log.Printf("[ERROR] get all sessions: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data event sessions")
	}

	// map + status
	now := time.Now()
	res := make([]dto.EventSessionResponse, 0, len(sessions))
	for i := range sessions {
		status := "upcoming"
		if now.After(sessions[i].EventSessionStartTime) && now.Before(sessions[i].EventSessionEndTime) {
			status = "ongoing"
		} else if now.After(sessions[i].EventSessionEndTime) {
			status = "completed"
		}
		item := dto.ToEventSessionResponse(&sessions[i])
		item.EventSessionStatus = status
		res = append(res, *item)
	}

	pagination := fiber.Map{
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": int((total + int64(limit) - 1) / int64(limit)),
		"has_next":    int64(page*limit) < total,
		"has_prev":    page > 1,
	}
	return helper.JsonList(c, res, pagination)
}

// 🟢 GET /api/u/event-sessions/upcoming/:masjid_id?page=&limit=
// (kalau :masjid_id tidak dikirim, ambil semua publik upcoming)
func (ctrl *EventSessionController) GetUpcomingEventSessions(c *fiber.Ctx) error {
	var sessions []model.EventSessionModel

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

	// base query: upcoming & public
	query := ctrl.DB.
		Where("event_session_start_time > ? AND event_session_is_public = ?", time.Now(), true)

	// filter by masjid_id jika ada di path
	masjidIDStr := c.Params("masjid_id")
	if masjidIDStr != "" {
		masjidID, err := uuid.Parse(masjidIDStr)
		if err != nil {
			log.Printf("[ERROR] invalid masjid_id: %v", err)
			return helper.JsonError(c, fiber.StatusBadRequest, "Format ID masjid tidak valid")
		}
		query = query.Where("event_session_masjid_id = ?", masjidID)
	}

	// count
	var total int64
	if err := query.Model(&model.EventSessionModel{}).Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung sesi event")
	}

	// data
	if err := query.
		Order("event_session_start_time ASC").
		Limit(limit).Offset(offset).
		Find(&sessions).Error; err != nil {
		log.Printf("[ERROR] get upcoming sessions: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil sesi event upcoming")
	}

	pagination := fiber.Map{
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": int((total + int64(limit) - 1) / int64(limit)),
		"has_next":    int64(page*limit) < total,
		"has_prev":    page > 1,
		"masjid_id":   masjidIDStr,
	}

	return helper.JsonList(c, dto.ToEventSessionResponseList(sessions), pagination)
}


// 🟢 PUT /api/a/event-sessions/:id
func (ctrl *EventSessionController) UpdateEventSession(c *fiber.Ctx) error {
	// --- Param ---
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Format ID tidak valid")
	}

	// --- Ambil session ---
	var session model.EventSessionModel
	if err := ctrl.DB.First(&session, "event_session_id = ?", id).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "Event session tidak ditemukan")
	}

	// --- Authorization by masjid ---
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil || session.EventSessionMasjidID != masjidID {
		return helper.JsonError(c, fiber.StatusForbidden, "Tidak punya akses untuk update event session ini")
	}

	// --- Bind request ---
	var req dto.EventSessionUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Permintaan tidak valid")
	}

	// --- Apply partial fields ---
	if req.EventSessionSlug != nil {
		session.EventSessionSlug = *req.EventSessionSlug
	}
	if req.EventSessionTitle != nil {
		session.EventSessionTitle = *req.EventSessionTitle
	}
	if req.EventSessionDescription != nil {
		session.EventSessionDescription = *req.EventSessionDescription
	}
	if req.EventSessionStartTime != nil {
		session.EventSessionStartTime = *req.EventSessionStartTime
	}
	if req.EventSessionEndTime != nil {
		session.EventSessionEndTime = *req.EventSessionEndTime
	}
	if req.EventSessionLocation != nil {
		session.EventSessionLocation = *req.EventSessionLocation
	}
	if req.EventSessionImageURL != nil {
		session.EventSessionImageURL = *req.EventSessionImageURL
	}
	if req.EventSessionCapacity != nil {
		session.EventSessionCapacity = *req.EventSessionCapacity
	}
	if req.EventSessionMasjidID != nil {
		session.EventSessionMasjidID = *req.EventSessionMasjidID
	}
	if req.EventSessionIsPublic != nil {
		session.EventSessionIsPublic = *req.EventSessionIsPublic
	}
	if req.EventSessionIsRegistrationNeeded != nil {
		session.EventSessionIsRegistrationNeeded = *req.EventSessionIsRegistrationNeeded
	}
	// --- Simpan ---
	if err := ctrl.DB.Save(&session).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengupdate event session")
	}

	return helper.JsonOK(c, "Event session berhasil diperbarui", dto.ToEventSessionResponse(&session))
}

// 🟢 DELETE /api/a/event-sessions/:id
func (ctrl *EventSessionController) DeleteEventSession(c *fiber.Ctx) error {
	// --- Param ---
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
	return helper.JsonError(c, fiber.StatusBadRequest, "Format ID tidak valid")
	}

	// --- Ambil session ---
	var session model.EventSessionModel
	if err := ctrl.DB.First(&session, "event_session_id = ?", id).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "Event session tidak ditemukan")
	}

	// --- Authorization by masjid ---
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil || session.EventSessionMasjidID != masjidID {
		return helper.JsonError(c, fiber.StatusForbidden, "Tidak punya akses untuk menghapus event session ini")
	}

	// --- Hapus ---
	if err := ctrl.DB.Delete(&session).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus event session")
	}

	return helper.JsonOK(c, "Event session berhasil dihapus", nil)
}
