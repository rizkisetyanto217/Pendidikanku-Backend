// internals/features/lembaga/subjects/main/router/subjects_user_routes.go
package router

import (
	classSubjectController "madinahsalam_backend/internals/features/school/academics/subjects/controller/class_subjects"
	subjectsController "madinahsalam_backend/internals/features/school/academics/subjects/controller/subjects"
	schoolkuMiddleware "madinahsalam_backend/internals/middlewares/features"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

/*
User routes: read-only (student/parent/teacher)
Contoh mount:

	SubjectUserRoutes(app.Group("/api/u"), db)

Sehingga endpoint jadi:

	GET /api/u/:school_id/subjects/list
	GET /api/u/:school_slug/subjects/list
	dst.
*/
func AllSubjectRoutes(r fiber.Router, db *gorm.DB) {
	// Controllers
	subjectCtl := &subjectsController.SubjectsController{DB: db}
	classSubjectCtl := &classSubjectController.ClassSubjectController{DB: db}

	// ===== Base by school_id =====
	baseByID := r.Group("/:school_id") // set ctx school dari param
	// tambahkan middleware auth ringan bila perlu (mis. RequireLogin / IsSchoolMember)

	subjectsByID := baseByID.Group("/subjects")
	subjectsByID.Get("/list", subjectCtl.List)

	classSubjectsByID := baseByID.Group("/class-subjects")
	classSubjectsByID.Get("/list", classSubjectCtl.List)

	// ===== Base by school_slug (opsional dukung slug/subdomain) =====
	baseBySlug := r.Group("/:school_slug",
		schoolkuMiddleware.UseSchoolScope(),
	)

	subjectsBySlug := baseBySlug.Group("/subjects")
	subjectsBySlug.Get("/list", subjectCtl.List)

	classSubjectsBySlug := baseBySlug.Group("/class-subjects")
	classSubjectsBySlug.Get("/list", classSubjectCtl.List)

}
