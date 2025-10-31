// internals/route/details/lembaga_details_routes.go
package details

import (
	// ====== Lembaga features ======
	YayasanRoutes "masjidku_backend/internals/features/lembaga/masjid_yayasans/yayasans/route"
	LembagaStatsRoutes "masjidku_backend/internals/features/lembaga/stats/lembaga_stats/route"
	SemesterStatsRoutes "masjidku_backend/internals/features/lembaga/stats/semester_stats/route"
	AcademicYearRoutes "masjidku_backend/internals/features/school/academics/academic_terms/route"
	ClassBooksRoutes "masjidku_backend/internals/features/school/academics/books/route"
	CertificateRoutes "masjidku_backend/internals/features/school/academics/certificates/route"
	RoomsRoutes "masjidku_backend/internals/features/school/academics/rooms/route"
	SubjectRoutes "masjidku_backend/internals/features/school/academics/subjects/route"
	ClassAttendanceSessionsRoutes "masjidku_backend/internals/features/school/classes/class_attendance_sessions/route"
	EventRoutes "masjidku_backend/internals/features/school/classes/class_events/route"
	ScheduleRoutes "masjidku_backend/internals/features/school/classes/class_schedules/route"
	ClassSectionsRoutes "masjidku_backend/internals/features/school/classes/class_sections/route"
	ClassesRoutes "masjidku_backend/internals/features/school/classes/classes/route"
	AttendanceSettingsRoute "masjidku_backend/internals/features/school/others/assesments_settings/route"
	PostRoutes "masjidku_backend/internals/features/school/others/post/route"
	AssessmentsRoutes "masjidku_backend/internals/features/school/submissions_assesments/assesments/route"
	QuizzesRoutes "masjidku_backend/internals/features/school/submissions_assesments/quizzes/route"
	SubmissionsRoutes "masjidku_backend/internals/features/school/submissions_assesments/submissions/route"

	CSSTRoutes "masjidku_backend/internals/features/school/classes/class_section_subject_teachers/route"

	// Tambahkan import route lain di sini saat modul siap:
	// SectionRoutes "masjidku_backend/internals/features/lembaga/sections/main/route"
	// StudentRoutes "masjidku_backend/internals/features/lembaga/students/main/route"
	// TeacherRoutes "masjidku_backend/internals/features/lembaga/teachers/main/route"
	// ScheduleRoutes "masjidku_backend/internals/features/lembaga/schedules/main/route"
	// FinanceRoutes  "masjidku_backend/internals/features/lembaga/finance/main/route"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

/* ===================== PUBLIC ===================== */
// Endpoint publik (boleh diakses tanpa login, atau pakai SecondAuth untuk optional user)
func LembagaPublicRoutes(r fiber.Router, db *gorm.DB) {
	ClassesRoutes.AllClassRoutes(r, db)
	// ClassSectionsRoutes.ClassSectionAllRoutes(r, db)
	LembagaStatsRoutes.AllLembagaStatsRoutes(r, db)
	YayasanRoutes.AllYayasanRoutes(r, db)
	EventRoutes.AllEventRoutes(r, db)
	RoomsRoutes.AllRoomsRoutes(r, db)
	AcademicYearRoutes.AllAcademicTermsRoutes(r, db)
	ClassBooksRoutes.AllClassBooksRoutes(r, db)
	SubjectRoutes.AllSubjectRoutes(r, db)
	ClassSectionsRoutes.AllClassSectionRoutes(r, db)
	CSSTRoutes.AllCSSTRoutes(r, db)
	// Classes (public read)
	// ClassRoutes.ClassPublicRoutes(r, db)
	// Tambahkan modul lain (public) di sini:
	// SectionRoutes.SectionPublicRoutes(r, db)
	// ScheduleRoutes.SchedulePublicRoutes(r, db)
}

/* ===================== USER (PRIVATE) ===================== */
// Endpoint yang butuh login user biasa (token user)
func LembagaUserRoutes(r fiber.Router, db *gorm.DB) {
	ClassesRoutes.ClassUserRoutes(r, db)
	ClassAttendanceSessionsRoutes.AttendanceSessionsUserRoutes(r, db)
	PostRoutes.PostUserRoutes(r, db)
	SemesterStatsRoutes.UserClassAttendanceSemesterUserRoutes(r, db)
	ClassSectionsRoutes.ClassSectionUserRoutes(r, db)
	ClassAttendanceSessionsRoutes.AttendanceSessionsTeacherRoutes(r, db)
	ScheduleRoutes.ScheduleUserRoutes(r, db)

	CertificateRoutes.CertificateUserRoutes(r, db)
	AssessmentsRoutes.AssessmentUserRoutes(r, db)
	AssessmentsRoutes.AssessmentTeacherRoutes(r, db)
	SubmissionsRoutes.SubmissionUserRoutes(r, db)
	QuizzesRoutes.QuizzesTeacherRoutes(r, db)
	QuizzesRoutes.QuizzesUserRoutes(r, db)
	// Classes (user actions: enroll, progress, dsb)
	// ClassRoutes.ClassUserRoutes(r, db)

	// Tambahkan modul lain (user) di sini:
	// StudentRoutes.StudentUserRoutes(r, db)
	// ScheduleRoutes.ScheduleUserRoutes(r, db)
}

/* ===================== ADMIN ===================== */
// Endpoint khusus admin lembaga/masjid (token + guard admin)
func LembagaAdminRoutes(r fiber.Router, db *gorm.DB) {
	// Classes (CRUD admin)
	ClassesRoutes.ClassAdminRoutes(r, db)
	ClassSectionsRoutes.ClassSectionAdminRoutes(r, db)
	LembagaStatsRoutes.LembagaStatsAdminRoutes(r, db)
	PostRoutes.PostAdminRoutes(r, db)
	SemesterStatsRoutes.UserClassAttendanceSemesterAdminRoutes(r, db)
	SubjectRoutes.SubjectAdminRoutes(r, db)
	ClassBooksRoutes.ClassBooksAdminRoutes(r, db)
	AttendanceSettingsRoute.ClassAttendanceSettingsAdminRoutes(r, db)
	AcademicYearRoutes.AcademicTermsAdminRoutes(r, db)
	YayasanRoutes.YayasanAdminRoutes(r, db)
	RoomsRoutes.RoomsAdminRoutes(r, db)
	ScheduleRoutes.ScheduleAdminRoutes(r, db)
	ClassAttendanceSessionsRoutes.AttendanceSessionsAdminRoutes(r, db)
	CertificateRoutes.CertificateAdminRoutes(r, db)
	AssessmentsRoutes.AssessmentAdminRoutes(r, db)
	SubmissionsRoutes.SubmissionAdminRoutes(r, db)
	QuizzesRoutes.QuizzesAdminRoutes(r, db)
	EventRoutes.EventAdminRoutes(r, db)
	CSSTRoutes.CSSTAdminRoutes(r, db)
	// Tambahkan modul lain (admin) di sini:
	// SectionRoutes.SectionAdminRoutes(r, db)
	// TeacherRoutes.TeacherAdminRoutes(r, db)
	// FinanceRoutes.FinanceAdminRoutes(r, db)
}

/* ===================== SUPER ADMIN ===================== */
// Endpoint khusus super admin (token + guard super admin)
func LembagaOwnerRoutes(r fiber.Router, db *gorm.DB) {

}
