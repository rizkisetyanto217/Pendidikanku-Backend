// internals/route/details/lembaga_details_routes.go
package details

import (
	// ====== Lembaga features ======
	YayasanRoutes "schoolku_backend/internals/features/lembaga/school_yayasans/yayasans/route"
	LembagaStatsRoutes "schoolku_backend/internals/features/lembaga/stats/lembaga_stats/route"
	SemesterStatsRoutes "schoolku_backend/internals/features/lembaga/stats/semester_stats/route"
	AcademicYearRoutes "schoolku_backend/internals/features/school/academics/academic_terms/route"
	ClassBooksRoutes "schoolku_backend/internals/features/school/academics/books/route"

	// CertificateRoutes "schoolku_backend/internals/features/school/academics/certificates/route"
	RoomsRoutes "schoolku_backend/internals/features/school/academics/rooms/route"
	SubjectRoutes "schoolku_backend/internals/features/school/academics/subjects/route"
	ClassAttendanceSessionsRoutes "schoolku_backend/internals/features/school/classes/class_attendance_sessions/route"
	EventRoutes "schoolku_backend/internals/features/school/classes/class_events/route"
	ScheduleRoutes "schoolku_backend/internals/features/school/classes/class_schedules/route"
	ClassSectionsRoutes "schoolku_backend/internals/features/school/classes/class_sections/route"
	ClassesRoutes "schoolku_backend/internals/features/school/classes/classes/route"
	AttendanceSettingsRoute "schoolku_backend/internals/features/school/others/assesments_settings/route"
	AssessmentsRoutes "schoolku_backend/internals/features/school/submissions_assesments/assesments/route"
	QuizzesRoutes "schoolku_backend/internals/features/school/submissions_assesments/quizzes/route"
	SubmissionsRoutes "schoolku_backend/internals/features/school/submissions_assesments/submissions/route"

	SchoolRoutes "schoolku_backend/internals/features/lembaga/school_yayasans/schools/route"

	CSSTRoutes "schoolku_backend/internals/features/school/classes/class_section_subject_teachers/route"

	// Tambahkan import route lain di sini saat modul siap:
	// SectionRoutes "schoolku_backend/internals/features/lembaga/sections/main/route"
	// StudentRoutes "schoolku_backend/internals/features/lembaga/students/main/route"
	// TeacherRoutes "schoolku_backend/internals/features/lembaga/teachers/main/route"
	// ScheduleRoutes "schoolku_backend/internals/features/lembaga/schedules/main/route"
	// FinanceRoutes  "schoolku_backend/internals/features/lembaga/finance/main/route"

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
	ScheduleRoutes.AllScheduleRoutes(r, db)

	SchoolRoutes.AllSchoolRoutes(r, db)
}

/* ===================== USER (PRIVATE) ===================== */
// Endpoint yang butuh login user biasa (token user)
func LembagaUserRoutes(r fiber.Router, db *gorm.DB) {
	ClassesRoutes.ClassUserRoutes(r, db)
	ClassAttendanceSessionsRoutes.AttendanceSessionsUserRoutes(r, db)

	SemesterStatsRoutes.UserClassAttendanceSemesterUserRoutes(r, db)
	ClassSectionsRoutes.ClassSectionUserRoutes(r, db)
	ClassAttendanceSessionsRoutes.AttendanceSessionsTeacherRoutes(r, db)

	// CertificateRoutes.CertificateUserRoutes(r, db)
	AssessmentsRoutes.AssessmentUserRoutes(r, db)
	AssessmentsRoutes.AssessmentTeacherRoutes(r, db)
	SubmissionsRoutes.SubmissionUserRoutes(r, db)
	QuizzesRoutes.QuizzesTeacherRoutes(r, db)
	QuizzesRoutes.QuizzesUserRoutes(r, db)
	AcademicYearRoutes.AcademicUserTermsRoutes(r, db)
	ClassBooksRoutes.ClassBooksUserRoutes(r, db)

	SubjectRoutes.SubjectUserRoutes(r, db)
	RoomsRoutes.RoomsUserRoutes(r, db)
	CSSTRoutes.CSSTUserRoutes(r, db)

	ScheduleRoutes.ScheduleUserRoutes(r, db)
	// CertificateRoutes.CertificateUserRoutes(r, db)
	// CertificateRoutes.CertificateTeacherRoutes(r, db)
}

/* ===================== ADMIN ===================== */
// Endpoint khusus admin lembaga/school (token + guard admin)
func LembagaAdminRoutes(r fiber.Router, db *gorm.DB) {
	// Classes (CRUD admin)
	ClassesRoutes.ClassAdminRoutes(r, db)
	ClassSectionsRoutes.ClassSectionAdminRoutes(r, db)
	LembagaStatsRoutes.LembagaStatsAdminRoutes(r, db)
	SemesterStatsRoutes.UserClassAttendanceSemesterAdminRoutes(r, db)
	SubjectRoutes.SubjectAdminRoutes(r, db)
	ClassBooksRoutes.ClassBooksAdminRoutes(r, db)
	AttendanceSettingsRoute.ClassAttendanceSettingsAdminRoutes(r, db)
	AcademicYearRoutes.AcademicTermsAdminRoutes(r, db)
	YayasanRoutes.YayasanAdminRoutes(r, db)
	RoomsRoutes.RoomsAdminRoutes(r, db)
	ScheduleRoutes.ScheduleAdminRoutes(r, db)
	ClassAttendanceSessionsRoutes.AttendanceSessionsAdminRoutes(r, db)
	// CertificateRoutes.CertificateAdminRoutes(r, db)
	AssessmentsRoutes.AssessmentAdminRoutes(r, db)
	SubmissionsRoutes.SubmissionAdminRoutes(r, db)
	QuizzesRoutes.QuizzesAdminRoutes(r, db)
	EventRoutes.EventAdminRoutes(r, db)
	CSSTRoutes.CSSTAdminRoutes(r, db)
	// Tambahkan modul lain (admin) di sini:
	// SectionRoutes.SectionAdminRoutes(r, db)
	// TeacherRoutes.TeacherAdminRoutes(r, db)
	// FinanceRoutes.FinanceAdminRoutes(r, db)
	SchoolRoutes.SchoolAdminRoutes(r, db)
}

/* ===================== SUPER ADMIN ===================== */
// Endpoint khusus super admin (token + guard super admin)
func LembagaOwnerRoutes(r fiber.Router, db *gorm.DB) {
	SchoolRoutes.SchoolOwnerRoutes(r, db)
}
