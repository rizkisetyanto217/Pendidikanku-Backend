// internals/features/lembaga/stats/dto/lembaga_stats_dto.go
package dto

import (
	"time"

	model "masjidku_backend/internals/features/lembaga/stats/lembaga_stats/model"

	"github.com/google/uuid"
)

/* ===================== RESPONSES ===================== */

type LembagaStatsResponse struct {
	LembagaStatsLembagaID      uuid.UUID  `json:"lembaga_stats_lembaga_id"`
	LembagaStatsActiveClasses  int        `json:"lembaga_stats_active_classes"`
	LembagaStatsActiveSections int        `json:"lembaga_stats_active_sections"`
	LembagaStatsActiveStudents int        `json:"lembaga_stats_active_students"`
	LembagaStatsActiveTeachers int        `json:"lembaga_stats_active_teachers"`
	LembagaStatsCreatedAt      time.Time  `json:"lembaga_stats_created_at"`
	LembagaStatsUpdatedAt      *time.Time `json:"lembaga_stats_updated_at,omitempty"`
}

func FromModel(m model.LembagaStats) LembagaStatsResponse {
	return LembagaStatsResponse{
		LembagaStatsLembagaID:      m.LembagaStatsLembagaID,
		LembagaStatsActiveClasses:  m.LembagaStatsActiveClasses,
		LembagaStatsActiveSections: m.LembagaStatsActiveSections,
		LembagaStatsActiveStudents: m.LembagaStatsActiveStudents,
		LembagaStatsActiveTeachers: m.LembagaStatsActiveTeachers,
		LembagaStatsCreatedAt:      m.LembagaStatsCreatedAt,
		LembagaStatsUpdatedAt:      m.LembagaStatsUpdatedAt,
	}
}

/* ===================== REQUESTS ===================== */

// Untuk inisialisasi (upsert/create) satu baris stats lembaga.
// Biasanya dipakai saat migrasi/seed, atau saat lembaga baru dibuat.
type UpsertLembagaStatsRequest struct {
	LembagaStatsLembagaID      uuid.UUID `json:"lembaga_stats_lembaga_id" validate:"required"`
	LembagaStatsActiveClasses  int       `json:"lembaga_stats_active_classes"  validate:"gte=0"`
	LembagaStatsActiveSections int       `json:"lembaga_stats_active_sections" validate:"gte=0"`
	LembagaStatsActiveStudents int       `json:"lembaga_stats_active_students" validate:"gte=0"`
	LembagaStatsActiveTeachers int       `json:"lembaga_stats_active_teachers" validate:"gte=0"`
}

// Partial update (PATCH) — semua field opsional.
// Gunakan pointer supaya bisa bedakan "0" vs "tidak diubah".
type UpdateLembagaStatsRequest struct {
	LembagaStatsActiveClasses  *int `json:"lembaga_stats_active_classes"  validate:"omitempty,gte=0"`
	LembagaStatsActiveSections *int `json:"lembaga_stats_active_sections" validate:"omitempty,gte=0"`
	LembagaStatsActiveStudents *int `json:"lembaga_stats_active_students" validate:"omitempty,gte=0"`
	LembagaStatsActiveTeachers *int `json:"lembaga_stats_active_teachers" validate:"omitempty,gte=0"`
}

/* ================ HELPERS (opsional) ================= */

func (r UpsertLembagaStatsRequest) ToModel() model.LembagaStats {
	now := time.Now()
	return model.LembagaStats{
		LembagaStatsLembagaID:      r.LembagaStatsLembagaID,
		LembagaStatsActiveClasses:  r.LembagaStatsActiveClasses,
		LembagaStatsActiveSections: r.LembagaStatsActiveSections,
		LembagaStatsActiveStudents: r.LembagaStatsActiveStudents,
		LembagaStatsActiveTeachers: r.LembagaStatsActiveTeachers,
		LembagaStatsCreatedAt:      now,
		LembagaStatsUpdatedAt:      &now,
	}
}

// Terapkan perubahan partial ke model (untuk handler PATCH).
func (r UpdateLembagaStatsRequest) ApplyToModel(m *model.LembagaStats) {
	if r.LembagaStatsActiveClasses != nil {
		m.LembagaStatsActiveClasses = *r.LembagaStatsActiveClasses
	}
	if r.LembagaStatsActiveSections != nil {
		m.LembagaStatsActiveSections = *r.LembagaStatsActiveSections
	}
	if r.LembagaStatsActiveStudents != nil {
		m.LembagaStatsActiveStudents = *r.LembagaStatsActiveStudents
	}
	if r.LembagaStatsActiveTeachers != nil {
		m.LembagaStatsActiveTeachers = *r.LembagaStatsActiveTeachers
	}
	now := time.Now()
	m.LembagaStatsUpdatedAt = &now
}
