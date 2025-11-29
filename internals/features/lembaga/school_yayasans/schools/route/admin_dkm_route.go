// file: internals/features/schools/schools/route/admin_dkm_route.go
package route

import (
	"madinahsalam_backend/internals/constants"
	"madinahsalam_backend/internals/middlewares/auth"

	schoolctl "madinahsalam_backend/internals/features/lembaga/school_yayasans/schools/controller"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func SchoolAdminRoutes(admin fiber.Router, db *gorm.DB) {
	schoolCtrl := schoolctl.NewSchoolController(db, validator.New(), nil)
	profileCtrl := schoolctl.NewSchoolProfileController(db, validator.New())
	planCtrl := schoolctl.NewSchoolServicePlanController(db, validator.New())

	guard := auth.OnlyRolesSlice(
		constants.RoleErrorAdmin("aksi ini untuk admin/owner"),
		constants.AdminAndAbove,
	)

	// 🕌 MASJID (Admin/DKM)
	schools := admin.Group("/schools")

	// create school (DKM) — masih pakai body & context
	schools.Post("/", guard, schoolCtrl.CreateSchoolDKM)

	// Lebih spesifik dulu supaya tidak bentrok dengan "/:slug"
	schools.Get("/list", schoolCtrl.GetSchools) // 📄 Semua school

	// 🔑 TEACHER CODE (Admin/DKM)
	// school_id diambil dari token / active-school (ResolveSchoolIDFromContext)
	schools.Get("/teacher-code", guard, schoolCtrl.GetTeacherCode)
	schools.Patch("/teacher-code", guard, schoolCtrl.PatchTeacherCode)

	// ✏️ PATCH & DELETE SCHOOL — tetap pakai :school_id di path
	schools.Patch("/:school_id", guard, schoolCtrl.Patch)
	schools.Delete("/:school_id", guard, schoolCtrl.Delete)

	// 🏷️ MASJID PROFILE (Admin/DKM) — masih pakai :school_id di path
	profilesByID := admin.Group("/:school_id/school-profiles", guard)
	profilesByID.Post("/", profileCtrl.Create)
	profilesByID.Patch("/:id", profileCtrl.Update)
	profilesByID.Delete("/:id", profileCtrl.Delete)

	// 🧩 SERVICE PLANS (Admin/Owner)
	alias := admin.Group("/school-service-plans", guard)
	alias.Get("/", planCtrl.List)
	alias.Post("/", planCtrl.Create)
	alias.Patch("/:id", planCtrl.Patch)
	alias.Delete("/:id", planCtrl.Delete)
}
