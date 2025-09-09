// file: internals/features/school/submissions_assesments/assesments/service/assessment_type_service.go
package service

import (
	"masjidku_backend/internals/features/school/submissions_assesments/assesments/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SumActiveWeights menjumlahkan bobot semua assessment type AKTIF untuk satu masjid.
// - db: boleh *gorm.DB biasa atau transaction (tx)
// - masjidID: scope tenant
// - excludeID: kalau diisi, baris itu tidak ikut dihitung (berguna saat PATCH)
func SumActiveWeights(db *gorm.DB, masjidID uuid.UUID, excludeID *uuid.UUID) (float64, error) {
	var sum float64
	q := db.
		Model(&model.AssessmentTypeModel{}).
		Where("assessment_types_masjid_id = ? AND assessment_types_is_active = TRUE", masjidID).
		Select("COALESCE(SUM(assessment_types_weight_percent), 0)")

	if excludeID != nil && *excludeID != uuid.Nil {
		q = q.Where("assessment_types_id <> ?", *excludeID)
	}
	if err := q.Scan(&sum).Error; err != nil {
		return 0, err
	}
	return sum, nil
}
