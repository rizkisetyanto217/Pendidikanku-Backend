// internals/features/lembaga/subjects/main/router/subjects_user_routes.go
package router

import (
	subjectsController "masjidku_backend/internals/features/school/subject_books/subject/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

/*
User routes: read-only
Mount contoh: ClassLessonsUserRoutes(app.Group("/user"), db)
— cocok untuk student/parent/teacher non-admin.
*/
func SubjectUserRoutes(r fiber.Router, db *gorm.DB) {
	// SUBJECTS (master mapel) — read-only
	subjectCtl := &subjectsController.SubjectsController{DB: db}
	subjects := r.Group("/subjects")
	subjects.Get("/", subjectCtl.ListSubjects)  // GET /user/subjects?q=&is_active=&...
	subjects.Get("/:id", subjectCtl.GetSubject) // GET /user/subjects/:id

	// CLASS SUBJECTS (mapel per kelas) — read-only
	classSubjectCtl := &subjectsController.ClassSubjectController{DB: db}
	classSubjects := r.Group("/class-subjects")
	classSubjects.Get("/", classSubjectCtl.List)       // GET /user/class-subjects?include=books,teachers...
	classSubjects.Get("/:id", classSubjectCtl.GetByID) // GET /user/class-subjects/:id

	// CLASS SECTION SUBJECT TEACHERS — read-only
	csstCtl := &subjectsController.ClassSectionSubjectTeacherController{DB: db}
	csst := r.Group("/class-section-subject-teachers")
	csst.Get("/", csstCtl.List)       // GET /user/class-section-subject-teachers?section_id=&subject_id=&...
	csst.Get("/:id", csstCtl.GetByID) // GET /user/class-section-subject-teachers/:id
}