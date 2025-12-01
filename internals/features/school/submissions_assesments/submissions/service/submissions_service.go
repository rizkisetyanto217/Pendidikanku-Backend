// file: internals/features/school/assessments/submissions/service/submission_service.go
package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	qmodel "madinahsalam_backend/internals/features/school/submissions_assesments/quizzes/model"
	smodel "madinahsalam_backend/internals/features/school/submissions_assesments/submissions/model"
)

type SubmissionService struct {
	DB *gorm.DB
}

func NewSubmissionService(db *gorm.DB) *SubmissionService {
	return &SubmissionService{DB: db}
}

// UpsertSubmissionFromQuizAttempt:
// - diasumsikan 1 assessment = 1 quiz
// - pakai last_percent sebagai SubmissionScore
func (s *SubmissionService) UpsertSubmissionFromQuizAttempt(
	ctx context.Context,
	attempt *qmodel.StudentQuizAttemptModel,
) error {
	if attempt == nil {
		return nil
	}

	// 1) Ambil assessment_id dari quiz
	var assessmentID uuid.UUID
	if err := s.DB.WithContext(ctx).Raw(`
		SELECT quiz_assessment_id
		FROM quizzes
		WHERE quiz_id = ? AND quiz_school_id = ? AND quiz_deleted_at IS NULL
	`, attempt.StudentQuizAttemptQuizID, attempt.StudentQuizAttemptSchoolID).
		Scan(&assessmentID).Error; err != nil {
		return err
	}
	if assessmentID == uuid.Nil {
		// quiz tidak terhubung ke assessment → tidak perlu bikin submission
		return nil
	}

	// 2) Cari existing submission (1 row per student×assessment×school)
	var sub smodel.SubmissionModel
	err := s.DB.WithContext(ctx).
		Where(`
			submission_school_id = ?
			AND submission_assessment_id = ?
			AND submission_student_id = ?
		`,
			attempt.StudentQuizAttemptSchoolID,
			assessmentID,
			attempt.StudentQuizAttemptStudentID,
		).
		First(&sub).Error

	now := time.Now().UTC()

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 3a) Belum ada → buat baru
			status := smodel.SubmissionStatusSubmitted
			isLate := (*bool)(nil)

			sub = smodel.SubmissionModel{
				SubmissionSchoolID:     attempt.StudentQuizAttemptSchoolID,
				SubmissionAssessmentID: assessmentID,
				SubmissionStudentID:    attempt.StudentQuizAttemptStudentID,
				SubmissionStatus:       status,
				SubmissionSubmittedAt:  &now,
				SubmissionIsLate:       isLate,
			}

			// Kalau mau, pakai last_percent sebagai skor
			if attempt.StudentQuizAttemptLastPercent != nil {
				sub.SubmissionScore = attempt.StudentQuizAttemptLastPercent
			}

			// Satu quiz dianggap finished
			sub.SubmissionQuizFinished = 1

			return s.DB.WithContext(ctx).Create(&sub).Error
		}
		return err
	}

	// 3b) Sudah ada submission → update
	// Jika nanti ada multi-quiz, di sini kamu bisa:
	// - baca SubmissionScores JSON
	// - merge dengan score quiz ini
	// - hitung ulang SubmissionScore total
	status := smodel.SubmissionStatusSubmitted
	sub.SubmissionStatus = status
	sub.SubmissionSubmittedAt = &now

	if attempt.StudentQuizAttemptLastPercent != nil {
		sub.SubmissionScore = attempt.StudentQuizAttemptLastPercent
	}

	// Sementara: anggap 1 quiz, jadi selalu 1
	sub.SubmissionQuizFinished = 1

	return s.DB.WithContext(ctx).Save(&sub).Error
}
