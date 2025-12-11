// file: internals/features/school/submissions_assesments/quizzes/dto/student_quiz_attempt_dto.go
package dto

import (
	"encoding/json"
	"time"

	qmodel "madinahsalam_backend/internals/features/school/submissions_assesments/quizzes/model"
	"madinahsalam_backend/internals/helpers/dbtime"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

/* ==========================================================================================
   Type alias untuk history (optional helper)
   Biar kalau butuh decode history di layer lain, bisa pakai tipe ini.
========================================================================================== */

type StudentQuizAttemptQuestionItem = qmodel.StudentQuizAttemptQuestionItem
type StudentQuizAttemptHistoryItem = qmodel.StudentQuizAttemptHistoryItem

/* ==========================================================================================
   ITEM untuk create (jawaban per soal) — dipakai di CreateStudentQuizAttemptRequest
========================================================================================== */

type CreateStudentQuizAttemptItem struct {
	QuizQuestionID uuid.UUID `json:"quiz_question_id" validate:"required,uuid"`
	// SINGLE → isi answer_single
	// ESSAY  → isi answer_essay
	AnswerSingle *string `json:"answer_single,omitempty"`
	AnswerEssay  *string `json:"answer_essay,omitempty"`
}

/* ==========================================================================================
   REQUEST — CREATE
   Bisa dua mode:
   - Mode A: hanya "mulai attempt" (items dikosongkan)
   - Mode B: create + submit sekaligus (items diisi → langsung dinilai)
   Server boleh meng-derive school_id / student_id dari context.
========================================================================================== */

type CreateStudentQuizAttemptRequest struct {
	// Opsional: bisa diisi server dari context
	StudentQuizAttemptSchoolID *uuid.UUID `json:"student_quiz_attempt_school_id" validate:"omitempty,uuid"`

	// Wajib: quiz yang dimaksud
	StudentQuizAttemptQuizID uuid.UUID `json:"student_quiz_attempt_quiz_id" validate:"required,uuid"`

	// Opsional (untuk admin/dkm/teacher); untuk self-attempt bisa diisi server
	StudentQuizAttemptStudentID *uuid.UUID `json:"student_quiz_attempt_student_id" validate:"omitempty,uuid"`

	// Opsional: kalau mau override status awal
	StudentQuizAttemptStatus *qmodel.StudentQuizAttemptStatus `json:"student_quiz_attempt_status" validate:"omitempty,oneof=in_progress submitted finished abandoned"`

	// Waktu attempt (optional; kalau kosong → pakai waktu server di controller)
	AttemptStartedAt  *time.Time `json:"attempt_started_at,omitempty" validate:"omitempty"`
	AttemptFinishedAt *time.Time `json:"attempt_finished_at,omitempty" validate:"omitempty"`

	// Kalau diisi → berarti langsung sekalian submit jawaban (1 request)
	// Kalau dikosongkan → berarti hanya "mulai attempt" (tanpa jawaban dulu)
	Items []CreateStudentQuizAttemptItem `json:"items" validate:"omitempty,dive"`
}

// ToModel — bikin model summary-nya (tanpa history).
// History, count, best_*, last_* akan di-handle di controller/service.
func (r *CreateStudentQuizAttemptRequest) ToModel() *qmodel.StudentQuizAttemptModel {
	m := &qmodel.StudentQuizAttemptModel{
		StudentQuizAttemptQuizID: r.StudentQuizAttemptQuizID,
		// History default: [] (akan diisi di controller kalau kosong)
		// Count default: 0
		// Status default di DB: in_progress (kalau tidak di-set di sini)
	}

	if r.StudentQuizAttemptSchoolID != nil {
		m.StudentQuizAttemptSchoolID = *r.StudentQuizAttemptSchoolID
	}
	if r.StudentQuizAttemptStudentID != nil {
		m.StudentQuizAttemptStudentID = *r.StudentQuizAttemptStudentID
	}
	if r.StudentQuizAttemptStatus != nil {
		m.StudentQuizAttemptStatus = *r.StudentQuizAttemptStatus
	}

	// StartedAt & FinishedAt: dikelola di controller saat bikin history
	return m
}

/* ==========================================================================================
   REQUEST — UPDATE/PATCH (PARTIAL)
   Biasanya dipakai internal (service/admin) untuk update summary:
   - status, started/finished
   - history JSON
   - count
   - best_*
   - last_*
========================================================================================== */

type UpdateStudentQuizAttemptRequest struct {
	StudentQuizAttemptSchoolID  *uuid.UUID `json:"student_quiz_attempt_school_id" validate:"omitempty,uuid"`
	StudentQuizAttemptQuizID    *uuid.UUID `json:"student_quiz_attempt_quiz_id" validate:"omitempty,uuid"`
	StudentQuizAttemptStudentID *uuid.UUID `json:"student_quiz_attempt_student_id" validate:"omitempty,uuid"`

	// Status & waktu global attempt
	StudentQuizAttemptStatus     *qmodel.StudentQuizAttemptStatus `json:"student_quiz_attempt_status" validate:"omitempty,oneof=in_progress submitted finished abandoned"`
	StudentQuizAttemptStartedAt  *time.Time                       `json:"student_quiz_attempt_started_at" validate:"omitempty"`
	StudentQuizAttemptFinishedAt *time.Time                       `json:"student_quiz_attempt_finished_at" validate:"omitempty"`

	// Full history (opsional, biasanya diisi backend)
	// Struktur JSON-nya sekarang:
	// [
	//   {
	//     "attempt_no": ...,
	//     "attempt_started_at": "...",
	//     "attempt_finished_at": "...",
	//     "attempt_raw_score": ...,
	//     "attempt_percent": ...,
	//     "items": [
	//       {
	//         "quiz_id": "...",
	//         "quiz_question_id": "...",
	//         "quiz_question_version": 1,
	//         "quiz_question_type": "single",
	//         "answer_single": "C",
	//         "answer_essay": null,
	//         "is_correct": true,
	//         "points": 1,
	//         "points_earned": 1
	//       }
	//     ]
	//   }
	// ]
	StudentQuizAttemptHistory *json.RawMessage `json:"student_quiz_attempt_history" validate:"omitempty"`

	// Summary: total attempt
	StudentQuizAttemptCount *int `json:"student_quiz_attempt_count" validate:"omitempty,gte=0"`

	// Summary: nilai terbaik
	StudentQuizAttemptBestRaw        *float64   `json:"student_quiz_attempt_best_raw" validate:"omitempty"`
	StudentQuizAttemptBestPercent    *float64   `json:"student_quiz_attempt_best_percent" validate:"omitempty"`
	StudentQuizAttemptBestStartedAt  *time.Time `json:"student_quiz_attempt_best_started_at" validate:"omitempty"`
	StudentQuizAttemptBestFinishedAt *time.Time `json:"student_quiz_attempt_best_finished_at" validate:"omitempty"`

	// Summary: nilai terakhir
	StudentQuizAttemptLastRaw        *float64   `json:"student_quiz_attempt_last_raw" validate:"omitempty"`
	StudentQuizAttemptLastPercent    *float64   `json:"student_quiz_attempt_last_percent" validate:"omitempty"`
	StudentQuizAttemptLastStartedAt  *time.Time `json:"student_quiz_attempt_last_started_at" validate:"omitempty"`
	StudentQuizAttemptLastFinishedAt *time.Time `json:"student_quiz_attempt_last_finished_at" validate:"omitempty"`
}

// ApplyToModel — patch ke model yang sudah di-load.
// Business logic (recompute best/last dari history) sebaiknya di service.
func (r *UpdateStudentQuizAttemptRequest) ApplyToModel(m *qmodel.StudentQuizAttemptModel) error {
	if r.StudentQuizAttemptSchoolID != nil {
		m.StudentQuizAttemptSchoolID = *r.StudentQuizAttemptSchoolID
	}
	if r.StudentQuizAttemptQuizID != nil {
		m.StudentQuizAttemptQuizID = *r.StudentQuizAttemptQuizID
	}
	if r.StudentQuizAttemptStudentID != nil {
		m.StudentQuizAttemptStudentID = *r.StudentQuizAttemptStudentID
	}

	// Status & waktu global
	if r.StudentQuizAttemptStatus != nil {
		m.StudentQuizAttemptStatus = *r.StudentQuizAttemptStatus
	}
	if r.StudentQuizAttemptStartedAt != nil {
		m.StudentQuizAttemptStartedAt = r.StudentQuizAttemptStartedAt
	}
	if r.StudentQuizAttemptFinishedAt != nil {
		m.StudentQuizAttemptFinishedAt = r.StudentQuizAttemptFinishedAt
	}

	// History
	if r.StudentQuizAttemptHistory != nil {
		m.StudentQuizAttemptHistory = JSONFromRaw(*r.StudentQuizAttemptHistory)
	}

	// Count
	if r.StudentQuizAttemptCount != nil {
		m.StudentQuizAttemptCount = *r.StudentQuizAttemptCount
	}

	// Best summary
	if r.StudentQuizAttemptBestRaw != nil {
		m.StudentQuizAttemptBestRaw = r.StudentQuizAttemptBestRaw
	}
	if r.StudentQuizAttemptBestPercent != nil {
		m.StudentQuizAttemptBestPercent = r.StudentQuizAttemptBestPercent
	}
	if r.StudentQuizAttemptBestStartedAt != nil {
		m.StudentQuizAttemptBestStartedAt = r.StudentQuizAttemptBestStartedAt
	}
	if r.StudentQuizAttemptBestFinishedAt != nil {
		m.StudentQuizAttemptBestFinishedAt = r.StudentQuizAttemptBestFinishedAt
	}

	// Last summary
	if r.StudentQuizAttemptLastRaw != nil {
		m.StudentQuizAttemptLastRaw = r.StudentQuizAttemptLastRaw
	}
	if r.StudentQuizAttemptLastPercent != nil {
		m.StudentQuizAttemptLastPercent = r.StudentQuizAttemptLastPercent
	}
	if r.StudentQuizAttemptLastStartedAt != nil {
		m.StudentQuizAttemptLastStartedAt = r.StudentQuizAttemptLastStartedAt
	}
	if r.StudentQuizAttemptLastFinishedAt != nil {
		m.StudentQuizAttemptLastFinishedAt = r.StudentQuizAttemptLastFinishedAt
	}

	return nil
}

/* ==========================================================================================
   RESPONSE DTO
   Ini yang dikirim ke FE, sudah sesuai dengan model JSON summary.
   Field history tetap json.RawMessage supaya fleksibel.
========================================================================================== */

type StudentQuizAttemptResponse struct {
	StudentQuizAttemptID        uuid.UUID `json:"student_quiz_attempt_id"`
	StudentQuizAttemptSchoolID  uuid.UUID `json:"student_quiz_attempt_school_id"`
	StudentQuizAttemptQuizID    uuid.UUID `json:"student_quiz_attempt_quiz_id"`
	StudentQuizAttemptStudentID uuid.UUID `json:"student_quiz_attempt_student_id"`

	// Snapshot/cache user profile & siswa
	StudentQuizAttemptUserProfileNameSnapshot        *string `json:"student_quiz_attempt_user_profile_name_snapshot,omitempty"`
	StudentQuizAttemptUserProfileAvatarURLSnapshot   *string `json:"student_quiz_attempt_user_profile_avatar_url_snapshot,omitempty"`
	StudentQuizAttemptUserProfileWhatsappURLSnapshot *string `json:"student_quiz_attempt_user_profile_whatsapp_url_snapshot,omitempty"`
	StudentQuizAttemptUserProfileGenderSnapshot      *string `json:"student_quiz_attempt_user_profile_gender_snapshot,omitempty"`
	StudentQuizAttemptSchoolStudentCodeCache         *string `json:"student_quiz_attempt_school_student_code_cache,omitempty"`

	// Status & waktu global attempt
	StudentQuizAttemptStatus     qmodel.StudentQuizAttemptStatus `json:"student_quiz_attempt_status"`
	StudentQuizAttemptStartedAt  *time.Time                      `json:"student_quiz_attempt_started_at,omitempty"`
	StudentQuizAttemptFinishedAt *time.Time                      `json:"student_quiz_attempt_finished_at,omitempty"`

	// History full (biar FE bisa tampilkan riwayat attempt + jawaban)
	StudentQuizAttemptHistory json.RawMessage `json:"student_quiz_attempt_history"`

	// Summary
	StudentQuizAttemptCount int `json:"student_quiz_attempt_count"`

	StudentQuizAttemptBestRaw        *float64   `json:"student_quiz_attempt_best_raw,omitempty"`
	StudentQuizAttemptBestPercent    *float64   `json:"student_quiz_attempt_best_percent,omitempty"`
	StudentQuizAttemptBestStartedAt  *time.Time `json:"student_quiz_attempt_best_started_at,omitempty"`
	StudentQuizAttemptBestFinishedAt *time.Time `json:"student_quiz_attempt_best_finished_at,omitempty"`

	StudentQuizAttemptLastRaw        *float64   `json:"student_quiz_attempt_last_raw,omitempty"`
	StudentQuizAttemptLastPercent    *float64   `json:"student_quiz_attempt_last_percent,omitempty"`
	StudentQuizAttemptLastStartedAt  *time.Time `json:"student_quiz_attempt_last_started_at,omitempty"`
	StudentQuizAttemptLastFinishedAt *time.Time `json:"student_quiz_attempt_last_finished_at,omitempty"`

	StudentQuizAttemptCreatedAt time.Time `json:"student_quiz_attempt_created_at"`
	StudentQuizAttemptUpdatedAt time.Time `json:"student_quiz_attempt_updated_at"`
}

func FromModelStudentQuizAttempt(m *qmodel.StudentQuizAttemptModel) *StudentQuizAttemptResponse {
	return &StudentQuizAttemptResponse{
		StudentQuizAttemptID:        m.StudentQuizAttemptID,
		StudentQuizAttemptSchoolID:  m.StudentQuizAttemptSchoolID,
		StudentQuizAttemptQuizID:    m.StudentQuizAttemptQuizID,
		StudentQuizAttemptStudentID: m.StudentQuizAttemptStudentID,

		StudentQuizAttemptUserProfileNameSnapshot:        m.StudentQuizAttemptUserProfileNameSnapshot,
		StudentQuizAttemptUserProfileAvatarURLSnapshot:   m.StudentQuizAttemptUserProfileAvatarURLSnapshot,
		StudentQuizAttemptUserProfileWhatsappURLSnapshot: m.StudentQuizAttemptUserProfileWhatsappURLSnapshot,
		StudentQuizAttemptUserProfileGenderSnapshot:      m.StudentQuizAttemptUserProfileGenderSnapshot,
		StudentQuizAttemptSchoolStudentCodeCache:         m.StudentQuizAttemptSchoolStudentCodeCache,

		StudentQuizAttemptStatus:     m.StudentQuizAttemptStatus,
		StudentQuizAttemptStartedAt:  m.StudentQuizAttemptStartedAt,
		StudentQuizAttemptFinishedAt: m.StudentQuizAttemptFinishedAt,

		StudentQuizAttemptHistory: json.RawMessage(m.StudentQuizAttemptHistory),

		StudentQuizAttemptCount: m.StudentQuizAttemptCount,

		StudentQuizAttemptBestRaw:        m.StudentQuizAttemptBestRaw,
		StudentQuizAttemptBestPercent:    m.StudentQuizAttemptBestPercent,
		StudentQuizAttemptBestStartedAt:  m.StudentQuizAttemptBestStartedAt,
		StudentQuizAttemptBestFinishedAt: m.StudentQuizAttemptBestFinishedAt,

		StudentQuizAttemptLastRaw:        m.StudentQuizAttemptLastRaw,
		StudentQuizAttemptLastPercent:    m.StudentQuizAttemptLastPercent,
		StudentQuizAttemptLastStartedAt:  m.StudentQuizAttemptLastStartedAt,
		StudentQuizAttemptLastFinishedAt: m.StudentQuizAttemptLastFinishedAt,

		StudentQuizAttemptCreatedAt: m.StudentQuizAttemptCreatedAt,
		StudentQuizAttemptUpdatedAt: m.StudentQuizAttemptUpdatedAt,
	}
}

func FromModelsStudentQuizAttempts(items []qmodel.StudentQuizAttemptModel) []*StudentQuizAttemptResponse {
	out := make([]*StudentQuizAttemptResponse, 0, len(items))
	for i := range items {
		item := items[i]
		out = append(out, FromModelStudentQuizAttempt(&item))
	}
	return out
}

/* ==========================================================================================
   Helper kecil
========================================================================================== */

func JSONFromRaw(raw json.RawMessage) datatypes.JSON {
	return datatypes.JSON(raw)
}

// Versi timezone-aware: convert semua time field ke timezone sekolah
func FromModelStudentQuizAttemptWithCtx(c *fiber.Ctx, m *qmodel.StudentQuizAttemptModel) *StudentQuizAttemptResponse {
	resp := FromModelStudentQuizAttempt(m)

	// Global started/finished
	resp.StudentQuizAttemptStartedAt = dbtime.ToSchoolTimePtr(c, m.StudentQuizAttemptStartedAt)
	resp.StudentQuizAttemptFinishedAt = dbtime.ToSchoolTimePtr(c, m.StudentQuizAttemptFinishedAt)

	// Best
	resp.StudentQuizAttemptBestStartedAt = dbtime.ToSchoolTimePtr(c, m.StudentQuizAttemptBestStartedAt)
	resp.StudentQuizAttemptBestFinishedAt = dbtime.ToSchoolTimePtr(c, m.StudentQuizAttemptBestFinishedAt)

	// Last
	resp.StudentQuizAttemptLastStartedAt = dbtime.ToSchoolTimePtr(c, m.StudentQuizAttemptLastStartedAt)
	resp.StudentQuizAttemptLastFinishedAt = dbtime.ToSchoolTimePtr(c, m.StudentQuizAttemptLastFinishedAt)

	// Created / Updated
	resp.StudentQuizAttemptCreatedAt = dbtime.ToSchoolTime(c, m.StudentQuizAttemptCreatedAt)
	resp.StudentQuizAttemptUpdatedAt = dbtime.ToSchoolTime(c, m.StudentQuizAttemptUpdatedAt)

	return resp
}

func FromModelsStudentQuizAttemptsWithCtx(c *fiber.Ctx, items []qmodel.StudentQuizAttemptModel) []*StudentQuizAttemptResponse {
	out := make([]*StudentQuizAttemptResponse, 0, len(items))
	for i := range items {
		item := items[i]
		out = append(out, FromModelStudentQuizAttemptWithCtx(c, &item))
	}
	return out
}
