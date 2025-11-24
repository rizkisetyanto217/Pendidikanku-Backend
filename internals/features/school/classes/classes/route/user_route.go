// file: internals/features/lembaga/classes/user_classes/main/route/user_routes.go
package route

import (
	// Classes controller (read only)
	classCtrl "madinahsalam_backend/internals/features/school/classes/classes/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func ClassUserRoutes(r fiber.Router, db *gorm.DB) {
	// ===== Controllers =====
	classHandler := classCtrl.NewClassController(db)
	parentHandler := classCtrl.NewClassParentController(db, nil)
	enrollHandler := classCtrl.NewStudentClassEnrollmentController(db)

	// ================================
	// Classes (READ-ONLY untuk user)
	// ================================
	// Mirror admin: /classes
	classes := r.Group("/classes")
	{
		// GET /api/u/classes/list
		classes.Get("/list", classHandler.ListClasses)
	}

	// ================================
	// Class Parents (READ-ONLY untuk user)
	// ================================
	// Mirror admin: /class-parents
	classParents := r.Group("/class-parents")
	{
		// GET /api/u/class-parents/list
		classParents.Get("/list", parentHandler.List)
	}

	// ================================
	// Student Class Enrollments (USER)
	// ================================
	// Mirror admin prefix: /class-enrollments
	// Di user: bisa pakai ?student_id=me untuk "my enrollments"
	classEnrollments := r.Group("/class-enrollments")
	{
		// GET /api/u/class-enrollments/list?student_id=me
		classEnrollments.Get("/list", enrollHandler.List)

		// POST /api/u/class-enrollments/:id/join-section
		// body: { "class_section_id": "..." }
		classEnrollments.Post("/:id/join-section", enrollHandler.JoinSectionCSST)
	}
}
