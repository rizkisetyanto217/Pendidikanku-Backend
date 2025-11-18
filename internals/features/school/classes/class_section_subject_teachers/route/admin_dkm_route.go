// internals/features/lembaga/subjects/main/router/subjects_admin_routes.go
package router

import (
	csstController "schoolku_backend/internals/features/school/classes/class_section_subject_teachers/controller"
	schoolkuMiddleware "schoolku_backend/internals/middlewares/features"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

/*
Admin routes: full CRUD
Contoh mount: CSSTAdminRoutes(app.Group("/api/a"), db)

Final paths yang didukung:
- /api/a/class-section-subject-teachers ...
*/
func CSSTAdminRoutes(r fiber.Router, db *gorm.DB) {

	csstCtl := &csstController.ClassSectionSubjectTeacherController{DB: db}

	// Base group: admin saja (school context di-resolve via token / middleware lain)
	base := r.Group("",
		schoolkuMiddleware.IsSchoolAdmin(), // guard DKM/admin
	)

	csst := base.Group("/class-section-subject-teachers")
	csst.Post("/", csstCtl.Create)
	csst.Patch("/:id", csstCtl.Update)
	csst.Delete("/:id", csstCtl.Delete)
}
