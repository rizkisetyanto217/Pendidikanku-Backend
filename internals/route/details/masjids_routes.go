package details

import (
	CertificateRoutes "masjidku_backend/internals/features/masjids/certificate/route"
	EventRoutes "masjidku_backend/internals/features/masjids/events/route"
	LectureSessionRoutes "masjidku_backend/internals/features/masjids/lecture_sessions/main/route"
	LectureSessionsAssetRoutes "masjidku_backend/internals/features/masjids/lecture_sessions/materials/route"
	LectureSessionsQuestionRoutes "masjidku_backend/internals/features/masjids/lecture_sessions/questions/route"
	LectureSessionsQuizRoutes "masjidku_backend/internals/features/masjids/lecture_sessions/quizzes/route"
	LectureExamsRoutes "masjidku_backend/internals/features/masjids/lectures/exams/route"
	LectureRoutes "masjidku_backend/internals/features/masjids/lectures/main/route"
	MasjidAdmin "masjidku_backend/internals/features/lembaga/masjid_teachers/admins_teachers/route"
	masjidRoutes "masjidku_backend/internals/features/lembaga/masjids/route"
	MasjidMore "masjidku_backend/internals/features/lembaga/masjids_more/route"

	userFollowMasjid "masjidku_backend/internals/features/lembaga/user_follow_masjids/route"

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
	LectureSessionRoutes.AllLectureSessionRoutes(r, db)

	LectureSessionsAssetRoutes.AllLectureSessionsAssetRoutes(r, db)
	LectureSessionsQuizRoutes.AllLectureSessionsQuizRoutes(r, db)
	LectureSessionsQuestionRoutes.AllLectureSessionsQuestionRoutes(r, db)
	CertificateRoutes.AllCertificateRoutes(r, db)

}


func MasjidUserRoutes(r fiber.Router, db *gorm.DB) {
	// Ini endpoint yang butuh login user biasa (dengan token)
	masjidRoutes.MasjidUserRoutes(r, db)
	userFollowMasjid.UserFollowMasjidsRoutes(r, db)
	LectureRoutes.LectureUserRoutes(r, db)
	LectureSessionRoutes.LectureSessionUserRoutes(r, db)
	LectureSessionsQuizRoutes.LectureSessionsQuizUserRoutes(r, db)
	LectureExamsRoutes.LectureExamsUserRoutes(r, db)
	LectureSessionsQuestionRoutes.LectureSessionsQuestionUserRoutes(r, db)
	CertificateRoutes.CertificateUserRoutes(r, db)
	
	// LectureRoutes.UserLectureRoutes(r, db)
}

func MasjidAdminRoutes(r fiber.Router, db *gorm.DB) {
	// Ini endpoint khusus admin masjid
	masjidRoutes.MasjidAdminRoutes(r, db)
	MasjidAdmin.MasjidAdminRoutes(r, db)
	MasjidMore.MasjidMoreAdminRoutes(r, db)
	LectureRoutes.LectureAdminRoutes(r, db)
	EventRoutes.EventAdminRoutes(r, db)
	LectureSessionRoutes.LectureSessionAdminRoutes(r, db)
	LectureExamsRoutes.LectureExamsAdminRoutes(r, db)
	LectureSessionsAssetRoutes.LectureSessionsAssetAdminRoutes(r, db)
	LectureSessionsQuestionRoutes.LectureSessionsQuestionAdminRoutes(r, db)
	LectureSessionsQuizRoutes.LectureSessionsQuizAdminRoutes(r, db)
	CertificateRoutes.CertificateAdminRoutes(r, db)
}
