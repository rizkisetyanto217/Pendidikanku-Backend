// file: internals/features/school/submissions_assesments/assesments/service/assessment_type_service.go
package service

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	model "madinahsalam_backend/internals/features/school/submissions_assesments/assesments/model"
)

/*
   Service untuk helper terkait AssessmentType:
   - SumActiveWeights: hitung total bobot type aktif (per school)
   - SyncAssessmentTypeSnapshot: sinkronkan snapshot scalar di assessments
*/

type Service struct{}

func New() *Service {
	return &Service{}
}

// SumActiveWeights menghitung total bobot semua assessment_type
// yang aktif untuk 1 school, optional exclude 1 type (saat PATCH).
func (s *Service) SumActiveWeights(
	db *gorm.DB,
	schoolID uuid.UUID,
	excludeTypeID *uuid.UUID,
) (float64, error) {
	var sum float64

	q := db.
		Model(&model.AssessmentTypeModel{}).
		Where(`
			assessment_type_school_id = ?
			AND assessment_type_is_active = TRUE
			AND assessment_type_deleted_at IS NULL
		`, schoolID)

	if excludeTypeID != nil && *excludeTypeID != uuid.Nil {
		q = q.Where("assessment_type_id <> ?", *excludeTypeID)
	}

	if err := q.
		Select("COALESCE(SUM(assessment_type_weight_percent), 0)").
		Scan(&sum).Error; err != nil {
		return 0, err
	}

	return sum, nil
}

// SyncAssessmentTypeSnapshot:
// Setelah sebuah AssessmentType diubah, method ini akan
// meng-update semua Assessment yang memakai type tersebut
// agar snapshot scalarnya konsisten:
//   - assessment_type_is_graded_snapshot
//   - assessment_allow_late_submission_snapshot
//   - assessment_late_penalty_percent_snapshot
//   - assessment_passing_score_percent_snapshot
//   - assessment_type_category_snapshot
func (s *Service) SyncAssessmentTypeSnapshot(
	db *gorm.DB,
	schoolID uuid.UUID,
	at *model.AssessmentTypeModel,
) error {
	if at == nil || at.AssessmentTypeID == uuid.Nil {
		return nil
	}

	updates := map[string]any{
		"assessment_type_is_graded_snapshot":        at.AssessmentTypeIsGraded,
		"assessment_allow_late_submission_snapshot": at.AssessmentTypeAllowLateSubmission,
		"assessment_late_penalty_percent_snapshot":  at.AssessmentTypeLatePenaltyPercent,
		"assessment_passing_score_percent_snapshot": at.AssessmentTypePassingScorePercent,
		"assessment_type_category_snapshot":         at.AssessmentTypeCategory,
	}

	return db.
		Model(&model.AssessmentModel{}).
		Where(`
			assessment_school_id = ?
			AND assessment_type_id = ?
			AND assessment_deleted_at IS NULL
		`, schoolID, at.AssessmentTypeID).
		Updates(updates).Error
}
