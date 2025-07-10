package survey

import (
	"encoding/json"
	"log"
	"masjidku_backend/internals/features/users/survey/model"
	"os"

	"gorm.io/gorm"
)

type SurveyQuestionSeed struct {
	SurveyQuestionText       string   `json:"survey_question_text"`
	SurveyQuestionAnswer     []string `json:"survey_question_answer"`
	SurveyQuestionOrderIndex int      `json:"survey_question_order_index"`
}

func SeedSurveyQuestionsFromJSON(db *gorm.DB, filePath string) {
	log.Println("📥 Membaca file survey questions:", filePath)

	file, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("❌ Gagal membaca file JSON: %v", err)
	}

	var seeds []SurveyQuestionSeed
	if err := json.Unmarshal(file, &seeds); err != nil {
		log.Fatalf("❌ Gagal decode JSON: %v", err)
	}

	// Ambil semua survey_question_text yang sudah ada
	var existingQuestions []string
	if err := db.Model(&model.SurveyQuestion{}).
		Pluck("survey_question_text", &existingQuestions).Error; err != nil {
		log.Fatalf("❌ Gagal ambil data existing: %v", err)
	}

	existingMap := make(map[string]bool)
	for _, q := range existingQuestions {
		existingMap[q] = true
	}

	var newQuestions []model.SurveyQuestion
	for _, s := range seeds {
		if existingMap[s.SurveyQuestionText] {
			log.Printf("ℹ️ Pertanyaan '%s' sudah ada, dilewati.", s.SurveyQuestionText)
			continue
		}

		newQuestions = append(newQuestions, model.SurveyQuestion{
			SurveyQuestionText:       s.SurveyQuestionText,
			SurveyQuestionAnswer:     s.SurveyQuestionAnswer,
			SurveyQuestionOrderIndex: s.SurveyQuestionOrderIndex,
		})
	}

	if len(newQuestions) > 0 {
		if err := db.Create(&newQuestions).Error; err != nil {
			log.Fatalf("❌ Gagal insert survey_questions: %v", err)
		}
		log.Printf("✅ Berhasil insert %d survey questions", len(newQuestions))
	} else {
		log.Println("ℹ️ Tidak ada pertanyaan baru untuk diinsert.")
	}
}
