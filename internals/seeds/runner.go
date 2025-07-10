package seeds

import (
	// categories "masjidku_backend/internals/seeds/lessons/categories/categories"
	// categories_news "masjidku_backend/internals/seeds/lessons/categories/categories_news"

	// difficulties "masjidku_backend/internals/seeds/lessons/difficulties/difficulties"
	// difficulties_news "masjidku_backend/internals/seeds/lessons/difficulties/difficulties_news"

	// subcategories "masjidku_backend/internals/seeds/lessons/subcategories/subcategories"
	// subcategories_news "masjidku_backend/internals/seeds/lessons/subcategories/subcategories_news"

	// themes_or_levels "masjidku_backend/internals/seeds/lessons/themes_or_levels/themes_or_levels"
	// themes_or_levels_news "masjidku_backend/internals/seeds/lessons/themes_or_levels/themes_or_levels_news"

	// units "masjidku_backend/internals/seeds/lessons/units/units"
	// units_news "masjidku_backend/internals/seeds/lessons/units/units_news"

	// evaluations "masjidku_backend/internals/seeds/quizzes/evaluations"
	// exams "masjidku_backend/internals/seeds/quizzes/exams"
	// questions "masjidku_backend/internals/seeds/quizzes/questions"
	// quizzes "masjidku_backend/internals/seeds/quizzes/quizzes"
	// reading "masjidku_backend/internals/seeds/quizzes/readings"
	// section_quizzes "masjidku_backend/internals/seeds/quizzes/section_quizzes"

	// level "masjidku_backend/internals/seeds/progress/levels"
	// rank "masjidku_backend/internals/seeds/progress/ranks"

	// users "masjidku_backend/internals/seeds/users/auth"

	"gorm.io/gorm"
)

func RunAllSeeds(db *gorm.DB) {

	//* Category
	// difficulties.SeedDifficultiesFromJSON(db, "internals/seeds/category/difficulty/data_difficulty.json")
	// difficulties_news.SeedDifficultiesNewsFromJSON(db, "internals/seeds/category/difficulty_news/data_difficulty_news.json")

	// categories.SeedCategoriesFromJSON(db, "internals/seeds/category/category/data_category.json")
	// categories_news.SeedCategoriesNewsFromJSON(db, "internals/seeds/category/category_news/data_category_news.json")

	// subcategories.SeedSubcategoriesFromJSON(db, "internals/seeds/category/subcategory/data_subcategory.json")
	// subcategories_news.SeedSubcategoriesNewsFromJSON(db, "internals/seeds/category/subcategory_news/data_subcategory_news.json")

	// themes_or_levels.SeedThemesOrLevelsFromJSON(db, "internals/seeds/category/themes_or_levels/data_themes_or_levels.json")
	// themes_or_levels_news.SeedThemesOrLevelsNewsFromJSON(db, "internals/seeds/category/themes_or_levels_news/data_themes_or_levels_news.json")

	// units.SeedUnitsFromJSON(db, "internals/seeds/lessons/units/units/data_units.json")
	// units_news.SeedUnitsNewsFromJSON(db, "internals/seeds/lessons//units/units_news/data_units_news.json")

	// //* User
	// users.SeedUsersFromJSON(db, "internals/seeds/users/auth/data_users.json")

	// //* Quizzes
	// evaluations.SeedEvaluationsFromJSON(db, "internals/seeds/quizzes/evaluations/data_evaluations.json")
	// exams.SeedExamsFromJSON(db, "internals/seeds/quizzes/exams/data_exams.json")
	// questions.SeedQuestionsFromJSON(db, "internals/seeds/quizzes/questions/data_questions.json")
	// section_quizzes.SeedSectionQuizzesFromJSON(db, "internals/seeds/quizzes/section_quizzes/data_section_quizzes.json")
	// quizzes.SeedQuizzesFromJSON(db, "internals/seeds/quizzes/quizzes/data_quizzes.json")
	// reading.SeedReadingsFromJSON(db, "internals/seeds/quizzes/readings/data_readings.json")

	// //* Progress
	// level.SeedLevelRequirementsFromJSON(db, "internals/seeds/progress/levels/data_levels_requirements.json")
	// rank.SeedRanksRequirementsFromJSON(db, "internals/seeds/progress/ranks/data_ranks_requirements.json")

}
