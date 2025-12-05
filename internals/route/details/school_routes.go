// internals/route/details/lembaga_details_routes.go
package details

import (
	// ====== Lembaga features ======
	YayasanRoutes "madinahsalam_backend/internals/features/lembaga/school_yayasans/yayasans/route"
	LembagaStatsRoutes "madinahsalam_backend/internals/features/lembaga/stats/lembaga_stats/route"
	SemesterStatsRoutes "madinahsalam_backend/internals/features/lembaga/stats/semester_stats/route"
	AcademicYearRoutes "madinahsalam_backend/internals/features/school/academics/academic_terms/route"
	ClassBooksRoutes "madinahsalam_backend/internals/features/school/academics/books/route"

	// CertificateRoutes "madinahsalam_backend/internals/features/school/academics/certificates/route"
	RoomsRoutes "madinahsalam_backend/internals/features/school/academics/rooms/route"
	SubjectRoutes "madinahsalam_backend/internals/features/school/academics/subjects/route"
	ClassAttendanceSessionsRoutes "madinahsalam_backend/internals/features/school/class_others/class_attendance_sessions/route"
	EventRoutes "madinahsalam_backend/internals/features/school/class_others/class_events/route"
	ScheduleRoutes "madinahsalam_backend/internals/features/school/class_others/class_schedules/route"
	ClassSectionsRoutes "madinahsalam_backend/internals/features/school/classes/class_sections/route"
	ClassesRoutes "madinahsalam_backend/internals/features/school/classes/classes/route"
	AttendanceSettingsRoute "madinahsalam_backend/internals/features/school/others/assesments_settings/route"
	AssessmentsRoutes "madinahsalam_backend/internals/features/school/submissions_assesments/assesments/route"
	QuizzesRoutes "madinahsalam_backend/internals/features/school/submissions_assesments/quizzes/route"
	SubmissionsRoutes "madinahsalam_backend/internals/features/school/submissions_assesments/submissions/route"

	CSSTRoutes "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/route"

	ClassParentRoutes "madinahsalam_backend/internals/features/school/classes/class_parents/route"

	// Tambahkan import route lain di sini saat modul siap:
	// SectionRoutes "madinahsalam_backend/internals/features/lembaga/sections/main/route"
	// StudentRoutes "madinahsalam_backend/internals/features/lembaga/students/main/route"
	// TeacherRoutes "madinahsalam_backend/internals/features/lembaga/teachers/main/route"
	// ScheduleRoutes "madinahsalam_backend/internals/features/lembaga/schedules/main/route"
	// FinanceRoutes  "madinahsalam_backend/internals/features/lembaga/finance/main/route"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

/* ===================== PUBLIC ===================== */
// Endpoint publik (boleh diakses tanpa login, atau pakai SecondAuth untuk optional user)
func SchoolPublicRoutes(r fiber.Router, db *gorm.DB) {
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
	ClassParentRoutes.AllClassParentRoutes(r, db)
	ScheduleRoutes.AllScheduleRoutes(r, db)

}

/* ===================== USER (PRIVATE) ===================== */
// Endpoint yang butuh login user biasa (token user)
func SchoolUserRoutes(r fiber.Router, db *gorm.DB) {
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

	ClassParentRoutes.ClassParentUserRoutes(r, db)
}

/* ===================== ADMIN ===================== */
// Endpoint khusus admin lembaga/school (token + guard admin)
func SchoolAdminRoutes(r fiber.Router, db *gorm.DB) {
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

	ClassParentRoutes.ClassParentAdminRoutes(r, db)
}

/* ===================== SUPER ADMIN ===================== */
// Endpoint khusus super admin (token + guard super admin)
func SchoolOwnerRoutes(r fiber.Router, db *gorm.DB) {

}
