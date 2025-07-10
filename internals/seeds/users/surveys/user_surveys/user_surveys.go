package survey

import (
	"encoding/json"
	"log"
	"masjidku_backend/internals/features/users/survey/model"
	"os"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserSurveySeed struct {
	UserSurveyUserID     uuid.UUID `json:"user_survey_user_id"`
	UserSurveyQuestionID int       `json:"user_survey_question_id"`
	UserSurveyAnswer     string    `json:"user_survey_answer"`
}

func SeedUserSurveysFromJSON(db *gorm.DB, filePath string) {
	log.Println("📥 Membaca file user_survey:", filePath)

	file, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("❌ Gagal membaca file JSON: %v", err)
	}

	var seeds []UserSurveySeed
	if err := json.Unmarshal(file, &seeds); err != nil {
		log.Fatalf("❌ Gagal decode JSON: %v", err)
	}

	var userSurveys []model.UserSurvey
	for _, s := range seeds {
		userSurveys = append(userSurveys, model.UserSurvey{
			UserSurveyUserID:     s.UserSurveyUserID,
			UserSurveyQuestionID: s.UserSurveyQuestionID,
			UserSurveyAnswer:     s.UserSurveyAnswer,
		})
	}

	if len(userSurveys) > 0 {
		if err := db.Create(&userSurveys).Error; err != nil {
			log.Fatalf("❌ Gagal insert user_surveys: %v", err)
		}
		log.Printf("✅ Berhasil insert %d user survey", len(userSurveys))
	} else {
		log.Println("ℹ️ Tidak ada data user survey untuk diinsert.")
	}
}
