package details

import (
	FaqRoutes "masjidku_backend/internals/features/home/faqs/route"
	NotificationRoutes "masjidku_backend/internals/features/home/notifications/route"
	EventRoutes "masjidku_backend/internals/features/masjids/events/route"
	LectureSessionsExamsRoutes "masjidku_backend/internals/features/masjids/lecture_sessions/exams/route"
	LectureSessionRoutes "masjidku_backend/internals/features/masjids/lecture_sessions/main/route"
	LectureSessionsAssetRoutes "masjidku_backend/internals/features/masjids/lecture_sessions/materials/route"
	LectureSessionsQuestionRoutes "masjidku_backend/internals/features/masjids/lecture_sessions/questions/route"
	LectureSessionsQuizRoutes "masjidku_backend/internals/features/masjids/lecture_sessions/quiz/route"
	LectureRoutes "masjidku_backend/internals/features/masjids/lectures/route"
	MasjidAdmin "masjidku_backend/internals/features/masjids/masjid_admins/route"
	masjidRoutes "masjidku_backend/internals/features/masjids/masjids/route"
	MasjidMore "masjidku_backend/internals/features/masjids/masjids_more/route"

	userFollowMasjid "masjidku_backend/internals/features/masjids/user_follow_masjids/route"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func MasjidPublicRoutes(r fiber.Router, db *gorm.DB) {
	// Ini endpoint yang boleh diakses publik tanpa login
	masjidRoutes.AllMasjidRoutes(r, db)
	MasjidMore.AllMasjidMoreRoutes(r, db)
	LectureRoutes.AllLectureRoutes(r, db)
	LectureSessionRoutes.AllLectureSessionRoutes(r, db)
	EventRoutes.AllEventRoutes(r, db)
	LectureSessionsAssetRoutes.AllLectureSessionsAssetRoutes(r, db)
	LectureSessionsQuizRoutes.LectureSessionsQuizUserRoutes(r, db)
	LectureSessionsQuestionRoutes.AllLectureSessionsQuestionRoutes(r, db)
}

func MasjidUserRoutes(r fiber.Router, db *gorm.DB) {
	// Ini endpoint yang butuh login user biasa (dengan token)
	userFollowMasjid.UserFollowMasjidsRoutes(r, db)
	FaqRoutes.AllFaqQuestionRoutes(r, db)
	LectureSessionsExamsRoutes.LectureSessionsExamsUserRoutes(r, db)
	LectureSessionsQuestionRoutes.LectureSessionsQuestionUserRoutes(r, db)
	// LectureRoutes.UserLectureRoutes(r, db)
}

func MasjidAdminRoutes(r fiber.Router, db *gorm.DB) {
	// Ini endpoint khusus admin masjid
	masjidRoutes.MasjidAdminRoutes(r, db)
	MasjidAdmin.MasjidAdminRoutes(r, db)
	MasjidMore.MasjidMoreRoutes(r, db)
	LectureRoutes.LectureRoutes(r, db)
	EventRoutes.EventRoutes(r, db)
	NotificationRoutes.AllNotificationRoutes(r, db)
	FaqRoutes.FaqQuestionAdminRoutes(r, db)
	LectureSessionRoutes.LectureSessionAdminRoutes(r, db)
	LectureSessionsExamsRoutes.LectureSessionsExamsAdminRoutes(r, db)
	LectureSessionsAssetRoutes.LectureSessionsAssetAdminRoutes(r, db)
	LectureSessionsQuestionRoutes.LectureSessionsQuestionAdminRoutes(r, db)
	LectureSessionsQuizRoutes.LectureSessionsQuizAdminRoutes(r, db)
}
