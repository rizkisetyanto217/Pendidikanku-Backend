// internals/features/lembaga/subjects/main/router/subjects_routes.go
package router

import (
	subjectsController "masjidku_backend/internals/features/lembaga/class_lessons/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// SubjectsAdminRoutes mendaftarkan endpoint subjects, class_subjects, dan class_section_subject_teachers
// di bawah group yang sudah diproteksi (mis. /admin)
func ClassLessonsAdminRoutes(r fiber.Router, db *gorm.DB) {
	// ---------------------------
	// SUBJECTS (master mapel)
	// ---------------------------
	subjectCtl := &subjectsController.SubjectsController{DB: db}
	subjects := r.Group("/subjects")
	subjects.Post("/", subjectCtl.CreateSubject)      // POST   /admin/subjects
	subjects.Get("/", subjectCtl.ListSubjects)        // GET    /admin/subjects?q=&is_active=&order_by=&sort=&limit=&offset=
	subjects.Get("/:id", subjectCtl.GetSubject)       // GET    /admin/subjects/:id
	subjects.Put("/:id", subjectCtl.UpdateSubject)    // PUT    /admin/subjects/:id
	subjects.Delete("/:id", subjectCtl.DeleteSubject) // DELETE /admin/subjects/:id?force=true

	// ---------------------------
	// CLASS SUBJECTS (mapel per kelas)
	// ---------------------------
	classSubjectCtl := &subjectsController.ClassSubjectController{DB: db}
	classSubjects := r.Group("/class-subjects")
	classSubjects.Post("/", classSubjectCtl.Create)      // POST   /admin/class-subjects
	classSubjects.Get("/", classSubjectCtl.List)         // GET    /admin/class-subjects?q=&is_active=&order_by=&sort=&limit=&offset=
	classSubjects.Get("/:id", classSubjectCtl.GetByID)   // GET    /admin/class-subjects/:id
	classSubjects.Put("/:id", classSubjectCtl.Update)    // PUT    /admin/class-subjects/:id
	classSubjects.Delete("/:id", classSubjectCtl.Delete) // DELETE /admin/class-subjects/:id?force=true

	// ---------------------------
	// CLASS SECTION SUBJECT TEACHERS (penugasan guru per section+subject)
	// ---------------------------
	csstCtl := &subjectsController.ClassSectionSubjectTeacherController{DB: db}
	csst := r.Group("/class-section-subject-teachers")
	csst.Post("/", csstCtl.Create)      // POST   /admin/class-section-subject-teachers
	csst.Get("/", csstCtl.List)         // GET    /admin/class-section-subject-teachers?masjid_id=...
	csst.Get("/:id", csstCtl.GetByID)   // GET    /admin/class-section-subject-teachers/:id
	csst.Put("/:id", csstCtl.Update)    // PUT    /admin/class-section-subject-teachers/:id
	csst.Delete("/:id", csstCtl.Delete) // DELETE /admin/class-section-subject-teachers/:id (soft delete)
}
