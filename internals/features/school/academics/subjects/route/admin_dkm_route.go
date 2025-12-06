// internals/features/lembaga/subjects/main/router/subjects_admin_routes.go
package router

import (
	classSubjectController "madinahsalam_backend/internals/features/school/academics/subjects/controller/class_subjects"
	subjectsController "madinahsalam_backend/internals/features/school/academics/subjects/controller/subjects"
	schoolkuMiddleware "madinahsalam_backend/internals/middlewares/features"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

/*
Admin routes: full CRUD
Contoh mount: SubjectAdminRoutes(app.Group("/api/a"), db)

Final paths yang didukung (tanpa :school_id / :school_slug):
- /api/a/subjects ...
- /api/a/class-subjects ...
*/
func SubjectAdminRoutes(r fiber.Router, db *gorm.DB) {
	subjectCtl := &subjectsController.SubjectsController{DB: db}
	classSubjectCtl := &classSubjectController.ClassSubjectController{DB: db}

	// Base group: guard DKM/admin, school context diambil via helper (token/context),
	// bukan dari path.
	base := r.Group("", schoolkuMiddleware.IsSchoolAdmin())

	// ---------- SUBJECTS ----------
	subjects := base.Group("/subjects")
	subjects.Post("/", subjectCtl.Create)
	subjects.Patch("/:id", subjectCtl.Patch)
	subjects.Delete("/:id", subjectCtl.Delete)

	// ---------- CLASS SUBJECTS ----------
	classSubjects := base.Group("/class-subjects")
	classSubjects.Post("/", classSubjectCtl.Create)
	classSubjects.Patch("/:id", classSubjectCtl.Update)
	classSubjects.Delete("/:id", classSubjectCtl.Delete)
}
