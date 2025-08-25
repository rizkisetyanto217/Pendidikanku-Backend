// internals/route/details/lembaga_details_routes.go
package details

import (
	// ====== Lembaga features ======
	AcademicYearRoutes "masjidku_backend/internals/features/lembaga/academics/academic_terms/route"
	AnnouncementRoutes "masjidku_backend/internals/features/lembaga/announcements/announcement/route"
	AnnouncementThemaRoutes "masjidku_backend/internals/features/lembaga/announcements/announcement_thema/route"
	ClassBooksRoutes "masjidku_backend/internals/features/lembaga/class_books/route"
	ClassLessonsRoutes "masjidku_backend/internals/features/lembaga/class_lessons/route"
	ClassAttendanceSessionsRoutes "masjidku_backend/internals/features/lembaga/class_sections/attendance_sessions/route"
	AttendanceSettingsRoute "masjidku_backend/internals/features/lembaga/class_sections/attendance_sessions_settings/route"
	ClassSectionsRoutes "masjidku_backend/internals/features/lembaga/class_sections/main/route"
	ClassesRoutes "masjidku_backend/internals/features/lembaga/classes/main/route"
	LembagaStatsRoutes "masjidku_backend/internals/features/lembaga/stats/lembaga_stats/route"
	SemesterStatsRoutes "masjidku_backend/internals/features/lembaga/stats/semester_stats/route"

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
	ClassesRoutes.ClassAllRoutes(r, db)
	ClassSectionsRoutes.ClassSectionAllRoutes(r, db)
	LembagaStatsRoutes.LembagaStatsAllRoutes(r, db)
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
	AnnouncementThemaRoutes.AnnouncementUserRoute(r, db)
	AnnouncementRoutes.AnnouncementUserRoutes(r, db)
	SemesterStatsRoutes.UserClassAttendanceSemesterUserRoutes(r, db)
	AcademicYearRoutes.AcademicYearUserRoutes(r, db)
	ClassBooksRoutes.ClassBooksUserRoutes(r, db)


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
	ClassAttendanceSessionsRoutes.AttendanceSessionsTeacherRoutes(r, db)
	LembagaStatsRoutes.LembagaStatsAdminRoutes(r, db)
	AnnouncementThemaRoutes.AnnouncementAdminRoute(r, db)
	AnnouncementRoutes.AnnouncementAdminRoutes(r, db)
	SemesterStatsRoutes.UserClassAttendanceSemesterAdminRoutes(r, db)
	ClassLessonsRoutes.ClassLessonsAdminRoutes(r, db)
	ClassBooksRoutes.ClassBooksAdminRoutes(r, db)
	AttendanceSettingsRoute.ClassAttendanceSettingsAdminRoutes(r, db)
	AcademicYearRoutes.AcademicYearAdminRoutes(r, db)

	// Tambahkan modul lain (admin) di sini:
	// SectionRoutes.SectionAdminRoutes(r, db)
	// TeacherRoutes.TeacherAdminRoutes(r, db)
	// FinanceRoutes.FinanceAdminRoutes(r, db)
}
