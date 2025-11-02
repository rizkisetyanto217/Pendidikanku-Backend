package main

import (
	"log"
	"os"
	"strings"

	"schoolku_backend/internals/configs"
	"schoolku_backend/internals/seeds"

	// users "schoolku_backend/internals/seeds/users/auth"
	survey "schoolku_backend/internals/seeds/users/surveys/survey_questions"
	user_survey "schoolku_backend/internals/seeds/users/surveys/user_surveys"
)

func main() {
	configs.LoadEnv()
	db := configs.InitSeederDB()

	log.Println("🚀 Menjalankan seeder...")
	if len(os.Args) < 2 {
		log.Fatalln("❌ Mohon masukkan argumen seperti: all | users | user_profiles | lessons | quizzes | progress")
	}

	switch strings.ToLower(os.Args[1]) {
	case "all":
		seeds.RunAllSeeds(db)
	case "users":
		// users.SeedUsersFromJSON(db, "internals/seeds/users/auth/data_users.json")
	case "user_profiles":
		// userProfiles.SeedUsersProfileFromJSON(db, "internals/seeds/users/users/data_users_profiles.json")
	case "survey_test_exam":
		survey.SeedSurveyQuestionsFromJSON(db, "internals/seeds/users/surveys/survey_questions/data_survey_questions.json")
		user_survey.SeedUserSurveysFromJSON(db, "internals/seeds/users/surveys/user_surveys/data_user_surveys.json")

	case "schools":
		// schools.SeedSchoolsFromJSON(db, "internals/seeds/schools/school/data_schools.json")
	default:
		log.Fatalf("❌ Argumen '%s' tidak dikenali", os.Args[1])
	}
}
