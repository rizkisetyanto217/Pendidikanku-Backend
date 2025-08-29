package service

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	openingModel "masjidku_backend/internals/features/school/academics/academic_terms/model"
)

// Interface supaya gampang di-mock
type OpeningQuotaService interface {
	EnsureOpeningBelongsToMasjid(tx *gorm.DB, openingID, masjidID uuid.UUID) error
	Claim(tx *gorm.DB, openingID uuid.UUID) error
	Release(tx *gorm.DB, openingID uuid.UUID) error
}

type openingQuotaSvc struct{}

func NewOpeningQuotaService() OpeningQuotaService {
	return &openingQuotaSvc{}
}

func (s *openingQuotaSvc) EnsureOpeningBelongsToMasjid(tx *gorm.DB, openingID, masjidID uuid.UUID) error {
	var cnt int64
	if err := tx.Model(&openingModel.ClassTermOpeningModel{}).
		Where("class_term_openings_id = ? AND class_term_openings_masjid_id = ? AND class_term_openings_deleted_at IS NULL",
			openingID, masjidID).
		Count(&cnt).Error; err != nil {
		log.Printf("[OpeningQuota] ERROR EnsureOpeningBelongsToMasjid openingID=%s masjidID=%s err=%v", openingID, masjidID, err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi opening: "+err.Error())
	}
	if cnt == 0 {
		log.Printf("[OpeningQuota] NOT FOUND EnsureOpeningBelongsToMasjid openingID=%s masjidID=%s", openingID, masjidID)
		return fiber.NewError(fiber.StatusNotFound, "Opening tidak ditemukan di masjid ini")
	}
	log.Printf("[OpeningQuota] OK EnsureOpeningBelongsToMasjid openingID=%s masjidID=%s", openingID, masjidID)
	return nil
}

func (s *openingQuotaSvc) Claim(tx *gorm.DB, openingID uuid.UUID) error {
	now := time.Now()

	// Coba increment untuk opening yang BERKUOTA
	inc := tx.Model(&openingModel.ClassTermOpeningModel{}).
		Where("class_term_openings_id = ? AND class_term_openings_deleted_at IS NULL", openingID).
		Where("class_term_openings_is_open = TRUE").
		Where("(class_term_openings_registration_opens_at IS NULL OR ? >= class_term_openings_registration_opens_at)", now).
		Where("(class_term_openings_registration_closes_at IS NULL OR ? <= class_term_openings_registration_closes_at)", now).
		Where("class_term_openings_quota_total IS NOT NULL").
		Where("class_term_openings_quota_taken < class_term_openings_quota_total").
		Updates(map[string]any{
			"class_term_openings_quota_taken": gorm.Expr("class_term_openings_quota_taken + 1"),
			"class_term_openings_updated_at":  now,
		})
	if inc.Error != nil {
		log.Printf("[OpeningQuota] ERROR Claim increment openingID=%s err=%v", openingID, inc.Error)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal klaim kuota: "+inc.Error.Error())
	}
	if inc.RowsAffected == 1 {
		log.Printf("[OpeningQuota] SUCCESS Claim increment openingID=%s", openingID)
		return nil
	}

	// Cek unlimited
	var cnt int64
	if err := tx.Model(&openingModel.ClassTermOpeningModel{}).
		Where("class_term_openings_id = ? AND class_term_openings_deleted_at IS NULL", openingID).
		Where("class_term_openings_is_open = TRUE").
		Where("(class_term_openings_registration_opens_at IS NULL OR ? >= class_term_openings_registration_opens_at)", now).
		Where("(class_term_openings_registration_closes_at IS NULL OR ? <= class_term_openings_registration_closes_at)", now).
		Where("class_term_openings_quota_total IS NULL").
		Count(&cnt).Error; err != nil {
		log.Printf("[OpeningQuota] ERROR Claim validate unlimited openingID=%s err=%v", openingID, err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi opening: "+err.Error())
	}
	if cnt == 1 {
		touch := tx.Model(&openingModel.ClassTermOpeningModel{}).
			Where("class_term_openings_id = ?", openingID).
			UpdateColumn("class_term_openings_updated_at", now)
		if touch.Error != nil {
			log.Printf("[OpeningQuota] ERROR Claim touch unlimited openingID=%s err=%v", openingID, touch.Error)
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal update opening: "+touch.Error.Error())
		}
		log.Printf("[OpeningQuota] SUCCESS Claim unlimited openingID=%s", openingID)
		return nil
	}

	log.Printf("[OpeningQuota] FAIL Claim openingID=%s (quota penuh / closed)", openingID)
	return fiber.NewError(fiber.StatusConflict, "Kuota penuh atau opening tidak tersedia")
}

func (s *openingQuotaSvc) Release(tx *gorm.DB, openingID uuid.UUID) error {
	now := time.Now()
	dec := tx.Model(&openingModel.ClassTermOpeningModel{}).
		Where("class_term_openings_id = ? AND class_term_openings_deleted_at IS NULL", openingID).
		Where("class_term_openings_quota_total IS NOT NULL").
		UpdateColumn("class_term_openings_quota_taken", gorm.Expr("GREATEST(class_term_openings_quota_taken - 1, 0)"))
	if dec.Error != nil {
		log.Printf("[OpeningQuota] ERROR Release decrement openingID=%s err=%v", openingID, dec.Error)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal merilis kuota: "+dec.Error.Error())
	}
	log.Printf("[OpeningQuota] SUCCESS Release decrement openingID=%s rows=%d", openingID, dec.RowsAffected)

	if err := tx.Model(&openingModel.ClassTermOpeningModel{}).
		Where("class_term_openings_id = ?", openingID).
		UpdateColumn("class_term_openings_updated_at", now).Error; err != nil {
		log.Printf("[OpeningQuota] ERROR Release update updated_at openingID=%s err=%v", openingID, err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal update opening: "+err.Error())
	}
	log.Printf("[OpeningQuota] SUCCESS Release touch updated_at openingID=%s", openingID)
	return nil
}
