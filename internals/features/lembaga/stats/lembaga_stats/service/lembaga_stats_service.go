// internals/features/lembaga/stats/lembaga_stats/service/lembaga_stats_service.go
package service

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	statsModel "masjidku_backend/internals/features/lembaga/stats/lembaga_stats/model"
)

type LembagaStatsService struct{}

func NewLembagaStatsService() *LembagaStatsService { return &LembagaStatsService{} }

// Pastikan baris lembaga_stats ada untuk masjid ini (idempotent & race-safe).
func (s *LembagaStatsService) EnsureForMasjid(tx *gorm.DB, masjidID uuid.UUID) error {
	now := time.Now()
	row := statsModel.LembagaStats{
		LembagaStatsMasjidID:      masjidID,
		LembagaStatsActiveClasses:  0,
		LembagaStatsActiveSections: 0,
		LembagaStatsActiveStudents: 0,
		LembagaStatsActiveTeachers: 0,
		LembagaStatsCreatedAt:      now,
		// UpdatedAt dibiarkan NULL sampai ada perubahan
	}

	// INSERT ... ON CONFLICT DO NOTHING (PK = lembaga_stats_masjid_id)
	return tx.
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&row).Error
}

// Tambah/kurangi jumlah kelas aktif secara atomik.
func (s *LembagaStatsService) IncActiveClasses(tx *gorm.DB, masjidID uuid.UUID, delta int) error {
	return tx.Model(&statsModel.LembagaStats{}).
		Where("lembaga_stats_masjid_id = ?", masjidID).
		Updates(map[string]interface{}{
			"lembaga_stats_active_classes": gorm.Expr("lembaga_stats_active_classes + ?", delta),
			"lembaga_stats_updated_at":     gorm.Expr("CURRENT_TIMESTAMP"),
		}).Error
}


func (s *LembagaStatsService) IncActiveSections(tx *gorm.DB, masjidID uuid.UUID, delta int) error {
	return tx.Model(&statsModel.LembagaStats{}).
		Where("lembaga_stats_masjid_id = ?", masjidID).
		Updates(map[string]any{
			"lembaga_stats_active_sections": gorm.Expr("lembaga_stats_active_sections + ?", delta),
			"lembaga_stats_updated_at":      gorm.Expr("CURRENT_TIMESTAMP"),
		}).Error
}


// internals/features/lembaga/stats/lembaga_stats/service/lembaga_stats_service.go

func (s *LembagaStatsService) IncActiveStudents(tx *gorm.DB, masjidID uuid.UUID, delta int) error {
	return tx.Model(&statsModel.LembagaStats{}).
		Where("lembaga_stats_masjid_id = ?", masjidID).
		Updates(map[string]any{
			"lembaga_stats_active_students": gorm.Expr("lembaga_stats_active_students + ?", delta),
			"lembaga_stats_updated_at":      gorm.Expr("CURRENT_TIMESTAMP"),
		}).Error
}


// internals/features/lembaga/stats/lembaga_stats/service/lembaga_stats_service.go
func (s *LembagaStatsService) IncActiveTeachers(tx *gorm.DB, masjidID uuid.UUID, delta int) error {
	return tx.Model(&statsModel.LembagaStats{}).
		Where("lembaga_stats_masjid_id = ?", masjidID).
		Updates(map[string]any{
			"lembaga_stats_active_teachers": gorm.Expr("lembaga_stats_active_teachers + ?", delta),
			"lembaga_stats_updated_at":      gorm.Expr("CURRENT_TIMESTAMP"),
		}).Error
}
