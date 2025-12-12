// file: internals/features/school/submissions_assesments/quizzes/service/student_quiz_attempt_service.go
package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	qmodel "madinahsalam_backend/internals/features/school/submissions_assesments/quizzes/model"
	subsvc "madinahsalam_backend/internals/features/school/submissions_assesments/submissions/service"
)

/* =========================================================
   SERVICE
========================================================= */

type StudentQuizAttemptService struct {
	DB *gorm.DB
}

func NewStudentQuizAttemptService(db *gorm.DB) *StudentQuizAttemptService {
	return &StudentQuizAttemptService{DB: db}
}

/* =========================================================
   INPUT SUBMIT ATTEMPT
========================================================= */

// SubmitQuizAttemptInput merepresentasikan payload yang sudah
// "dibersihkan" di controller (student_id, quiz_id, attempt_id).
type SubmitQuizAttemptInput struct {
	// Attempt summary yang mau ditambah history-nya
	AttemptID uuid.UUID

	// Optional: override waktu selesai (kalau FE kirim)
	FinishedAt *time.Time

	// Map jawaban murid:
	// key   = quiz_question_id
	// value = jawaban murid:
	//        - SINGLE: key option ("A","B","C",dst)
	//        - ESSAY : text bebas
	Answers map[uuid.UUID]string
}

/* =========================================================
   PUBLIC API: SubmitAttempt
========================================================= */

