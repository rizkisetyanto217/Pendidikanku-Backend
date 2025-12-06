// file: internals/features/finance/payments/controller/payment_gateway_event_controller.go
package controller

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	dto "madinahsalam_backend/internals/features/finance/payments/dto"
	model "madinahsalam_backend/internals/features/finance/payments/model"
)

/* =======================================================================
   Controller
======================================================================= */

type PaymentGatewayEventController struct {
	DB *gorm.DB
}

func NewPaymentGatewayEventController(db *gorm.DB) *PaymentGatewayEventController {
	return &PaymentGatewayEventController{DB: db}
}

func (h *PaymentGatewayEventController) RegisterRoutes(r fiber.Router) {
	gr := r.Group("/payment-gateway-events")
	gr.Get("/", h.ListEvents)      // GET /payment-gateway-events?provider=&status=&payment_id=&school_id=&q=&start=&end=&page=&limit=
	gr.Get("/:id", h.GetByID)      // GET /payment-gateway-events/:id
	gr.Post("/", h.CreateEvent)    // POST /payment-gateway-events
	gr.Patch("/:id", h.PatchEvent) // PATCH /payment-gateway-events/:id
}

/* =======================================================================
   List (filter + pagination)
   Query params:
     - provider: midtrans|xendit|...
     - status: received|processing|success|failed
     - payment_id: uuid
     - school_id: uuid
     - q: cari di external_id / external_ref (ilike)
     - start, end: ISO8601 (filter received_at)
     - page (default 1), limit (default 20, max 200)
======================================================================= */

func (h *PaymentGatewayEventController) ListEvents(c *fiber.Ctx) error {
	db := h.DB.Model(&model.PaymentGatewayEventModel{}).
		Where("gateway_event_deleted_at IS NULL")

	if p := strings.TrimSpace(c.Query("provider")); p != "" {
		db = db.Where("gateway_event_provider = ?", strings.ToLower(p))
	}
	if s := strings.TrimSpace(c.Query("status")); s != "" {
		db = db.Where("gateway_event_status = ?", strings.ToLower(s))
	}
	if pid := strings.TrimSpace(c.Query("payment_id")); pid != "" {
		if id, err := uuid.Parse(pid); err == nil {
			db = db.Where("gateway_event_payment_id = ?", id)
		} else {
			return fiber.NewError(fiber.StatusBadRequest, "invalid payment_id")
		}
	}
	if sid := strings.TrimSpace(c.Query("school_id")); sid != "" {
		if id, err := uuid.Parse(sid); err == nil {
			db = db.Where("gateway_event_school_id = ?", id)
		} else {
			return fiber.NewError(fiber.StatusBadRequest, "invalid school_id")
		}
	}

	// search di external_id / external_ref
	if q := strings.TrimSpace(c.Query("q")); q != "" {
		like := "%" + q + "%"
		db = db.Where(`
			COALESCE(gateway_event_external_id,'') ILIKE ? 
			OR COALESCE(gateway_event_external_ref,'') ILIKE ?
		`, like, like)
	}

	// time range by received_at
	if start := strings.TrimSpace(c.Query("start")); start != "" {
		if t, err := time.Parse(time.RFC3339, start); err == nil {
			db = db.Where("gateway_event_received_at >= ?", t)
		} else {
			return fiber.NewError(fiber.StatusBadRequest, "invalid start (use RFC3339)")
		}
	}
	if end := strings.TrimSpace(c.Query("end")); end != "" {
		if t, err := time.Parse(time.RFC3339, end); err == nil {
			db = db.Where("gateway_event_received_at < ?", t)
		} else {
			return fiber.NewError(fiber.StatusBadRequest, "invalid end (use RFC3339)")
		}
	}

	// pagination
	page := clampInt(queryInt(c, "page", 1), 1, 1_000_000)
	limit := clampInt(queryInt(c, "limit", 20), 1, 200)
	offset := (page - 1) * limit

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	var rows []model.PaymentGatewayEventModel
	if err := db.Order("gateway_event_received_at DESC").
		Limit(limit).Offset(offset).
		Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	out := make([]*dto.PaymentGatewayEventResponse, 0, len(rows))
	for i := range rows {
		out = append(out, dto.FromModelPGW(&rows[i]))
	}

	return c.JSON(fiber.Map{
		"page":  page,
		"limit": limit,
		"total": total,
		"data":  out,
	})
}

/* =======================================================================
   Detail
======================================================================= */

func (h *PaymentGatewayEventController) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	var m model.PaymentGatewayEventModel
	if err := h.DB.First(
		&m,
		"gateway_event_id = ? AND gateway_event_deleted_at IS NULL",
		id,
	).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "event not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(dto.FromModelPGW(&m))
}

/* =======================================================================
   Create (manual insert event)
======================================================================= */

func (h *PaymentGatewayEventController) CreateEvent(c *fiber.Ctx) error {
	var req dto.CreatePaymentGatewayEventRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid json: "+err.Error())
	}
	if err := req.Validate(); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	m := req.ToModel()
	if err := h.DB.Create(m).Error; err != nil {
		// handle duplicate unique (provider, external_id) gracefully
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			return fiber.NewError(fiber.StatusConflict, "duplicated provider+external_id")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(dto.FromModelPGW(m))
}

/* =======================================================================
   Patch (tri-state)
======================================================================= */

func (h *PaymentGatewayEventController) PatchEvent(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	var m model.PaymentGatewayEventModel
	if err := h.DB.First(
		&m,
		"gateway_event_id = ? AND gateway_event_deleted_at IS NULL",
		id,
	).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "event not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	var patch dto.UpdatePaymentGatewayEventRequest
	if err := c.BodyParser(&patch); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid json: "+err.Error())
	}

	if err := patch.Apply(&m); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Auto-set processed_at:
	// kalau status sudah success/failed dan processed_at masih null → isi sekarang
	if (m.GatewayEventStatus == model.GatewayEventStatusSuccess ||
		m.GatewayEventStatus == model.GatewayEventStatusFailed) &&
		m.GatewayEventProcessedAt == nil {
		now := time.Now().UTC()
		m.GatewayEventProcessedAt = &now
	}

	if err := h.DB.Save(&m).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(dto.FromModelPGW(&m))
}

/* =======================================================================
   Helpers
======================================================================= */

func queryInt(c *fiber.Ctx, key string, def int) int {
	if v := c.Query(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
