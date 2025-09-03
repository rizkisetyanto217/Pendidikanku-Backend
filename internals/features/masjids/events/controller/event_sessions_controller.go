package controller

import (
	"log"
	"strconv"
	"time"

	"masjidku_backend/internals/features/masjids/events/dto"
	"masjidku_backend/internals/features/masjids/events/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

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

// view object: response + computed status (DTO tidak menyimpan status)
type eventSessionView struct {
	dto.EventSessionResponse
	EventSessionStatus string `json:"event_session_status"`
}

func computeStatus(now time.Time, start, end time.Time) string {
	switch {
	case now.Before(start):
		return "upcoming"
	case (now.Equal(start) || now.After(start)) && now.Before(end):
		return "ongoing"
	default:
		// now >= end
		return "completed"
	}
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

	// Validasi ringan sesuai constraint DB
	if !req.EventSessionEndTime.After(req.EventSessionStartTime) {
		return helper.JsonError(c, fiber.StatusBadRequest, "End time harus lebih besar dari start time")
	}
	if req.EventSessionCapacity != nil && *req.EventSessionCapacity < 0 {
		return helper.JsonError(c, fiber.StatusBadRequest, "Capacity tidak boleh negatif")
	}

	session := req.ToModel()
	session.EventSessionCreatedBy = &userID

	if err := ctrl.DB.Create(session).Error; err != nil {
		log.Printf("[ERROR] create session: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan event session")
	}

	// bungkus dengan status terhitung
	now := time.Now()
	view := eventSessionView{
		EventSessionResponse: *dto.ToEventSessionResponse(session),
		EventSessionStatus:   computeStatus(now, session.EventSessionStartTime, session.EventSessionEndTime),
	}

	return helper.JsonCreated(c, "Event session berhasil dibuat", view)
}

// 🟢 GET /api/u/event-sessions/by-event/:event_id?page=&limit=
func (ctrl *EventSessionController) GetEventSessionsByEvent(c *fiber.Ctx) error {
	eventIDStr := c.Params("event_id")
	if eventIDStr == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Event ID tidak boleh kosong")
	}
	// validasi UUID (menghindari cast implicit error di DB)
	if _, err := uuid.Parse(eventIDStr); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Event ID tidak valid")
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

	// count (alive only)
	var total int64
	if err := ctrl.DB.
		Model(&model.EventSessionModel{}).
		Where("event_session_event_id = ? AND event_session_deleted_at IS NULL", eventIDStr).
		Count(&total).Error; err != nil {
		log.Printf("[ERROR] count sessions: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung event sessions")
	}

	// data (alive only)
	var sessions []model.EventSessionModel
	if err := ctrl.DB.
		Where("event_session_event_id = ? AND event_session_deleted_at IS NULL", eventIDStr).
		Order("event_session_start_time ASC").
		Limit(limit).Offset(offset).
		Find(&sessions).Error; err != nil {
		log.Printf("[ERROR] get sessions: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil event sessions")
	}

	// map + status (computed)
	now := time.Now()
	out := make([]eventSessionView, 0, len(sessions))
	for i := range sessions {
		item := dto.ToEventSessionResponse(&sessions[i])
		out = append(out, eventSessionView{
			EventSessionResponse: *item,
			EventSessionStatus:   computeStatus(now, sessions[i].EventSessionStartTime, sessions[i].EventSessionEndTime),
		})
	}

	pagination := fiber.Map{
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": int((total + int64(limit) - 1) / int64(limit)),
		"has_next":    int64(page*limit) < total,
		"has_prev":    page > 1,
		"event_id":    eventIDStr,
	}
	return helper.JsonList(c, out, pagination)
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

	// count (alive only)
	var total int64
	if err := ctrl.DB.
		Model(&model.EventSessionModel{}).
		Where("event_session_deleted_at IS NULL").
		Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung event sessions")
	}

	// data (alive only)
	var sessions []model.EventSessionModel
	if err := ctrl.DB.
		Where("event_session_deleted_at IS NULL").
		Order("event_session_start_time DESC").
		Limit(limit).Offset(offset).
		Find(&sessions).Error; err != nil {
		log.Printf("[ERROR] get all sessions: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data event sessions")
	}

	// map + status
	now := time.Now()
	out := make([]eventSessionView, 0, len(sessions))
	for i := range sessions {
		item := dto.ToEventSessionResponse(&sessions[i])
		out = append(out, eventSessionView{
			EventSessionResponse: *item,
			EventSessionStatus:   computeStatus(now, sessions[i].EventSessionStartTime, sessions[i].EventSessionEndTime),
		})
	}

	pagination := fiber.Map{
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": int((total + int64(limit) - 1) / int64(limit)),
		"has_next":    int64(page*limit) < total,
		"has_prev":    page > 1,
	}
	return helper.JsonList(c, out, pagination)
}


