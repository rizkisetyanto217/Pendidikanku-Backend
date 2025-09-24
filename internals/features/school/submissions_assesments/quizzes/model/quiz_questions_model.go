package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

/* =========================================================
   ENUM / Types
   ========================================================= */

type QuizQuestionType string

const (
	QuizQuestionTypeSingle QuizQuestionType = "single"
	QuizQuestionTypeEssay  QuizQuestionType = "essay"
)

/* =========================================================
   QuizQuestion (quiz_questions)
   ========================================================= */

type QuizQuestionModel struct {
	// PK
	QuizQuestionID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:quiz_question_id" json:"quiz_question_id"`

	// Relasi → quizzes
	QuizQuestionQuizID uuid.UUID `gorm:"type:uuid;not null;column:quiz_question_quiz_id;index:idx_qq_quiz_alive,priority:1" json:"quiz_question_quiz_id"`

	// Tenant
	QuizQuestionMasjidID uuid.UUID `gorm:"type:uuid;not null;column:quiz_question_masjid_id;index:idx_qq_masjid_alive,priority:1" json:"quiz_question_masjid_id"`

	// Jenis soal
	QuizQuestionType QuizQuestionType `gorm:"type:varchar(8);not null;column:quiz_question_type" json:"quiz_question_type"`

	// Isi & penilaian
	QuizQuestionText        string         `gorm:"type:text;not null;column:quiz_question_text" json:"quiz_question_text"`
	QuizQuestionPoints      float64        `gorm:"type:numeric(6,2);not null;default:1;column:quiz_question_points" json:"quiz_question_points"`
	QuizQuestionAnswers     datatypes.JSON `gorm:"type:jsonb;column:quiz_question_answers" json:"quiz_question_answers,omitempty"`
	QuizQuestionCorrect     *string        `gorm:"type:char(1);column:quiz_question_correct" json:"quiz_question_correct,omitempty"`
	QuizQuestionExplanation *string        `gorm:"type:text;column:quiz_question_explanation" json:"quiz_question_explanation,omitempty"`

	// Unique pair (quiz_question_id, quiz_question_quiz_id)
	_ struct{} `gorm:"uniqueIndex:uq_quiz_question_id_quiz"`

	// Timestamps
	QuizQuestionCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();autoCreateTime;column:quiz_question_created_at" json:"quiz_question_created_at"`
	QuizQuestionUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();autoUpdateTime;column:quiz_question_updated_at" json:"quiz_question_updated_at"`
	QuizQuestionDeletedAt gorm.DeletedAt `gorm:"column:quiz_question_deleted_at;index" json:"quiz_question_deleted_at,omitempty"`

	// Parent
	Quiz *QuizModel `gorm:"foreignKey:QuizQuestionQuizID;references:QuizID" json:"quiz,omitempty"`
}

func (QuizQuestionModel) TableName() string { return "quiz_questions" }

func (m *QuizQuestionModel) ValidateShape() error {
	// text wajib
	if strings.TrimSpace(m.QuizQuestionText) == "" {
		return errors.New("quiz_question_text required")
	}
	// points >= 0
	if m.QuizQuestionPoints < 0 {
		return errors.New("quiz_question_points must be >= 0")
	}

	switch m.QuizQuestionType {
	case QuizQuestionTypeEssay:
		// essay: tidak boleh punya correct; answers boleh kosong
		if m.QuizQuestionCorrect != nil {
			return errors.New("essay question must not have quiz_question_correct")
		}
		return nil

	case QuizQuestionTypeSingle:
		// single: harus ada correct A..D dan answers valid (array/object)
		if m.QuizQuestionCorrect == nil {
			return errors.New("single choice requires quiz_question_correct")
		}
		c := strings.ToUpper(strings.TrimSpace(*m.QuizQuestionCorrect))
		if c != "A" && c != "B" && c != "C" && c != "D" {
			return errors.New("quiz_question_correct must be one of A,B,C,D")
		}
		if len(m.QuizQuestionAnswers) == 0 {
			return errors.New("single choice requires quiz_question_answers")
		}
		var v any
		if err := json.Unmarshal(m.QuizQuestionAnswers, &v); err != nil {
			return fmt.Errorf("quiz_question_answers invalid json: %w", err)
		}
		switch vv := v.(type) {
		case []any:
			if len(vv) < 2 {
				return errors.New("quiz_question_answers array must have at least 2 options")
			}
		case map[string]any:
			// opsional: cek minimal A & B ada
			// _, hasA := vv["A"]; _, hasB := vv["B"]
			// if !hasA || !hasB { return errors.New("answers object should contain at least A and B") }
		default:
			return errors.New("quiz_question_answers must be array or object")
		}
		return nil

	default:
		return errors.New("invalid quiz_question_type")
	}
}
