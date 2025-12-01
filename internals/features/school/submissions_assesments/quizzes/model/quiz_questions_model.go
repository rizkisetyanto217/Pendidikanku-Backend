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
   History item
   ========================================================= */

type QuizQuestionHistoryItem struct {
	Version     int             `json:"version"`
	SavedAt     time.Time       `json:"saved_at"`
	ChangeKind  string          `json:"change_kind,omitempty"` // "major" / "minor" (kalau mau dipakai nanti)
	Text        string          `json:"text"`
	Answers     json.RawMessage `json:"answers,omitempty"`
	Correct     *string         `json:"correct,omitempty"`
	Explanation *string         `json:"explanation,omitempty"`
	Points      float64         `json:"points"`
}

/* =========================================================
   QuizQuestion (quiz_questions)
   ========================================================= */

type QuizQuestionModel struct {
	// PK
	QuizQuestionID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:quiz_question_id" json:"quiz_question_id"`

	// Relasi → quizzes
	QuizQuestionQuizID uuid.UUID `gorm:"type:uuid;not null;column:quiz_question_quiz_id;index:idx_qq_quiz_alive,priority:1" json:"quiz_question_quiz_id"`

	// Tenant
	QuizQuestionSchoolID uuid.UUID `gorm:"type:uuid;not null;column:quiz_question_school_id;index:idx_qq_school_alive,priority:1" json:"quiz_question_school_id"`

	// Jenis soal
	QuizQuestionType QuizQuestionType `gorm:"type:varchar(8);not null;column:quiz_question_type" json:"quiz_question_type"`

	// Isi & penilaian
	QuizQuestionText   string  `gorm:"type:text;not null;column:quiz_question_text" json:"quiz_question_text"`
	QuizQuestionPoints float64 `gorm:"type:numeric(6,2);not null;default:1;column:quiz_question_points" json:"quiz_question_points"`

	// JSON object: { "A": "...", "B": "...", ... }
	QuizQuestionAnswers datatypes.JSON `gorm:"type:jsonb;column:quiz_question_answers" json:"quiz_question_answers,omitempty"`
	// Key jawaban benar, harus salah satu key di Answers
	QuizQuestionCorrect     *string `gorm:"type:text;column:quiz_question_correct" json:"quiz_question_correct,omitempty"`
	QuizQuestionExplanation *string `gorm:"type:text;column:quiz_question_explanation" json:"quiz_question_explanation,omitempty"`

	// Versioning ringan
	QuizQuestionVersion int            `gorm:"type:int;not null;default:1;column:quiz_question_version" json:"quiz_question_version"`
	QuizQuestionHistory datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'::jsonb;column:quiz_question_history" json:"quiz_question_history"`

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

/* =========================================================
   Domain validation
   ========================================================= */

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
		// ESSAY: tidak boleh punya answers & correct (mirror constraint DB)
		if len(m.QuizQuestionAnswers) > 0 {
			return errors.New("essay question must not have quiz_question_answers")
		}
		if m.QuizQuestionCorrect != nil {
			return errors.New("essay question must not have quiz_question_correct")
		}
		return nil

	case QuizQuestionTypeSingle:
		// SINGLE: wajib punya correct & answers
		if m.QuizQuestionCorrect == nil || strings.TrimSpace(*m.QuizQuestionCorrect) == "" {
			return errors.New("single choice requires quiz_question_correct")
		}
		if len(m.QuizQuestionAnswers) == 0 {
			return errors.New("single choice requires quiz_question_answers")
		}

		// answers harus JSON object: { "A": "...", "B": "...", ... }
		var raw any
		if err := json.Unmarshal(m.QuizQuestionAnswers, &raw); err != nil {
			return fmt.Errorf("quiz_question_answers invalid json: %w", err)
		}

		obj, ok := raw.(map[string]any)
		if !ok {
			return errors.New("quiz_question_answers must be a json object with keys like A,B,C")
		}
		if len(obj) < 2 {
			return errors.New("quiz_question_answers must contain at least 2 options")
		}

		// cek: correct harus salah satu key di answers
		key := strings.TrimSpace(*m.QuizQuestionCorrect)
		if key == "" {
			return errors.New("quiz_question_correct cannot be empty")
		}
		if _, exists := obj[key]; !exists {
			return fmt.Errorf("quiz_question_correct %q must be one of the keys in quiz_question_answers", key)
		}

		return nil

	default:
		return errors.New("invalid quiz_question_type")
	}
}

/* =========================================================
   History helper
   ========================================================= */

// AppendHistorySnapshot dipanggil saat perubahan "major":
// - Menyimpan snapshot state lama ke QuizQuestionHistory
// - Menaikkan QuizQuestionVersion
func (m *QuizQuestionModel) AppendHistorySnapshot(changeKind string) error {
	kind := strings.ToLower(strings.TrimSpace(changeKind))

	var items []QuizQuestionHistoryItem
	if len(m.QuizQuestionHistory) > 0 {
		_ = json.Unmarshal(m.QuizQuestionHistory, &items)
	}

	item := QuizQuestionHistoryItem{
		Version:     m.QuizQuestionVersion,
		SavedAt:     time.Now().UTC(),
		ChangeKind:  kind,
		Text:        m.QuizQuestionText,
		Answers:     json.RawMessage(m.QuizQuestionAnswers),
		Correct:     m.QuizQuestionCorrect,
		Explanation: m.QuizQuestionExplanation,
		Points:      m.QuizQuestionPoints,
	}

	items = append(items, item)

	buf, err := json.Marshal(items)
	if err != nil {
		return fmt.Errorf("failed to marshal quiz_question_history: %w", err)
	}

	m.QuizQuestionHistory = datatypes.JSON(buf)
	m.QuizQuestionVersion++

	return nil
}
