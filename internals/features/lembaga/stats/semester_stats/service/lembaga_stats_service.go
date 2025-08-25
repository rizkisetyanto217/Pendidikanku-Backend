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

/* ---------------------------------------------------
   Ensure: pastikan 1 baris per masjid (idempotent)
--------------------------------------------------- */
func (s *LembagaStatsService) EnsureForMasjid(tx *gorm.DB, masjidID uuid.UUID) error {
	now := time.Now()
	row := statsModel.LembagaStats{
		LembagaStatsMasjidID:      masjidID,
		LembagaStatsActiveClasses:  0,
		LembagaStatsActiveSections: 0,
		LembagaStatsActiveStudents: 0,
		LembagaStatsActiveTeachers: 0,
		LembagaStatsCreatedAt:      now,
	}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error
}

/* ---------------------------------------------------
   Helpers: increment/decrement yang anti minus
--------------------------------------------------- */
func (s *LembagaStatsService) incField(tx *gorm.DB, masjidID uuid.UUID, field string, delta int) error {
	return tx.Model(&statsModel.LembagaStats{}).
		Where("lembaga_stats_masjid_id = ?", masjidID).
		Updates(map[string]any{
			field:                     gorm.Expr("CASE WHEN "+field+" + ? < 0 THEN 0 ELSE "+field+" + ? END", delta, delta),
			"lembaga_stats_updated_at": gorm.Expr("CURRENT_TIMESTAMP"),
		}).Error
}

func (s *LembagaStatsService) IncActiveClasses(tx *gorm.DB, masjidID uuid.UUID, delta int) error {
	return s.incField(tx, masjidID, "lembaga_stats_active_classes", delta)
}
func (s *LembagaStatsService) IncActiveSections(tx *gorm.DB, masjidID uuid.UUID, delta int) error {
	return s.incField(tx, masjidID, "lembaga_stats_active_sections", delta)
}
func (s *LembagaStatsService) IncActiveStudents(tx *gorm.DB, masjidID uuid.UUID, delta int) error {
	return s.incField(tx, masjidID, "lembaga_stats_active_students", delta)
}
func (s *LembagaStatsService) IncActiveTeachers(tx *gorm.DB, masjidID uuid.UUID, delta int) error {
	return s.incField(tx, masjidID, "lembaga_stats_active_teachers", delta)
}

/* ---------------------------------------------------
   Setters / Getters / Recompute
--------------------------------------------------- */
func (s *LembagaStatsService) SetCounts(tx *gorm.DB, masjidID uuid.UUID, classes, sections, students, teachers int) error {
	if classes < 0 { classes = 0 }
	if sections < 0 { sections = 0 }
	if students < 0 { students = 0 }
	if teachers < 0 { teachers = 0 }
	return tx.Model(&statsModel.LembagaStats{}).
		Where("lembaga_stats_masjid_id = ?", masjidID).
		Updates(map[string]any{
			"lembaga_stats_active_classes":  classes,
			"lembaga_stats_active_sections": sections,
			"lembaga_stats_active_students": students,
			"lembaga_stats_active_teachers": teachers,
			"lembaga_stats_updated_at":      gorm.Expr("CURRENT_TIMESTAMP"),
		}).Error
}

func (s *LembagaStatsService) Get(tx *gorm.DB, masjidID uuid.UUID) (*statsModel.LembagaStats, error) {
	var row statsModel.LembagaStats
	if err := tx.Where("lembaga_stats_masjid_id = ?", masjidID).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *LembagaStatsService) GetForUpdate(tx *gorm.DB, masjidID uuid.UUID) (*statsModel.LembagaStats, error) {
	var row statsModel.LembagaStats
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("lembaga_stats_masjid_id = ?", masjidID).
		First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

type RecomputeResult struct {
	Classes, Sections, Students, Teachers int
}

func (s *LembagaStatsService) RecomputeFromSources(tx *gorm.DB, masjidID uuid.UUID) (RecomputeResult, error) {
	r := RecomputeResult{}

	if err := tx.Raw(`
		SELECT COALESCE(COUNT(*),0) AS n
		FROM classes
		WHERE classes_masjid_id = ?
		  AND (classes_is_active IS TRUE OR classes_is_active IS NULL)
		  AND classes_deleted_at IS NULL
	`, masjidID).Scan(&struct{ N *int }{&r.Classes}).Error; err != nil {
		return r, err
	}

	if err := tx.Raw(`
		SELECT COALESCE(COUNT(*),0) AS n
		FROM class_sections
		WHERE class_sections_masjid_id = ?
		  AND (class_sections_is_active IS TRUE OR class_sections_is_active IS NULL)
		  AND class_sections_deleted_at IS NULL
	`, masjidID).Scan(&struct{ N *int }{&r.Sections}).Error; err != nil {
		return r, err
	}

	if err := tx.Raw(`
		SELECT COALESCE(COUNT(*),0) AS n
		FROM user_classes uc
		JOIN classes c ON c.classes_id = uc.user_classes_class_id
		WHERE c.classes_masjid_id = ?
		  AND c.classes_deleted_at IS NULL
		  AND uc.user_classes_status = 'active'
	`, masjidID).Scan(&struct{ N *int }{&r.Students}).Error; err != nil {
		return r, err
	}

	if err := tx.Raw(`
		SELECT COALESCE(COUNT(*),0) AS n
		FROM masjid_teachers
		WHERE masjid_teachers_masjid_id = ?
		  AND masjid_teachers_deleted_at IS NULL
	`, masjidID).Scan(&struct{ N *int }{&r.Teachers}).Error; err != nil {
		return r, err
	}

	if err := s.SetCounts(tx, masjidID, r.Classes, r.Sections, r.Students, r.Teachers); err != nil {
		return r, err
	}
	return r, nil
}

func (s *LembagaStatsService) Bootstrap(tx *gorm.DB, masjidID uuid.UUID) error {
	if err := s.EnsureForMasjid(tx, masjidID); err != nil {
		return err
	}
	_, err := s.RecomputeFromSources(tx, masjidID)
	return err
}

func (s *LembagaStatsService) TouchUpdatedAt(tx *gorm.DB, masjidID uuid.UUID) error {
	return tx.Model(&statsModel.LembagaStats{}).
		Where("lembaga_stats_masjid_id = ?", masjidID).
		Update("lembaga_stats_updated_at", gorm.Expr("CURRENT_TIMESTAMP")).Error
}
