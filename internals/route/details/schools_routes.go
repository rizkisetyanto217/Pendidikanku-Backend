package details

import (
	schoolRoutes "schoolku_backend/internals/features/lembaga/school_yayasans/schools/route"
	SchoolMore "schoolku_backend/internals/features/lembaga/school_yayasans/schools_more/route"
	SchoolAdmin "schoolku_backend/internals/features/lembaga/school_yayasans/teachers_students/route"
	CertificateRoutes "schoolku_backend/internals/features/schools/certificate/route"
	EventRoutes "schoolku_backend/internals/features/schools/events/route"
	LectureSessionRoutes "schoolku_backend/internals/features/schools/lecture_sessions/main/route"
	LectureSessionsAssetRoutes "schoolku_backend/internals/features/schools/lecture_sessions/materials/route"
	LectureSessionsQuestionRoutes "schoolku_backend/internals/features/schools/lecture_sessions/questions/route"
	LectureSessionsQuizRoutes "schoolku_backend/internals/features/schools/lecture_sessions/quizzes/route"
	LectureExamsRoutes "schoolku_backend/internals/features/schools/lectures/exams/route"
	LectureRoutes "schoolku_backend/internals/features/schools/lectures/main/route"

	userFollowSchool "schoolku_backend/internals/features/lembaga/school_yayasans/user_follow_schools/route"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func SchoolPublicRoutes(r fiber.Router, db *gorm.DB) {
	// Ini endpoint yang boleh diakses publik tanpa login
	schoolRoutes.AllSchoolRoutes(r, db)
	SchoolMore.AllSchoolMoreRoutes(r, db)
	LectureRoutes.AllLectureRoutes(r, db)
	LectureSessionRoutes.AllLectureSessionRoutes(r, db)
	EventRoutes.AllEventRoutes(r, db)
	LectureSessionRoutes.AllLectureSessionRoutes(r, db)

	LectureSessionsAssetRoutes.AllLectureSessionsAssetRoutes(r, db)
	LectureSessionsQuizRoutes.AllLectureSessionsQuizRoutes(r, db)
	LectureSessionsQuestionRoutes.AllLectureSessionsQuestionRoutes(r, db)
	CertificateRoutes.AllCertificateRoutes(r, db)

}

func SchoolUserRoutes(r fiber.Router, db *gorm.DB) {
	// Ini endpoint yang butuh login user biasa (dengan token)
	schoolRoutes.SchoolUserRoutes(r, db)
	userFollowSchool.UserFollowSchoolsRoutes(r, db)
	LectureRoutes.LectureUserRoutes(r, db)
	LectureSessionRoutes.LectureSessionUserRoutes(r, db)
	LectureSessionsQuizRoutes.LectureSessionsQuizUserRoutes(r, db)
	LectureExamsRoutes.LectureExamsUserRoutes(r, db)
	LectureSessionsQuestionRoutes.LectureSessionsQuestionUserRoutes(r, db)
	CertificateRoutes.CertificateUserRoutes(r, db)

	// LectureRoutes.UserLectureRoutes(r, db)
}

func SchoolAdminRoutes(r fiber.Router, db *gorm.DB) {
	// Ini endpoint khusus admin school
	schoolRoutes.SchoolAdminRoutes(r, db)
	SchoolAdmin.SchoolAdminRoutes(r, db)
	SchoolMore.SchoolMoreAdminRoutes(r, db)
	LectureRoutes.LectureAdminRoutes(r, db)
	EventRoutes.EventAdminRoutes(r, db)
	LectureSessionRoutes.LectureSessionAdminRoutes(r, db)
	LectureExamsRoutes.LectureExamsAdminRoutes(r, db)
	LectureSessionsAssetRoutes.LectureSessionsAssetAdminRoutes(r, db)
	LectureSessionsQuestionRoutes.LectureSessionsQuestionAdminRoutes(r, db)
	LectureSessionsQuizRoutes.LectureSessionsQuizAdminRoutes(r, db)
	CertificateRoutes.CertificateAdminRoutes(r, db)
}

func SchoolOwnerRoutes(r fiber.Router, db *gorm.DB) {
	// Ini endpoint khusus super admin
	schoolRoutes.SchoolOwnerRoutes(r, db)
}
