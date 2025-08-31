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
		// UpdatedAt biarkan diisi trigger/UPDATE berikutnya
	}

	// INSERT ... ON CONFLICT DO NOTHING (PK = lembaga_stats_masjid_id)
	return tx.
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&row).Error
}

// --- helper privat: pastikan ada + kunci baris (hindari race) ---
func (s *LembagaStatsService) ensureAndLock(tx *gorm.DB, masjidID uuid.UUID) error {
	// ensure row exists
	if err := s.EnsureForMasjid(tx, masjidID); err != nil {
		return err
	}
	// lock the row so concurrent updates don't race
	return tx.Exec(`
		SELECT 1 FROM lembaga_stats
		WHERE lembaga_stats_masjid_id = ?
		FOR UPDATE
	`, masjidID).Error
}

// --- Versi generic: apply delta ke semua kolom dengan clamp >= 0 ---
type Delta struct {
	Classes  int
	Sections int
	Students int
	Teachers int
}

func (s *LembagaStatsService) ApplyDelta(tx *gorm.DB, masjidID uuid.UUID, d Delta) error {
	if err := s.ensureAndLock(tx, masjidID); err != nil {
		return err
	}
	return tx.Exec(`
		UPDATE lembaga_stats
		SET
			lembaga_stats_active_classes  = GREATEST(lembaga_stats_active_classes  + ?, 0),
			lembaga_stats_active_sections = GREATEST(lembaga_stats_active_sections + ?, 0),
			lembaga_stats_active_students = GREATEST(lembaga_stats_active_students + ?, 0),
			lembaga_stats_active_teachers = GREATEST(lembaga_stats_active_teachers + ?, 0),
			lembaga_stats_updated_at = NOW()
		WHERE lembaga_stats_masjid_id = ?
	`, d.Classes, d.Sections, d.Students, d.Teachers, masjidID).Error
}

// --- API lama (compat): masing-masing kolom ---
// Semua pakai clamp >= 0 + row lock di belakang layar.

func (s *LembagaStatsService) IncActiveClasses(tx *gorm.DB, masjidID uuid.UUID, delta int) error {
	return s.ApplyDelta(tx, masjidID, Delta{Classes: delta})
}

func (s *LembagaStatsService) IncActiveSections(tx *gorm.DB, masjidID uuid.UUID, delta int) error {
	return s.ApplyDelta(tx, masjidID, Delta{Sections: delta})
}

func (s *LembagaStatsService) IncActiveStudents(tx *gorm.DB, masjidID uuid.UUID, delta int) error {
	return s.ApplyDelta(tx, masjidID, Delta{Students: delta})
}

func (s *LembagaStatsService) IncActiveTeachers(tx *gorm.DB, masjidID uuid.UUID, delta int) error {
	return s.ApplyDelta(tx, masjidID, Delta{Teachers: delta})
}
