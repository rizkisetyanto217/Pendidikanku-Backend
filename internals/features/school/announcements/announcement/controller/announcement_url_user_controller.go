package controller

import (
	"masjidku_backend/internals/features/school/announcements/announcement/dto"
	"masjidku_backend/internals/features/school/announcements/announcement/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

/* =========================================================
   LIST
   GET /api/a/announcement-urls?announcement_id=...&q=...&with_deleted=false
========================================================= */
func (ctl *AnnouncementURLController) List(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	q := strings.TrimSpace(c.Query("q"))
	withDeleted := strings.EqualFold(c.Query("with_deleted"), "true")

	var items []model.AnnouncementURLModel
	tx := ctl.DB.WithContext(c.Context()).
		Where("announcement_url_masjid_id = ?", masjidID)

	if annIDStr := strings.TrimSpace(c.Query("announcement_id")); annIDStr != "" {
		annID, perr := uuid.Parse(annIDStr)
		if perr == nil {
			tx = tx.Where("announcement_url_announcement_id = ?", annID)
		}
	}

	if q != "" {
		like := "%" + q + "%"
		tx = tx.Where(
			"(announcement_url_label ILIKE ? OR announcement_url_href ILIKE ?)",
			like, like,
		)
	}

	if !withDeleted {
		tx = tx.Where("announcement_url_deleted_at IS NULL")
	}

	if err := tx.Order("announcement_url_created_at DESC").
		Find(&items).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// map ke response
	out := make([]dto.AnnouncementURLResponse, 0, len(items))
	for _, it := range items {
		out = append(out, dto.AnnouncementURLResponse{
			AnnouncementURLID:                 it.AnnouncementURLID,
			AnnouncementURLMasjidID:           it.AnnouncementURLMasjidID,
			AnnouncementURLAnnouncementID:     it.AnnouncementURLAnnouncementID,
			AnnouncementURLLabel:              it.AnnouncementURLLabel,
			AnnouncementURLHref:               it.AnnouncementURLHref,
			AnnouncementURLTrashURL:           it.AnnouncementURLTrashURL,
			AnnouncementURLDeletePendingUntil: it.AnnouncementURLDeletePendingUntil,
			AnnouncementURLCreatedAt:          it.AnnouncementURLCreatedAt,
			AnnouncementURLUpdatedAt:          it.AnnouncementURLUpdatedAt,
			AnnouncementURLDeletedAt:          it.AnnouncementURLDeletedAt,
		})
	}

	// belum ada pagination → kirim nil / fiber.Map{} sebagai placeholder
	return helper.JsonList(c, out, nil)
}
