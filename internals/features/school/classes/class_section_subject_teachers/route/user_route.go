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

	SubjectUserRoutes(app.Group("/api/u"), db)

Sehingga endpoint jadi:

	GET /api/u/:school_id/subjects/list
	GET /api/u/:school_slug/subjects/list
	dst.
*/
func AllCSSTRoutes(r fiber.Router, db *gorm.DB) {
	// Controllers

	csstCtl := &csstController.ClassSectionSubjectTeacherController{DB: db}

	// ===== Base by school_id =====
	baseByID := r.Group("/:school_id")
	csstByID := baseByID.Group("/class-section-subject-teachers")
	csstByID.Get("/list", csstCtl.List)

	// ===== Base by school_slug (beri prefix 'slug' agar tidak bentrok) =====
	baseBySlug := r.Group("/slug/:school_slug", schoolkuMiddleware.UseSchoolScope())
	csstBySlug := baseBySlug.Group("/class-section-subject-teachers")
	csstBySlug.Get("/list", csstCtl.List)

}
