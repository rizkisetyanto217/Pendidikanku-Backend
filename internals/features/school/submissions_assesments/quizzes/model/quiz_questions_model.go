// file: internals/features/school/submissions_assesments/quizzes/model/quiz_question_model.go
package model

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type QuizQuestionType string

const (
	QuizQuestionTypeSingle QuizQuestionType = "single"
	QuizQuestionTypeEssay  QuizQuestionType = "essay"
)

type QuizQuestionModel struct {
	QuizQuestionsID          uuid.UUID        `gorm:"column:quiz_questions_id;type:uuid;primaryKey;default:gen_random_uuid()" json:"quiz_questions_id"`
	QuizQuestionsQuizID      uuid.UUID        `gorm:"column:quiz_questions_quiz_id;type:uuid;not null" json:"quiz_questions_quiz_id"`
	QuizQuestionsMasjidID    uuid.UUID        `gorm:"column:quiz_questions_masjid_id;type:uuid;not null" json:"quiz_questions_masjid_id"`
	QuizQuestionsType        QuizQuestionType `gorm:"column:quiz_questions_type;type:varchar(8);not null" json:"quiz_questions_type"`
	QuizQuestionsText        string           `gorm:"column:quiz_questions_text;type:text;not null" json:"quiz_questions_text"`
	QuizQuestionsPoints      float64          `gorm:"column:quiz_questions_points;type:numeric(6,2);not null;default:1" json:"quiz_questions_points"`
	QuizQuestionsAnswers     datatypes.JSON   `gorm:"column:quiz_questions_answers;type:jsonb" json:"quiz_questions_answers,omitempty"`
	QuizQuestionsCorrect     *string          `gorm:"column:quiz_questions_correct;type:char(1)" json:"quiz_questions_correct,omitempty"`
	QuizQuestionsExplanation *string          `gorm:"column:quiz_questions_explanation;type:text" json:"quiz_questions_explanation,omitempty"`

	QuizQuestionsCreatedAt time.Time      `gorm:"column:quiz_questions_created_at;autoCreateTime" json:"quiz_questions_created_at"`
	QuizQuestionsUpdatedAt time.Time      `gorm:"column:quiz_questions_updated_at;autoUpdateTime" json:"quiz_questions_updated_at"`
	QuizQuestionsDeletedAt gorm.DeletedAt `gorm:"column:quiz_questions_deleted_at" json:"quiz_questions_deleted_at,omitempty"`
}

func (QuizQuestionModel) TableName() string { return "quiz_questions" }

// ------------------------
// Helpers
// ------------------------

func (m *QuizQuestionModel) IsEssay() bool  { return m.QuizQuestionsType == QuizQuestionTypeEssay }
func (m *QuizQuestionModel) IsSingle() bool { return m.QuizQuestionsType == QuizQuestionTypeSingle }

// SingleOption bentuk array untuk SINGLE
type SingleOption struct {
	Text      string `json:"text"`
	IsCorrect bool   `json:"is_correct"`
}

// SetSingleAnswersObject → simpan jawaban model OBJECT {"A":"..","B":".."} + correct ('A'..'D')
func (m *QuizQuestionModel) SetSingleAnswersObject(opts map[string]string, correct string) error {
	if !m.IsSingle() {
		return errors.New("tipe soal bukan 'single'")
	}
	correct = strings.ToUpper(strings.TrimSpace(correct))
	if correct == "" {
		return errors.New("correct key wajib ('A'..'D')")
	}
	if _, ok := opts[correct]; !ok {
		return errors.New("correct key tidak ada pada answers object")
	}
	// batasi key
	for k := range opts {
		if !inSet(strings.ToUpper(k), "A", "B", "C", "D") {
			return errors.New("answers mengandung key di luar A..D")
		}
	}
	if len(opts) < 2 {
		return errors.New("minimal 2 opsi diperlukan")
	}
	b, _ := json.Marshal(opts)
	m.QuizQuestionsAnswers = datatypes.JSON(b)
	m.QuizQuestionsCorrect = &correct
	return nil
}

