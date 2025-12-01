package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	qmodel "madinahsalam_backend/internals/features/school/submissions_assesments/quizzes/model"
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
   (dipanggil dari controller submit/create)
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
// - AppendAttemptHistory (nambah elemen ke JSON history + update count/best/last)
// - update status + finished_at (status = finished utk attempt terakhir)
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

	// 1) Load summary attempt
	var attempt qmodel.StudentQuizAttemptModel
	if err := s.DB.WithContext(ctx).
		First(&attempt, "student_quiz_attempt_id = ?", in.AttemptID).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("attempt tidak ditemukan")
		}
		return nil, err
	}

	// ⚠️ PENTING:
	// Jangan blokir kalau status sudah "finished".
	// Karena summary row ini mewakili 1 student×quiz,
	// dan history di dalam JSON boleh berisi banyak attempt.
	// Jadi setiap SubmitAttempt = nambah 1 history item baru.
	//
	// Kalau kamu mau batasi jumlah attempt, logiknya taruh di sini
	// (misal: kalau attempt.StudentQuizAttemptCount >= maxAttempt → error)
	// tapi BUKAN berdasarkan status finished.
	// ----------------------------------------------------------------
	// if attempt.StudentQuizAttemptStatus == qmodel.StudentQuizAttemptFinished {
	// 	   return nil, errors.New("attempt sudah finished")
	// }

	// 2) Load semua soal di quiz ini (tenant-safe)
	var questions []qmodel.QuizQuestionModel
	if err := s.DB.WithContext(ctx).
		Where("quiz_question_quiz_id = ? AND quiz_question_school_id = ?", attempt.StudentQuizAttemptQuizID, attempt.StudentQuizAttemptSchoolID).
		Where("quiz_question_deleted_at IS NULL").
		Find(&questions).Error; err != nil {
		return nil, err
	}

	// 3) Build items (1 per question)
	items := make([]qmodel.StudentQuizAttemptQuestionItem, 0, len(questions))

	for _, q := range questions {
		item := qmodel.StudentQuizAttemptQuestionItem{
			QuizID:              q.QuizQuestionQuizID,
			QuizQuestionID:      q.QuizQuestionID,
			QuizQuestionVersion: q.QuizQuestionVersion,
			QuizQuestionType:    q.QuizQuestionType,
			Points:              q.QuizQuestionPoints,
			PointsEarned:        0, // default 0
		}

		rawAns, ok := in.Answers[q.QuizQuestionID]
		ans := strings.TrimSpace(rawAns)

		switch q.QuizQuestionType {
		case qmodel.QuizQuestionTypeSingle:
			if ok && ans != "" {
				item.AnswerSingle = &ans
			}

			// Koreksi otomatis single choice
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
	}

	// 4) Tentukan startedAt & finishedAt untuk attempt kali ini
	now := time.Now().UTC()

	startedAt := now
	// Kalau summary sudah punya started_at (mis: waktu pertama kali mulai quiz),
	// kita boleh pakai ini sebagai referensi; tapi untuk per-attempt,
	// waktunya disimpan di history item.
	if attempt.StudentQuizAttemptStartedAt != nil {
		startedAt = *attempt.StudentQuizAttemptStartedAt
	}

	finishedAt := now
	if in.FinishedAt != nil {
		finishedAt = in.FinishedAt.UTC()
	}

	// 5) Append ke history (auto hitung skor total + percent + best/last)
	if err := attempt.AppendAttemptHistory(startedAt, finishedAt, items); err != nil {
		return nil, err
	}

	// 6) Update status & timestamps global (status = selesai)
	attempt.StudentQuizAttemptStatus = qmodel.StudentQuizAttemptFinished
	attempt.StudentQuizAttemptFinishedAt = &finishedAt

	// 7) Persist
	if err := s.DB.WithContext(ctx).Save(&attempt).Error; err != nil {
		return nil, err
	}

	return &attempt, nil
}
