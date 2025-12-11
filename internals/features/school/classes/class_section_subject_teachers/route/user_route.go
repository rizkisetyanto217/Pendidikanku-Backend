// file: internals/features/lembaga/subjects/main/router/subjects_user_routes.go
package router

import (
	csstController "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/controller/csst"
	csstStudentController "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/controller/student_csst"
	schoolkuMiddleware "madinahsalam_backend/internals/middlewares/features"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

/*
Mount:

	CSSTUserRoutes(app.Group("/api/u"), db)

Endpoint:

	GET   /api/u/class-section-subject-teachers/list

	GET   /api/u/student-class-section-subject-teachers/list
	PATCH /api/u/student-class-section-subject-teachers/:id/student-notes
	PATCH /api/u/student-class-section-subject-teachers/:id/homeroom-notes
	PATCH /api/u/student-class-section-subject-teachers/:id/subject-teacher-notes
*/
func CSSTUserRoutes(r fiber.Router, db *gorm.DB) {
	csstCtl := &csstController.ClassSectionSubjectTeacherController{DB: db}
	stuCtl := &csstStudentController.StudentCSSTController{DB: db}

	base := r.Group("/", schoolkuMiddleware.UseSchoolScope())
	csst := base.Group("/class-section-subject-teachers")

	// Satu route saja: semua filter via query params
	csst.Get("/list", csstCtl.List)

	stu := base.Group("/student-class-section-subject-teachers")

	// Satu route saja: list mapping murid ↔ CSST, filter via query params
	stu.Get("/list", stuCtl.List)

	// ====== NOTES ENDPOINTS (PATCH) ======
	stu.Patch("/:id/student-notes", stuCtl.UpdateStudentNotes)
	stu.Patch("/:id/homeroom-notes", stuCtl.UpdateHomeroomNotes)
	stu.Patch("/:id/subject-teacher-notes", stuCtl.UpdateSubjectTeacherNotes)
}
