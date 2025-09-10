// file: internals/features/school/quizzes/dto/quiz_item_dto.go
package dto

import (
	"fmt"
	"strings"

	qmodel "masjidku_backend/internals/features/school/submissions_assesments/quizzes/model"

	"github.com/google/uuid"
)

/* ==========================================================================================
   REQUESTS — CREATE (single row) & BULK CREATE (single with options)
========================================================================================== */

// CreateQuizItemRequest — buat 1 baris quiz_items.
// - Untuk SINGLE: kirim beberapa request ini (satu per opsi) dengan question_id sama.
// - Untuk ESSAY : kirim satu request dengan kolom opsi dikosongkan (NULL).
type CreateQuizItemRequest struct {
	QuizItemsQuizID       uuid.UUID            `json:"quiz_items_quiz_id" validate:"required"`
	QuizItemsQuestionID   uuid.UUID            `json:"quiz_items_question_id" validate:"required"`
	QuizItemsQuestionType qmodel.QuizQuestionType `json:"quiz_items_question_type" validate:"required,oneof=single essay"`
	QuizItemsQuestionText string               `json:"quiz_items_question_text" validate:"required"`
	QuizItemsPoints       *float64             `json:"quiz_items_points" validate:"omitempty,gte=0"`

	// Opsi (WAJIB untuk SINGLE; harus kosong untuk ESSAY)
	QuizItemsOptionID        *uuid.UUID `json:"quiz_items_option_id" validate:"omitempty"`
	QuizItemsOptionText      *string    `json:"quiz_items_option_text" validate:"omitempty"`
	QuizItemsOptionIsCorrect *bool      `json:"quiz_items_option_is_correct" validate:"omitempty"`
}

// ToModel — builder model dari payload Create (tanpa set created_at/updated_at — biar GORM yang isi).
func (r *CreateQuizItemRequest) ToModel() (*qmodel.QuizItemModel, error) {
	points := 1.0
	if r.QuizItemsPoints != nil {
		points = *r.QuizItemsPoints
	}
	m := &qmodel.QuizItemModel{
		QuizItemsQuizID:        r.QuizItemsQuizID,
		QuizItemsQuestionID:    r.QuizItemsQuestionID,
		QuizItemsQuestionType:  r.QuizItemsQuestionType,
		QuizItemsQuestionText:  strings.TrimSpace(r.QuizItemsQuestionText),
		QuizItemsPoints:        points,
		QuizItemsOptionID:      r.QuizItemsOptionID,
		QuizItemsOptionText:    trimPtr(r.QuizItemsOptionText),
		QuizItemsOptionIsCorrect: r.QuizItemsOptionIsCorrect,
	}
	if err := m.RowShapeValid(); err != nil {
		return nil, err
	}
	return m, nil
}

// ========================= BULK CREATE (untuk SINGLE) =========================

// CreateSingleQuestionWithOptionsRequest — utility buat satu soal SINGLE + banyak opsi.
// Server akan mengkonversi ke banyak baris quiz_items dengan question_id sama.
type CreateSingleQuestionWithOptionsRequest struct {
	QuizItemsQuizID       uuid.UUID `json:"quiz_items_quiz_id" validate:"required"`
	QuizItemsQuestionID   uuid.UUID `json:"quiz_items_question_id" validate:"required"`
	QuizItemsQuestionText string    `json:"quiz_items_question_text" validate:"required"`
	QuizItemsPoints       *float64  `json:"quiz_items_points" validate:"omitempty,gte=0"`

	Options []SingleOptionPayload `json:"options" validate:"required,min=2,dive"`
}

// SingleOptionPayload — satu opsi untuk SINGLE
type SingleOptionPayload struct {
	// Jika ingin kontrol ID opsi dari luar, isi; kalau tidak, biarkan kosong → biar DB/servis yang set.
	QuizItemsOptionID *uuid.UUID `json:"quiz_items_option_id" validate:"omitempty"`
	OptionText        string     `json:"option_text" validate:"required"`
	IsCorrect         bool       `json:"is_correct"`
}

// ValidateDomainRules — pastikan tepat satu opsi benar.
func (r *CreateSingleQuestionWithOptionsRequest) ValidateDomainRules() error {
	correct := 0
	for _, op := range r.Options {
		if op.IsCorrect {
			correct++
		}
		if strings.TrimSpace(op.OptionText) == "" {
			return fmt.Errorf("option_text tidak boleh kosong")
		}
	}
	if correct != 1 {
		return fmt.Errorf("harus tepat satu opsi yang benar (is_correct=true); ditemukan %d", correct)
	}
	return nil
}

