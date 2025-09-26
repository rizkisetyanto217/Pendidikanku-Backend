// internals/features/lembaga/subjects/main/router/subjects_user_routes.go
package router

import (
	subjectsController "masjidku_backend/internals/features/school/academics/subject/controller"
	masjidkuMiddleware "masjidku_backend/internals/middlewares/features"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

/*
User routes: read-only (student/parent/teacher)
Contoh mount:

	SubjectUserRoutes(app.Group("/api/u"), db)

Sehingga endpoint jadi:

	GET /api/u/:masjid_id/subjects/list
	GET /api/u/:masjid_slug/subjects/list
	dst.
*/
func SubjectUserRoutes(r fiber.Router, db *gorm.DB) {
	// Controllers
	subjectCtl := &subjectsController.SubjectsController{DB: db}
	classSubjectCtl := &subjectsController.ClassSubjectController{DB: db}
	csstCtl := &subjectsController.ClassSectionSubjectTeacherController{DB: db}

	// ===== Base by masjid_id =====
	baseByID := r.Group("/:masjid_id") // set ctx masjid dari param
	// tambahkan middleware auth ringan bila perlu (mis. RequireLogin / IsMasjidMember)

	subjectsByID := baseByID.Group("/subjects")
	subjectsByID.Get("/list", subjectCtl.ListSubjects)

	classSubjectsByID := baseByID.Group("/class-subjects")
	classSubjectsByID.Get("/list", classSubjectCtl.List)

	csstByID := baseByID.Group("/class-section-subject-teachers")
	csstByID.Get("/list", csstCtl.List)

	// ===== Base by masjid_slug (opsional dukung slug/subdomain) =====
	baseBySlug := r.Group("/:masjid_slug",
		masjidkuMiddleware.UseMasjidScope(),
	)

	subjectsBySlug := baseBySlug.Group("/subjects")
	subjectsBySlug.Get("/list", subjectCtl.ListSubjects)

	classSubjectsBySlug := baseBySlug.Group("/class-subjects")
	classSubjectsBySlug.Get("/list", classSubjectCtl.List)

	csstBySlug := baseBySlug.Group("/class-section-subject-teachers")
	csstBySlug.Get("/list", csstCtl.List)
}