// SubmitAttempt:
// - load summary attempt (1 row student×quiz)
// - load questions
// - hitung skor + build history item (dengan quiz_id, question_id, version)
// - AppendAttemptHistory (nambah elemen ke JSON history + update count/best/last/first/avg)
// - update status + finished_at (status = finished utk attempt terakhir)
// - sinkron ke tabel submissions (1 assessment × 1 student)
func (s *StudentQuizAttemptService) SubmitAttempt(
	ctx context.Context,
	in *SubmitQuizAttemptInput,
) (*qmodel.StudentQuizAttemptModel, error) {
	if in == nil {
		return nil, errors.New("input cannot be nil")
	}
	if in.AttemptID == uuid.Nil {
		return nil, errors.New("attempt_id kosong")
	}

	log.Printf(
		"[StudentQuizAttemptService] SubmitAttempt called. attempt_id=%s finished_at=%v answers_count=%d",
		in.AttemptID,
		in.FinishedAt,
		len(in.Answers),
	)

	// 1) Load summary attempt
	var attempt qmodel.StudentQuizAttemptModel
	if err := s.DB.WithContext(ctx).
		First(&attempt, "student_quiz_attempt_id = ?", in.AttemptID).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("[StudentQuizAttemptService] ERROR: attempt tidak ditemukan. attempt_id=%s", in.AttemptID)
			return nil, errors.New("attempt tidak ditemukan")
		}
		log.Printf("[StudentQuizAttemptService] ERROR load attempt: %v", err)
		return nil, err
	}

	log.Printf(
		"[StudentQuizAttemptService] Loaded attempt. quiz_id=%s school_id=%s student_id=%s status=%s count=%d",
		attempt.StudentQuizAttemptQuizID,
		attempt.StudentQuizAttemptSchoolID,
		attempt.StudentQuizAttemptStudentID,
		attempt.StudentQuizAttemptStatus,
		attempt.StudentQuizAttemptCount,
	)

	// 2) Load semua soal di quiz ini (tenant-safe)
	var questions []qmodel.QuizQuestionModel
	if err := s.DB.WithContext(ctx).
		Where("quiz_question_quiz_id = ? AND quiz_question_school_id = ?", attempt.StudentQuizAttemptQuizID, attempt.StudentQuizAttemptSchoolID).
		Where("quiz_question_deleted_at IS NULL").
		Find(&questions).Error; err != nil {

		log.Printf("[StudentQuizAttemptService] ERROR load questions: %v", err)
		return nil, err
	}

	log.Printf(
		"[StudentQuizAttemptService] Loaded %d questions for quiz_id=%s",
		len(questions),
		attempt.StudentQuizAttemptQuizID,
	)

	// 3) Build items (1 per question)
	items := make([]qmodel.StudentQuizAttemptQuestionItem, 0, len(questions))

	for _, q := range questions {
		rawAns, ok := in.Answers[q.QuizQuestionID]
		ans := strings.TrimSpace(rawAns)

		item := qmodel.StudentQuizAttemptQuestionItem{
			QuizID:              q.QuizQuestionQuizID,
			QuizQuestionID:      q.QuizQuestionID,
			QuizQuestionVersion: q.QuizQuestionVersion,
			QuizQuestionType:    q.QuizQuestionType,
			Points:              q.QuizQuestionPoints,
			PointsEarned:        0, // default 0
		}

		switch q.QuizQuestionType {
		case qmodel.QuizQuestionTypeSingle:
			if ok && ans != "" {
				item.AnswerSingle = &ans
			}

			if q.QuizQuestionCorrect != nil && ans != "" {
				correctKey := strings.TrimSpace(*q.QuizQuestionCorrect)
				isCorrect := strings.EqualFold(ans, correctKey)
				item.IsCorrect = &isCorrect
				if isCorrect {
					item.PointsEarned = q.QuizQuestionPoints
				}
			}

		case qmodel.QuizQuestionTypeEssay:
			if ok && ans != "" {
				item.AnswerEssay = &ans
			}
			// Essay default: belum dinilai
			// IsCorrect & PointsEarned bisa diupdate di endpoint lain (grading)
		}

		items = append(items, item)

		// 🔍 Log per soal (ringkas)
		log.Printf(
			"[StudentQuizAttemptService] Q item built. question_id=%s type=%s answered=%v points=%.2f earned=%.2f",
			q.QuizQuestionID,
			q.QuizQuestionType,
			ans != "",
			q.QuizQuestionPoints,
			item.PointsEarned,
		)
	}

	// 4) Tentukan startedAt & finishedAt untuk attempt kali ini
	now := time.Now().UTC()

	startedAt := now
	if attempt.StudentQuizAttemptStartedAt != nil {
		startedAt = *attempt.StudentQuizAttemptStartedAt
	}

	finishedAt := now
	if in.FinishedAt != nil {
		finishedAt = in.FinishedAt.UTC()
	}

	log.Printf(
		"[StudentQuizAttemptService] Attempt times. started_at=%v finished_at=%v",
		startedAt, finishedAt,
	)

	// 5) Append ke history (auto hitung skor total + percent + best/last/first/avg)
	if err := attempt.AppendAttemptHistory(startedAt, finishedAt, items); err != nil {
		log.Printf("[StudentQuizAttemptService] ERROR AppendAttemptHistory: %v", err)
		return nil, err
	}

	// Ambil nilai readable buat log
	var lastPercentStr, bestPercentStr, firstPercentStr, avgPercentStr string

	if attempt.StudentQuizAttemptLastPercent != nil {
		lastPercentStr = fmt.Sprintf("%.3f", *attempt.StudentQuizAttemptLastPercent)
	} else {
		lastPercentStr = "nil"
	}

	if attempt.StudentQuizAttemptBestPercent != nil {
		bestPercentStr = fmt.Sprintf("%.3f", *attempt.StudentQuizAttemptBestPercent)
	} else {
		bestPercentStr = "nil"
	}

	if attempt.StudentQuizAttemptFirstPercent != nil {
		firstPercentStr = fmt.Sprintf("%.3f", *attempt.StudentQuizAttemptFirstPercent)
	} else {
		firstPercentStr = "nil"
	}

	if attempt.StudentQuizAttemptAvgPercent != nil {
		avgPercentStr = fmt.Sprintf("%.3f", *attempt.StudentQuizAttemptAvgPercent)
	} else {
		avgPercentStr = "nil"
	}

	log.Printf(
		"[StudentQuizAttemptService] History appended. total_history=%d last_raw=%v last_percent=%s best_percent=%s first_percent=%s avg_percent=%s",
		attempt.StudentQuizAttemptCount,
		func() interface{} {
			if attempt.StudentQuizAttemptLastRaw == nil {
				return "nil"
			}
			return *attempt.StudentQuizAttemptLastRaw
		}(),
		lastPercentStr,
		bestPercentStr,
		firstPercentStr,
		avgPercentStr,
	)

	// 6) Update status & timestamps global (status = selesai)
	attempt.StudentQuizAttemptStatus = qmodel.StudentQuizAttemptFinished
	attempt.StudentQuizAttemptFinishedAt = &finishedAt

	// 7) Persist
	if err := s.DB.WithContext(ctx).Save(&attempt).Error; err != nil {
		log.Printf("[StudentQuizAttemptService] ERROR Save attempt: %v", err)
		return nil, err
	}

	log.Printf(
		"[StudentQuizAttemptService] Attempt saved. attempt_id=%s status=%s count=%d",
		attempt.StudentQuizAttemptID,
		attempt.StudentQuizAttemptStatus,
		attempt.StudentQuizAttemptCount,
	)

	// 8) 🔗 SINKRON KE SUBMISSIONS (1 assessment × 1 student)
	subService := subsvc.NewSubmissionService(s.DB)
	if err := subService.UpsertSubmissionFromQuizAttempt(ctx, &attempt); err != nil {
		log.Printf("[StudentQuizAttemptService] UpsertSubmissionFromQuizAttempt error: %v", err)
		// kalau mau keras, bisa return err; sekarang cuma log biar quiz tetap "berhasil"
	}

	return &attempt, nil
}
