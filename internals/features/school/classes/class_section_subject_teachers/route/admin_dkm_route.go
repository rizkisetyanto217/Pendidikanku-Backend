// internals/features/lembaga/subjects/main/router/subjects_admin_routes.go
package router

import (
	csstController "schoolku_backend/internals/features/school/classes/class_section_subject_teachers/controller"
	schoolkuMiddleware "schoolku_backend/internals/middlewares/features" // ⬅️ tambah ini

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

/*
Admin routes: full CRUD
Contoh mount: SubjectAdminRoutes(app.Group("/api/a"), db)

Final paths yang didukung:
- /api/a/:school_id/subjects ...
- /api/a/:school_slug/subjects ...
*/
func CSSTAdminRoutes(r fiber.Router, db *gorm.DB) {

	csstCtl := &csstController.ClassSectionSubjectTeacherController{DB: db}

	// ====== BASE: by school_id ======
	baseByID := r.Group("/:school_id",
		schoolkuMiddleware.IsSchoolAdmin(), // guard DKM/admin
	)

	csstByID := baseByID.Group("/class-section-subject-teachers")
	csstByID.Post("/", csstCtl.Create)
	csstByID.Patch("/:id", csstCtl.Update)
	csstByID.Delete("/:id", csstCtl.Delete)

	// ====== BASE: by school_slug (opsional, kalau mau dukung subdomain/slug) ======
	baseBySlug := r.Group("/:school_slug",
		schoolkuMiddleware.UseSchoolScope(),
		schoolkuMiddleware.IsSchoolAdmin(),
	)

	csstBySlug := baseBySlug.Group("/class-section-subject-teachers")
	csstBySlug.Post("/", csstCtl.Create)
	csstBySlug.Put("/:id", csstCtl.Update)
	csstBySlug.Delete("/:id", csstCtl.Delete)
}
