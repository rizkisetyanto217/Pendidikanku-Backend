// file: internals/features/school/submissions_assesments/assesments/service/sum_assesment_service.go
package service

// import (
// 	"github.com/google/uuid"
// 	"gorm.io/gorm"
// )

// type Service struct{}

// func New() *Service { return &Service{} }

// // SumActiveWeights menghitung total bobot tipe assessment AKTIF untuk 1 school.
// // excludeID opsional: tidak ikut menghitung row dengan id tsb (berguna saat PATCH).
// func (s *Service) SumActiveWeights(db *gorm.DB, schoolID uuid.UUID, excludeID *uuid.UUID) (float64, error) {
// 	var sum float64

// 	q := db.
// 		Table("assessment_types").
// 		Select("COALESCE(SUM(assessment_type_weight_percent), 0)").
// 		Where(`
// 			assessment_type_school_id = ?
// 			AND assessment_type_is_active = TRUE
// 			AND assessment_type_deleted_at IS NULL
// 		`, schoolID)

// 	if excludeID != nil {
// 		q = q.Where("assessment_type_id <> ?", *excludeID)
// 	}

// 	if err := q.Scan(&sum).Error; err != nil {
// 		return 0, err
// 	}
// 	return sum, nil
// }
