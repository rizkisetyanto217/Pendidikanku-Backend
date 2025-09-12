// internals/features/lembaga/subjects/main/router/subjects_admin_routes.go
package router

import (
	subjectsController "masjidku_backend/internals/features/school/subject_books/subject/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

/*
Admin routes: full CRUD
Mount contoh: ClassLessonsAdminRoutes(app.Group("/admin"), db)
*/
func SubjectAdminRoutes(r fiber.Router, db *gorm.DB) {
	// SUBJECTS (master mapel)
	subjectCtl := &subjectsController.SubjectsController{DB: db}
	subjects := r.Group("/subjects")
	subjects.Post("/", subjectCtl.CreateSubject)      // POST   /admin/subjects
	subjects.Put("/:id", subjectCtl.UpdateSubject)    // PUT    /admin/subjects/:id
	subjects.Delete("/:id", subjectCtl.DeleteSubject) // DELETE /admin/subjects/:id?force=true

	// CLASS SUBJECTS (mapel per kelas)
	classSubjectCtl := &subjectsController.ClassSubjectController{DB: db}
	classSubjects := r.Group("/class-subjects")
	classSubjects.Post("/", classSubjectCtl.Create)      // POST   /admin/class-subjects
	classSubjects.Put("/:id", classSubjectCtl.Update)    // PUT    /admin/class-subjects/:id
	classSubjects.Delete("/:id", classSubjectCtl.Delete) // DELETE /admin/class-subjects/:id?force=true

	// CLASS SECTION SUBJECT TEACHERS (penugasan guru per section+subject)
	csstCtl := &subjectsController.ClassSectionSubjectTeacherController{DB: db}
	csst := r.Group("/class-section-subject-teachers")
	csst.Post("/", csstCtl.Create)      // POST   /admin/class-section-subject-teachers
	csst.Put("/:id", csstCtl.Update)    // PUT    /admin/class-section-subject-teachers/:id
	csst.Delete("/:id", csstCtl.Delete) // DELETE /admin/class-section-subject-teachers/:id (soft delete)
}
