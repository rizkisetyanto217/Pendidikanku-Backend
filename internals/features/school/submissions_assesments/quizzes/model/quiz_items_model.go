// file: internals/features/school/quizzes/model/quiz_item_model.go
package model

import (
	"database/sql/driver"
	"fmt"

	"github.com/google/uuid"
)

/* ============================================================================
   ENUM-like: question_type ('single' | 'essay')
============================================================================ */
type QuizQuestionType string

const (
	QuizQuestionTypeSingle QuizQuestionType = "single"
	QuizQuestionTypeEssay  QuizQuestionType = "essay"
)

func (t QuizQuestionType) String() string { return string(t) }
func (t QuizQuestionType) Valid() bool {
	return t == QuizQuestionTypeSingle || t == QuizQuestionTypeEssay
}

// Optional: make it friendly to sql.Scanner / driver.Valuer
func (t *QuizQuestionType) Scan(value any) error {
	if value == nil {
		*t = ""
		return nil
	}
	switch v := value.(type) {
	case string:
		*t = QuizQuestionType(v)
	case []byte:
		*t = QuizQuestionType(string(v))
	default:
		return fmt.Errorf("unsupported type for QuizQuestionType: %T", value)
	}
	if !t.Valid() {
		return fmt.Errorf("invalid QuizQuestionType: %q", *t)
	}
	return nil
}
func (t QuizQuestionType) Value() (driver.Value, error) {
	if t == "" {
		return nil, nil
	}
	if !t.Valid() {
		return nil, fmt.Errorf("invalid QuizQuestionType: %q", t)
	}
	return string(t), nil
}

/* ============================================================================
   MODEL: quiz_items
   Catatan:
   - Baris merepresentasikan "opsi" untuk SINGLE; dan satu baris tunggal untuk ESSAY.
   - Partial unique indexes & predicate indexes sudah ditangani di SQL (DDL).
============================================================================ */
type QuizItemModel struct {
	// PK
	QuizItemsID uuid.UUID `json:"quiz_items_id" gorm:"column:quiz_items_id;type:uuid;default:gen_random_uuid();primaryKey"`

	// FK → quizzes(quizzes_id)
	QuizItemsQuizID uuid.UUID `json:"quiz_items_quiz_id" gorm:"column:quiz_items_quiz_id;type:uuid;not null;index:idx_quiz_items_quiz"`

	// (Optional) Relasi — sesuaikan 'references' dgn field PK di QuizModel milikmu.
	// Misal jika di model quiz kamu memakai field 'QuizzesID uuid.UUID `gorm:"column:quizzes_id;..."`'
	// maka gunakan references:QuizzesID (jangan lupa import/declare type QuizModel).
	// Quiz *QuizModel `gorm:"foreignKey:QuizItemsQuizID;references:QuizzesID"`

	// Info soal (shared oleh beberapa baris untuk SINGLE)
	QuizItemsQuestionID   uuid.UUID        `json:"quiz_items_question_id" gorm:"column:quiz_items_question_id;type:uuid;not null;index:idx_quiz_items_question;index:idx_quiz_items_quiz_question,priority:2"`
	QuizItemsQuestionType QuizQuestionType `json:"quiz_items_question_type" gorm:"column:quiz_items_question_type;type:varchar(8);not null;index:idx_quiz_items_type"`
	QuizItemsQuestionText string           `json:"quiz_items_question_text" gorm:"column:quiz_items_question_text;type:text;not null"`
	QuizItemsPoints       float64          `json:"quiz_items_points" gorm:"column:quiz_items_points;type:numeric(6,2);not null;default:1"`

	// Info opsi (Wajib terisi untuk SINGLE; semuanya NULL untuk ESSAY)
	QuizItemsOptionID       *uuid.UUID `json:"quiz_items_option_id,omitempty" gorm:"column:quiz_items_option_id;type:uuid;index:idx_quiz_items_quiz_question,priority:3"`
	QuizItemsOptionText     *string    `json:"quiz_items_option_text,omitempty" gorm:"column:quiz_items_option_text;type:text"`
	QuizItemsOptionIsCorrect *bool     `json:"quiz_items_option_is_correct,omitempty" gorm:"column:quiz_items_option_is_correct"`
}

// Nama tabel eksplisit
func (QuizItemModel) TableName() string { return "quiz_items" }

/* ============================================================================
   Helper methods (opsional, untuk validasi bentuk data sebelum insert/update)
============================================================================ */
func (m *QuizItemModel) IsEssay() bool  { return m.QuizItemsQuestionType == QuizQuestionTypeEssay }
func (m *QuizItemModel) IsSingle() bool { return m.QuizItemsQuestionType == QuizQuestionTypeSingle }

// RowShapeValid meniru constraint ck_quiz_items_shape agar error lebih awal di app layer
func (m *QuizItemModel) RowShapeValid() error {
	if !m.QuizItemsQuestionType.Valid() {
		return fmt.Errorf("quiz_items_question_type must be 'single' or 'essay'")
	}
	if m.IsSingle() {
		if m.QuizItemsOptionID == nil || m.QuizItemsOptionText == nil || m.QuizItemsOptionIsCorrect == nil {
			return fmt.Errorf("single: option_id, option_text, option_is_correct must be non-NULL")
		}
	} else { // essay
		if m.QuizItemsOptionID != nil || m.QuizItemsOptionText != nil || m.QuizItemsOptionIsCorrect != nil {
			return fmt.Errorf("essay: option fields must be NULL")
		}
	}
	return nil
}
