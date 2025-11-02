// internals/features/lembaga/stats/lembaga_stats/service/lembaga_stats_service.go
package service

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	statsModel "schoolku_backend/internals/features/lembaga/stats/lembaga_stats/model"
)

type LembagaStatsService struct{}

func NewLembagaStatsService() *LembagaStatsService { return &LembagaStatsService{} }

// Pastikan baris lembaga_stats ada untuk school ini (idempotent & race-safe).
func (s *LembagaStatsService) EnsureForSchool(tx *gorm.DB, schoolID uuid.UUID) error {
	now := time.Now()
	row := statsModel.LembagaStats{
		LembagaStatsSchoolID:       schoolID,
		LembagaStatsActiveClasses:  0,
		LembagaStatsActiveSections: 0,
		LembagaStatsActiveStudents: 0,
		LembagaStatsActiveTeachers: 0,
		LembagaStatsCreatedAt:      now,
		// UpdatedAt biarkan diisi trigger/UPDATE berikutnya
	}

	// INSERT ... ON CONFLICT DO NOTHING (PK = lembaga_stats_school_id)
	return tx.
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&row).Error
}

// --- helper privat: pastikan ada + kunci baris (hindari race) ---
func (s *LembagaStatsService) ensureAndLock(tx *gorm.DB, schoolID uuid.UUID) error {
	// ensure row exists
	if err := s.EnsureForSchool(tx, schoolID); err != nil {
		return err
	}
	// lock the row so concurrent updates don't race
	return tx.Exec(`
		SELECT 1 FROM lembaga_stats
		WHERE lembaga_stats_school_id = ?
		FOR UPDATE
	`, schoolID).Error
}

// --- Versi generic: apply delta ke semua kolom dengan clamp >= 0 ---
type Delta struct {
	Classes  int
	Sections int
	Students int
	Teachers int
}

func (s *LembagaStatsService) ApplyDelta(tx *gorm.DB, schoolID uuid.UUID, d Delta) error {
	if err := s.ensureAndLock(tx, schoolID); err != nil {
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
		WHERE lembaga_stats_school_id = ?
	`, d.Classes, d.Sections, d.Students, d.Teachers, schoolID).Error
}

// --- API lama (compat): masing-masing kolom ---
// Semua pakai clamp >= 0 + row lock di belakang layar.

func (s *LembagaStatsService) IncActiveClasses(tx *gorm.DB, schoolID uuid.UUID, delta int) error {
	return s.ApplyDelta(tx, schoolID, Delta{Classes: delta})
}

func (s *LembagaStatsService) IncActiveSections(tx *gorm.DB, schoolID uuid.UUID, delta int) error {
	return s.ApplyDelta(tx, schoolID, Delta{Sections: delta})
}

func (s *LembagaStatsService) IncActiveStudents(tx *gorm.DB, schoolID uuid.UUID, delta int) error {
	return s.ApplyDelta(tx, schoolID, Delta{Students: delta})
}

func (s *LembagaStatsService) IncActiveTeachers(tx *gorm.DB, schoolID uuid.UUID, delta int) error {
	return s.ApplyDelta(tx, schoolID, Delta{Teachers: delta})
}
