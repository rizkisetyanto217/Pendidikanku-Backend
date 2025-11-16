// internals/features/lembaga/classes/user_classes/main/route/user_routes.go
package route

import (
	// Classes controller (read only)
	classCtrl "schoolku_backend/internals/features/school/classes/classes/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func ClassUserRoutes(r fiber.Router, db *gorm.DB) {
	// ===== Classes (READ-ONLY untuk user) =====
	clsParent := classCtrl.NewClassParentController(db, nil)
	cls := classCtrl.NewClassController(db)
	// Tenant-aware prefix
	classes := r.Group("/classes")
	classes.Get("/list", cls.ListClasses) // list kelas (read-only)

	// ===== Class Enrollments (khusus murid: hanya miliknya sendiri) =====
	enroll := classCtrl.NewStudentClassEnrollmentController(db)

	// Prefix: /api/u/:school_id/my/class-enrollments
	// (asumsi: router ini di-mount di /api/u)
	r.Get("/my/class-enrollments", enroll.ListMy)

	classParent := r.Group("/class-parents")
	classParent.Get("/list", clsParent.List)
}
