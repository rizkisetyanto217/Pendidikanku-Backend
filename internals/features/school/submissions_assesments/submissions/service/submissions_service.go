// file: internals/features/school/submissions_assesments/submissions/service/submission_service.go
package service

import (
	"context"
	"log"
	"strings"
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

	// 1) Ambil assessment_id dari quiz (scan ke string dulu baru parse UUID)
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
			"[SubmissionService] quiz_id=%s tidak terhubung ke assessment (hasil NULL/empty). Skip submission.",
			attempt.StudentQuizAttemptQuizID,
		)
		return nil
	}

	assessmentID, err := uuid.Parse(strings.TrimSpace(assessmentIDStr))
	if err != nil {
		log.Printf("[SubmissionService] ERROR parse assessment_id (%s): %v", assessmentIDStr, err)
		return err
	}

	log.Printf(
		"[SubmissionService] Mapped quiz_id=%s -> assessment_id=%s",
		attempt.StudentQuizAttemptQuizID,
		assessmentID,
	)

	// 2) Cari existing submission (1 row per student×assessment×school)
	var sub smodel.SubmissionModel
	err = s.DB.WithContext(ctx).
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
			log.Printf("[SubmissionService] No existing submission. Creating new row...")

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

			// Pakai last_percent sebagai skor kalau ada
			if attempt.StudentQuizAttemptLastPercent != nil {
				sub.SubmissionScore = attempt.StudentQuizAttemptLastPercent
			}

			// Satu quiz dianggap finished
			sub.SubmissionQuizFinished = 1

			if err := s.DB.WithContext(ctx).Create(&sub).Error; err != nil {
				log.Printf("[SubmissionService] ERROR create submission: %v", err)
				return err
			}

			log.Printf(
				"[SubmissionService] Submission created. submission_id=%s score=%v",
				sub.SubmissionID,
				sub.SubmissionScore,
			)
			return nil
		}

		log.Printf("[SubmissionService] ERROR find existing submission: %v", err)
		return err
	}

	// 3b) Sudah ada submission → update
	log.Printf(
		"[SubmissionService] Existing submission found. submission_id=%s old_score=%v old_status=%s",
		sub.SubmissionID,
		sub.SubmissionScore,
		sub.SubmissionStatus,
	)

	status := smodel.SubmissionStatusSubmitted
	sub.SubmissionStatus = status
	sub.SubmissionSubmittedAt = &now

	if attempt.StudentQuizAttemptLastPercent != nil {
		sub.SubmissionScore = attempt.StudentQuizAttemptLastPercent
	}

	// Sementara: anggap 1 quiz, jadi selalu 1
	sub.SubmissionQuizFinished = 1

	if err := s.DB.WithContext(ctx).Save(&sub).Error; err != nil {
		log.Printf("[SubmissionService] ERROR update submission: %v", err)
		return err
	}

	log.Printf(
		"[SubmissionService] Submission updated. submission_id=%s new_score=%v new_status=%s",
		sub.SubmissionID,
		sub.SubmissionScore,
		sub.SubmissionStatus,
	)

	return nil
}