// 🟢 GET /api/u/event-sessions/upcoming/:masjid_id?page=&limit=
// (kalau :masjid_id kosong, ambil semua publik upcoming)
func (ctrl *EventSessionController) GetUpcomingEventSessions(c *fiber.Ctx) error {
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

	now := time.Now()

	// base query: upcoming & alive
	query := ctrl.DB.
		Model(&model.EventSessionModel{}).
		Where("event_session_start_time > ? AND event_session_deleted_at IS NULL", now)

	masjidIDStr := c.Params("masjid_id")
	if masjidIDStr == "" {
		// tanpa masjid → hanya public
		query = query.Where("event_session_is_public = TRUE")
	} else {
		// filter by masjid_id
		if _, err := uuid.Parse(masjidIDStr); err != nil {
			log.Printf("[ERROR] invalid masjid_id: %v", err)
			return helper.JsonError(c, fiber.StatusBadRequest, "Format ID masjid tidak valid")
		}
		query = query.Where("event_session_masjid_id = ?", masjidIDStr)
	}

	// count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung sesi event")
	}

	// data
	var sessions []model.EventSessionModel
	if err := query.
		Order("event_session_start_time ASC").
		Limit(limit).Offset(offset).
		Find(&sessions).Error; err != nil {
		log.Printf("[ERROR] get upcoming sessions: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil sesi event upcoming")
	}

	// map
	out := make([]eventSessionView, 0, len(sessions))
	for i := range sessions {
		item := dto.ToEventSessionResponse(&sessions[i])
		out = append(out, eventSessionView{
			EventSessionResponse: *item,
			EventSessionStatus:   "upcoming", // by definition of the endpoint
		})
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
	return helper.JsonList(c, out, pagination)
}

// 🟢 PUT /api/a/event-sessions/:id
func (ctrl *EventSessionController) UpdateEventSession(c *fiber.Ctx) error {
	// Param
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Format ID tidak valid")
	}

	// Ambil session (alive only)
	var session model.EventSessionModel
	if err := ctrl.DB.
		Where("event_session_id = ? AND event_session_deleted_at IS NULL", id).
		First(&session).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "Event session tidak ditemukan")
	}

	// Authorization by masjid
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil || session.EventSessionMasjidID != masjidID {
		return helper.JsonError(c, fiber.StatusForbidden, "Tidak punya akses untuk update event session ini")
	}

	// Bind request
	var req dto.EventSessionUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Permintaan tidak valid")
	}

	// Apply partial fields (gunakan helper dari DTO agar konsisten)
	req.ApplyToModel(&session)

	// Validasi ringan
	if session.EventSessionEndTime.Before(session.EventSessionStartTime) || session.EventSessionEndTime.Equal(session.EventSessionStartTime) {
		return helper.JsonError(c, fiber.StatusBadRequest, "End time harus lebih besar dari start time")
	}
	if session.EventSessionCapacity != nil && *session.EventSessionCapacity < 0 {
		return helper.JsonError(c, fiber.StatusBadRequest, "Capacity tidak boleh negatif")
	}

	// Simpan
	if err := ctrl.DB.Save(&session).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengupdate event session")
	}

	return helper.JsonOK(c, "Event session berhasil diperbarui", dto.ToEventSessionResponse(&session))
}

// 🟢 DELETE /api/a/event-sessions/:id
func (ctrl *EventSessionController) DeleteEventSession(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Format ID tidak valid")
	}

	// Ambil session (alive only)
	var session model.EventSessionModel
	if err := ctrl.DB.
		Where("event_session_id = ? AND event_session_deleted_at IS NULL", id).
		First(&session).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "Event session tidak ditemukan")
	}

	// Authorization by masjid
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil || session.EventSessionMasjidID != masjidID {
		return helper.JsonError(c, fiber.StatusForbidden, "Tidak punya akses untuk menghapus event session ini")
	}

	// Soft delete (gorm.DeletedAt akan mengisi event_session_deleted_at)
	if err := ctrl.DB.Delete(&session).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus event session")
	}

	return helper.JsonOK(c, "Event session berhasil dihapus", nil)
}