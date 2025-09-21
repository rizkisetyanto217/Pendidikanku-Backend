// internals/features/school/quizzes/model/quiz_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type QuizModel struct {
	QuizzesID           uuid.UUID      `gorm:"column:quizzes_id;type:uuid;primaryKey;default:gen_random_uuid()" json:"quizzes_id"`
	QuizzesMasjidID     uuid.UUID      `gorm:"column:quizzes_masjid_id;type:uuid;not null"                     json:"quizzes_masjid_id"`
	QuizzesAssessmentID *uuid.UUID     `gorm:"column:quizzes_assessment_id;type:uuid"                          json:"quizzes_assessment_id,omitempty"`

	// SLUG (opsional; unik per tenant saat alive)
	QuizzesSlug *string `gorm:"column:quizzes_slug;type:varchar(160)" json:"quizzes_slug,omitempty"`

	QuizzesTitle        string   `gorm:"column:quizzes_title;type:varchar(180);not null" json:"quizzes_title"`
	QuizzesDescription  *string  `gorm:"column:quizzes_description;type:text"            json:"quizzes_description,omitempty"`
	QuizzesIsPublished  bool     `gorm:"column:quizzes_is_published;not null;default:false" json:"quizzes_is_published"`
	QuizzesTimeLimitSec *int     `gorm:"column:quizzes_time_limit_sec"                   json:"quizzes_time_limit_sec,omitempty"`

	QuizzesCreatedAt time.Time     `gorm:"column:quizzes_created_at;type:timestamptz;not null;autoCreateTime" json:"quizzes_created_at"`
	QuizzesUpdatedAt time.Time     `gorm:"column:quizzes_updated_at;type:timestamptz;not null;autoUpdateTime" json:"quizzes_updated_at"`
	QuizzesDeletedAt gorm.DeletedAt`gorm:"column:quizzes_deleted_at;index"                                     json:"quizzes_deleted_at,omitempty"`

	// Relasi ke table pertanyaan (sesuaikan nama field FK di model question-mu)
	Questions []QuizQuestionModel `gorm:"foreignKey:QuizQuestionsQuizID;references:QuizzesID" json:"-"`
}

func (QuizModel) TableName() string { return "quizzes" }
