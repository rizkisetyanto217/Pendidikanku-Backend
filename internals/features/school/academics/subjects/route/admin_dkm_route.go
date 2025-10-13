// internals/features/lembaga/subjects/main/router/subjects_admin_routes.go
package router

import (
	subjectsController "masjidku_backend/internals/features/school/academics/subjects/controller"
	masjidkuMiddleware "masjidku_backend/internals/middlewares/features" // ⬅️ tambah ini

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

/*
Admin routes: full CRUD
Contoh mount: SubjectAdminRoutes(app.Group("/api/a"), db)

Final paths yang didukung:
- /api/a/:masjid_id/subjects ...
- /api/a/:masjid_slug/subjects ...
*/
func SubjectAdminRoutes(r fiber.Router, db *gorm.DB) {
	subjectCtl := &subjectsController.SubjectsController{DB: db}
	classSubjectCtl := &subjectsController.ClassSubjectController{DB: db}

	// ====== BASE: by masjid_id ======
	baseByID := r.Group("/:masjid_id",
		masjidkuMiddleware.IsMasjidAdmin(), // guard DKM/admin
	)

	subjectsByID := baseByID.Group("/subjects")
	subjectsByID.Post("/", subjectCtl.Create)
	subjectsByID.Patch("/:id", subjectCtl.Patch)
	subjectsByID.Delete("/:id", subjectCtl.Delete)

	classSubjectsByID := baseByID.Group("/class-subjects")
	classSubjectsByID.Post("/", classSubjectCtl.Create)
	classSubjectsByID.Patch("/:id", classSubjectCtl.Update)
	classSubjectsByID.Delete("/:id", classSubjectCtl.Delete)

	// ====== BASE: by masjid_slug (opsional, kalau mau dukung subdomain/slug) ======
	baseBySlug := r.Group("/:masjid_slug",
		masjidkuMiddleware.UseMasjidScope(),
		masjidkuMiddleware.IsMasjidAdmin(),
	)

	subjectsBySlug := baseBySlug.Group("/subjects")
	subjectsBySlug.Post("/", subjectCtl.Create)
	subjectsBySlug.Patch("/:id", subjectCtl.Patch)
	subjectsBySlug.Delete("/:id", subjectCtl.Delete)

	classSubjectsBySlug := baseBySlug.Group("/class-subjects")
	classSubjectsBySlug.Post("/", classSubjectCtl.Create)
	classSubjectsBySlug.Put("/:id", classSubjectCtl.Update)
	classSubjectsBySlug.Delete("/:id", classSubjectCtl.Delete)

}
