package main

import (
	"log"
	"os"
	"strings"

	"masjidku_backend/internals/configs"
	"masjidku_backend/internals/seeds"


	level "masjidku_backend/internals/seeds/progress/levels"
	rank "masjidku_backend/internals/seeds/progress/ranks"
	
	users "masjidku_backend/internals/seeds/users/auth"
	userProfiles "masjidku_backend/internals/seeds/users/users"
	tooltips "masjidku_backend/internals/seeds/utils/tooltips"
	survey "masjidku_backend/internals/seeds/users/surveys/survey_questions"
	user_survey "masjidku_backend/internals/seeds/users/surveys/user_surveys"
	
	masjids "masjidku_backend/internals/seeds/masjids/masjids"

)

func main() {
	configs.LoadEnv()
	db := configs.InitSeederDB()

	log.Println("🚀 Menjalankan seeder...")
	if len(os.Args) < 2 {
		log.Fatalln("❌ Mohon masukkan argumen seperti: all | users | users_profile | lessons | quizzes | progress")
	}

	switch strings.ToLower(os.Args[1]) {
	case "all":
		seeds.RunAllSeeds(db)
	case "users":
		users.SeedUsersFromJSON(db, "internals/seeds/users/auth/data_users.json")
	case "users_profile":
		userProfiles.SeedUsersProfileFromJSON(db, "internals/seeds/users/users/data_users_profiles.json")
	
	case "progress":
		level.SeedLevelRequirementsFromJSON(db, "internals/seeds/progress/levels/data_levels_requirements.json")
		rank.SeedRanksRequirementsFromJSON(db, "internals/seeds/progress/ranks/data_ranks_requirements.json")
	case "utils":
		tooltips.SeedTooltipsFromJSON(db, "internals/seeds/utils/tooltips/data_tooltips.json")

	case "survey_test_exam":
		survey.SeedSurveyQuestionsFromJSON(db, "internals/seeds/users/surveys/survey_questions/data_survey_questions.json")
		user_survey.SeedUserSurveysFromJSON(db, "internals/seeds/users/surveys/user_surveys/data_user_surveys.json")


	case "masjids":
		masjids.SeedMasjidsFromJSON(db, "internals/seeds/masjids/masjid/data_masjids.json")
	default:
		log.Fatalf("❌ Argumen '%s' tidak dikenali", os.Args[1])
	}
}
