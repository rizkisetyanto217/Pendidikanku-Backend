// internals/features/lembaga/subjects/main/router/subjects_user_routes.go
package router

import (
	subjectsController "madinahsalam_backend/internals/features/school/academics/subjects/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

/*
User routes: read-only (student/parent/teacher)
Contoh mount:

	SubjectUserRoutes(app.Group("/api/u"), db)

Sehingga endpoint jadi:

	GET /api/u/subjects/list
	GET /api/u/class-subjects/list
*/
func SubjectUserRoutes(r fiber.Router, db *gorm.DB) {
	// Controllers
	subjectCtl := &subjectsController.SubjectsController{DB: db}
	classSubjectCtl := &subjectsController.ClassSubjectController{DB: db}

	// Base: token-based school context (no :school_id / :school_slug di path)
	// r di sini biasanya sudah /api/u

	subjects := r.Group("/subjects")
	subjects.Get("/list", subjectCtl.List)

	classSubjects := r.Group("/class-subjects")
	classSubjects.Get("/list", classSubjectCtl.List)
}
