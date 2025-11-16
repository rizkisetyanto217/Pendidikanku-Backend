// internals/features/lembaga/subjects/main/router/subjects_user_routes.go
package router

import (
	csstController "schoolku_backend/internals/features/school/classes/class_section_subject_teachers/controller"
	schoolkuMiddleware "schoolku_backend/internals/middlewares/features"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

/*
User routes: read-only (student/parent/teacher)

Contoh mount:

	CSSTUserRoutes(app.Group("/api/u"), db)

Sehingga endpoint jadi:

	GET /api/u/class-section-subject-teachers/list

Catatan:
- school diambil dari token (active_school) via UseSchoolScope.
- Kalau token tidak ada / active_school kosong → harusnya balikin ErrSchoolContextMissing dari helper.
*/
func CSSTUserRoutes(r fiber.Router, db *gorm.DB) {
	// Controller
	csstCtl := &csstController.ClassSectionSubjectTeacherController{DB: db}

	// Pure token-scope: Tidak pakai :school_id / :school_slug di path
	base := r.Group("/", schoolkuMiddleware.UseSchoolScope())
	csst := base.Group("/class-section-subject-teachers")
	csst.Get("/list", csstCtl.List)
}
