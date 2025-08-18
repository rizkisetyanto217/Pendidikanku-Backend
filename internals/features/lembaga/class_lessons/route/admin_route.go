// internals/features/lembaga/subjects/main/router/subjects_routes.go
package router

import (
	subjectsController "masjidku_backend/internals/features/lembaga/class_lessons/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// SubjectsAdminRoutes mendaftarkan endpoint subjects di bawah group yang sudah diproteksi (mis. /admin)
func ClassLessonsAdminRoutes(r fiber.Router, db *gorm.DB) {
	ctl := &subjectsController.SubjectsController{DB: db}

	g := r.Group("/subjects")
	g.Post("/", ctl.CreateSubject)      // POST   /admin/subjects
	g.Get("/", ctl.ListSubjects)        // GET    /admin/subjects?q=&is_active=&order_by=&sort=&limit=&offset=
	g.Get("/:id", ctl.GetSubject)       // GET    /admin/subjects/:id
	g.Put("/:id", ctl.UpdateSubject)    // PUT    /admin/subjects/:id
	g.Delete("/:id", ctl.DeleteSubject) // DELETE /admin/subjects/:id?force=true
}