// SetSingleAnswersArray → simpan jawaban model ARRAY [{text,is_correct}]
func (m *QuizQuestionModel) SetSingleAnswersArray(options []SingleOption) error {
	if !m.IsSingle() {
		return errors.New("tipe soal bukan 'single'")
	}
	if len(options) < 2 {
		return errors.New("minimal 2 opsi diperlukan")
	}
	correctCount := 0
	for _, op := range options {
		if strings.TrimSpace(op.Text) == "" {
			return errors.New("option text tidak boleh kosong")
		}
		if op.IsCorrect {
			correctCount++
		}
	}
	if correctCount != 1 {
		return errors.New("harus tepat satu opsi dengan is_correct=true")
	}
	b, _ := json.Marshal(options)
	m.QuizQuestionsAnswers = datatypes.JSON(b)
	// mode ARRAY tidak pakai kolom 'correct' (biarkan NULL)
	m.QuizQuestionsCorrect = nil
	return nil
}

// ValidateShape → mirror sebagian besar CHECK constraints di DB agar cepat fail di app
func (m *QuizQuestionModel) ValidateShape() error {
	if m.IsEssay() {
		if len(m.QuizQuestionsAnswers) != 0 || (m.QuizQuestionsCorrect != nil && *m.QuizQuestionsCorrect != "") {
			return errors.New("ESSAY: answers & correct harus NULL/kosong")
		}
		return nil
	}

	// SINGLE
	if len(m.QuizQuestionsAnswers) == 0 {
		return errors.New("SINGLE: answers wajib diisi")
	}
	// Coba parse sebagai OBJECT
	var obj map[string]any
	if err := json.Unmarshal(m.QuizQuestionsAnswers, &obj); err == nil && obj != nil {
		// OBJECT mode
		if m.QuizQuestionsCorrect == nil || *m.QuizQuestionsCorrect == "" {
			return errors.New("SINGLE OBJECT: 'correct' wajib diisi ('A'..'D')")
		}
		c := strings.ToUpper(*m.QuizQuestionsCorrect)
		if !inSet(c, "A", "B", "C", "D") {
			return errors.New("SINGLE OBJECT: correct harus salah satu A..D")
		}
		if _, ok := obj[c]; !ok {
			return errors.New("SINGLE OBJECT: kunci correct tidak ada pada answers")
		}
		if len(obj) < 2 {
			return errors.New("SINGLE OBJECT: minimal 2 opsi")
		}
		for k := range obj {
			if !inSet(strings.ToUpper(k), "A", "B", "C", "D") {
				return errors.New("SINGLE OBJECT: answers mengandung key di luar A..D")
			}
		}
		return nil
	}

	// Coba parse sebagai ARRAY
	var arr []map[string]any
	if err := json.Unmarshal(m.QuizQuestionsAnswers, &arr); err == nil {
		if len(arr) < 2 {
			return errors.New("SINGLE ARRAY: minimal 2 opsi")
		}
		correct := 0
		for _, e := range arr {
			if _, has := e["text"]; !has {
				return errors.New("SINGLE ARRAY: setiap opsi wajib punya 'text'")
			}
			if v, ok := e["is_correct"]; ok {
				if vb, ok2 := v.(bool); ok2 && vb {
					correct++
				}
			}
		}
		if correct != 1 {
			return errors.New("SINGLE ARRAY: harus tepat satu 'is_correct'=true")
		}
		// ARRAY mode: kolom correct harus NULL
		if m.QuizQuestionsCorrect != nil && *m.QuizQuestionsCorrect != "" {
			return errors.New("SINGLE ARRAY: kolom 'correct' harus NULL")
		}
		return nil
	}

	return errors.New("answers bukan OBJECT atau ARRAY JSON yang valid")
}

func inSet(v string, set ...string) bool {
	for _, s := range set {
		if v == s {
			return true
		}
	}
	return false
}