// ToModels — expand menjadi beberapa QuizItemModel (1 baris per opsi).
func (r *CreateSingleQuestionWithOptionsRequest) ToModels() ([]*qmodel.QuizItemModel, error) {
	if err := r.ValidateDomainRules(); err != nil {
		return nil, err
	}
	points := 1.0
	if r.QuizItemsPoints != nil {
		points = *r.QuizItemsPoints
	}
	var out []*qmodel.QuizItemModel
	for _, op := range r.Options {
		txt := strings.TrimSpace(op.OptionText)
		m := &qmodel.QuizItemModel{
			QuizItemsQuizID:        r.QuizItemsQuizID,
			QuizItemsQuestionID:    r.QuizItemsQuestionID,
			QuizItemsQuestionType:  qmodel.QuizQuestionTypeSingle,
			QuizItemsQuestionText:  strings.TrimSpace(r.QuizItemsQuestionText),
			QuizItemsPoints:        points,
			QuizItemsOptionID:      op.QuizItemsOptionID,   // boleh nil
			QuizItemsOptionText:    &txt,                   // wajib untuk SINGLE
			QuizItemsOptionIsCorrect: boolPtr(op.IsCorrect), // wajib untuk SINGLE
		}
		if err := m.RowShapeValid(); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

/* ==========================================================================================
   REQUEST — UPDATE (PARTIAL)
   Gunakan pointer agar field yg tidak dikirim tidak mengubah nilai di DB.
========================================================================================== */

type UpdateQuizItemRequest struct {
	// Catatan: ID baris di path param (/:id). DTO tak perlu memuat ID.
	QuizItemsQuizID       *uuid.UUID           `json:"quiz_items_quiz_id" validate:"omitempty"`
	QuizItemsQuestionID   *uuid.UUID           `json:"quiz_items_question_id" validate:"omitempty"`
	QuizItemsQuestionType *qmodel.QuizQuestionType `json:"quiz_items_question_type" validate:"omitempty,oneof=single essay"`
	QuizItemsQuestionText *string              `json:"quiz_items_question_text" validate:"omitempty"`
	QuizItemsPoints       *float64             `json:"quiz_items_points" validate:"omitempty,gte=0"`

	QuizItemsOptionID        *uuid.UUID `json:"quiz_items_option_id" validate:"omitempty"`
	QuizItemsOptionText      *string    `json:"quiz_items_option_text" validate:"omitempty"`
	QuizItemsOptionIsCorrect *bool      `json:"quiz_items_option_is_correct" validate:"omitempty"`
}

// ApplyToModel — patch ke model yang sudah di-load dari DB, lalu cek shape.
func (r *UpdateQuizItemRequest) ApplyToModel(m *qmodel.QuizItemModel) error {
	if r.QuizItemsQuizID != nil {
		m.QuizItemsQuizID = *r.QuizItemsQuizID
	}
	if r.QuizItemsQuestionID != nil {
		m.QuizItemsQuestionID = *r.QuizItemsQuestionID
	}
	if r.QuizItemsQuestionType != nil {
		m.QuizItemsQuestionType = *r.QuizItemsQuestionType
	}
	if r.QuizItemsQuestionText != nil {
		m.QuizItemsQuestionText = strings.TrimSpace(*r.QuizItemsQuestionText)
	}
	if r.QuizItemsPoints != nil {
		m.QuizItemsPoints = *r.QuizItemsPoints
	}
	// Opsi
	if r.QuizItemsOptionID != nil {
		m.QuizItemsOptionID = r.QuizItemsOptionID // bisa set ke nil untuk ESSAY
	}
	if r.QuizItemsOptionText != nil {
		m.QuizItemsOptionText = trimPtr(r.QuizItemsOptionText) // normalize trim
	}
	if r.QuizItemsOptionIsCorrect != nil {
		m.QuizItemsOptionIsCorrect = r.QuizItemsOptionIsCorrect
	}

	// Validasi shape (mirror constraint DB)
	if err := m.RowShapeValid(); err != nil {
		return err
	}
	return nil
}

/* ==========================================================================================
   RESPONSE DTO
========================================================================================== */

type QuizItemResponse struct {
	QuizItemsID             uuid.UUID              `json:"quiz_items_id"`
	QuizItemsQuizID         uuid.UUID              `json:"quiz_items_quiz_id"`
	QuizItemsQuestionID     uuid.UUID              `json:"quiz_items_question_id"`
	QuizItemsQuestionType   qmodel.QuizQuestionType `json:"quiz_items_question_type"`
	QuizItemsQuestionText   string                 `json:"quiz_items_question_text"`
	QuizItemsPoints         float64                `json:"quiz_items_points"`
	QuizItemsOptionID       *uuid.UUID             `json:"quiz_items_option_id,omitempty"`
	QuizItemsOptionText     *string                `json:"quiz_items_option_text,omitempty"`
	QuizItemsOptionIsCorrect *bool                 `json:"quiz_items_option_is_correct,omitempty"`
}

func FromModelQuizItem(m *qmodel.QuizItemModel) *QuizItemResponse {
	return &QuizItemResponse{
		QuizItemsID:              m.QuizItemsID,
		QuizItemsQuizID:          m.QuizItemsQuizID,
		QuizItemsQuestionID:      m.QuizItemsQuestionID,
		QuizItemsQuestionType:    m.QuizItemsQuestionType,
		QuizItemsQuestionText:    m.QuizItemsQuestionText,
		QuizItemsPoints:          m.QuizItemsPoints,
		QuizItemsOptionID:        m.QuizItemsOptionID,
		QuizItemsOptionText:      m.QuizItemsOptionText,
		QuizItemsOptionIsCorrect: m.QuizItemsOptionIsCorrect,
	}
}

func FromModelsQuizItems(items []*qmodel.QuizItemModel) []*QuizItemResponse {
	out := make([]*QuizItemResponse, 0, len(items))
	for _, it := range items {
		out = append(out, FromModelQuizItem(it))
	}
	return out
}

/* ==========================================================================================
   HELPERS
========================================================================================== */

func boolPtr(b bool) *bool { return &b }

// trimPtr: kembalikan *string hasil trim; jika kosong → tetap kosong (bukan nil).
// (Jika ingin kosong menjadi nil, ubah sesuai kebutuhan.)
func trimPtr(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	return &t
}
