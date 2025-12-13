// file: internals/features/school/submissions_assesments/submissions/service/submission_service.go
package service

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	smodel "madinahsalam_backend/internals/features/school/submissions_assesments/submissions/model"
	qmodel "madinahsalam_backend/internals/features/school/submissions_assesments/quizzes/model"
)

type SubmissionService struct {
	DB *gorm.DB
}

func NewSubmissionService(db *gorm.DB) *SubmissionService {
	return &SubmissionService{DB: db}
}

// UpsertSubmissionFromQuizAttempt:
// - Ambil assessment_id dari quiz
// - Upsert ke submission attempt TERAKHIR untuk student×assessment×school
// - Pakai last_percent sebagai SubmissionScore
// Catatan:
// - Untuk quiz, biasanya kita UPDATE attempt terakhir (bukan create attempt baru tiap progress)
// - attempt baru dibuat oleh flow "start attempt" / "submit attempt" kalau kamu mau strict
func (s *SubmissionService) UpsertSubmissionFromQuizAttempt(
	ctx context.Context,
	attempt *qmodel.StudentQuizAttemptModel,
) error {
	if attempt == nil {
		log.Printf("[SubmissionService] attempt is nil, skipping")
		return nil
	}

	log.Printf(
		"[SubmissionService] UpsertSubmissionFromQuizAttempt called. school_id=%s quiz_id=%s student_id=%s last_percent=%v",
		attempt.StudentQuizAttemptSchoolID,
		attempt.StudentQuizAttemptQuizID,
		attempt.StudentQuizAttemptStudentID,
		attempt.StudentQuizAttemptLastPercent,
	)

	// 1) Ambil assessment_id dari quiz (scan string -> parse UUID)
	var assessmentIDStr string
	if err := s.DB.WithContext(ctx).Raw(`
		SELECT quiz_assessment_id::text
		FROM quizzes
		WHERE quiz_id = ? AND quiz_school_id = ? AND quiz_deleted_at IS NULL
	`, attempt.StudentQuizAttemptQuizID, attempt.StudentQuizAttemptSchoolID).
		Scan(&assessmentIDStr).Error; err != nil {

		log.Printf("[SubmissionService] ERROR get assessment_id from quiz: %v", err)
		return err
	}

	if strings.TrimSpace(assessmentIDStr) == "" {
		log.Printf(
			"[SubmissionService] quiz_id=%s tidak terhubung ke assessment (NULL/empty). Skip submission.",
			attempt.StudentQuizAttemptQuizID,
		)
		return nil
	}

	assessmentID, err := uuid.Parse(strings.TrimSpace(assessmentIDStr))
	if err != nil {
		log.Printf("[SubmissionService] ERROR parse assessment_id (%s): %v", assessmentIDStr, err)
		return err
	}

	log.Printf("[SubmissionService] Mapped quiz_id=%s -> assessment_id=%s",
		attempt.StudentQuizAttemptQuizID, assessmentID)

	// 2) Ambil submission attempt TERAKHIR (by attempt_count desc)
	var sub smodel.SubmissionModel
	err = s.DB.WithContext(ctx).
		Where(`
			submission_school_id = ?
			AND submission_assessment_id = ?
			AND submission_student_id = ?
			AND submission_deleted_at IS NULL
		`,
			attempt.StudentQuizAttemptSchoolID,
			assessmentID,
			attempt.StudentQuizAttemptStudentID,
		).
		Order("submission_attempt_count DESC").
		First(&sub).Error

	now := time.Now().UTC()

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 3a) Belum ada → create attempt_count=1
			log.Printf("[SubmissionService] No existing submission. Creating attempt_count=1 ...")

			sub = smodel.SubmissionModel{
				SubmissionSchoolID:     attempt.StudentQuizAttemptSchoolID,
				SubmissionAssessmentID: assessmentID,
				SubmissionStudentID:    attempt.StudentQuizAttemptStudentID,

				SubmissionAttemptCount: 1,

				SubmissionStatus:       smodel.SubmissionStatusSubmitted,
				SubmissionSubmittedAt:  &now,
				SubmissionIsLate:       false,
				SubmissionQuizFinished: 1,
			}

			if attempt.StudentQuizAttemptLastPercent != nil {
				sub.SubmissionScore = attempt.StudentQuizAttemptLastPercent
			}

			if err := s.DB.WithContext(ctx).Create(&sub).Error; err != nil {
				log.Printf("[SubmissionService] ERROR create submission: %v", err)
				return err
			}

			log.Printf("[SubmissionService] Submission created. submission_id=%s attempt_count=%d score=%v",
				sub.SubmissionID, sub.SubmissionAttemptCount, sub.SubmissionScore)

			return nil
		}

		log.Printf("[SubmissionService] ERROR find latest submission: %v", err)
		return err
	}

	// 3b) Sudah ada submission latest → update row itu
	log.Printf("[SubmissionService] Latest submission found. submission_id=%s attempt_count=%d old_score=%v old_status=%s",
		sub.SubmissionID, sub.SubmissionAttemptCount, sub.SubmissionScore, sub.SubmissionStatus)

	sub.SubmissionStatus = smodel.SubmissionStatusSubmitted
	sub.SubmissionSubmittedAt = &now

	if attempt.StudentQuizAttemptLastPercent != nil {
		sub.SubmissionScore = attempt.StudentQuizAttemptLastPercent
	}

	// Sementara: anggap 1 quiz -> finished=1
	sub.SubmissionQuizFinished = 1

	// IsLate tetap false (atau hitung dari due_at kalau mau)
	sub.SubmissionIsLate = false

	if err := s.DB.WithContext(ctx).Save(&sub).Error; err != nil {
		log.Printf("[SubmissionService] ERROR update submission: %v", err)
		return err
	}

	log.Printf("[SubmissionService] Submission updated. submission_id=%s attempt_count=%d new_score=%v new_status=%s",
		sub.SubmissionID, sub.SubmissionAttemptCount, sub.SubmissionScore, sub.SubmissionStatus)

	return nil
}
