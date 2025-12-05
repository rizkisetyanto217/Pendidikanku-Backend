// file: internals/features/school/materials/route/materials_admin_route.go
package route

import (
	"madinahsalam_backend/internals/constants"
	"madinahsalam_backend/internals/middlewares/auth"


	schoolMatCtl "madinahsalam_backend/internals/features/school/class_others/class_materials/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// MaterialsAdminRoutes
//
// Contoh pemakaian di main router:
//
//	api := app.Group("/api", auth.JWTProtected())
//	admin := api.Group("/a")
//	route.MaterialsAdminRoutes(admin, db)
//
// Hasil endpoint:
//
//	// CLASS MATERIALS (level CSST) — admin area
//	GET    /api/a/csst/:csst_id/materials
//	POST   /api/a/csst/:csst_id/materials
//	PATCH  /api/a/csst/:csst_id/materials/:material_id
//	DELETE /api/a/csst/:csst_id/materials/:material_id
//
//	// SCHOOL MATERIALS (template per school) — admin area
//	GET    /api/a/school-materials
//	POST   /api/a/school-materials
//	PATCH  /api/a/school-materials/:id
//	DELETE /api/a/school-materials/:id
func MaterialsAdminRoutes(admin fiber.Router, db *gorm.DB) {
	classMaterialsCtrl := schoolMatCtl.NewClassMaterialsController(db)
	schoolMaterialsCtrl := schoolMatCtl.NewSchoolMaterialController(db)

	// Guard: hanya Admin/DKM/Owner (AdminAndAbove)
	guard := auth.OnlyRolesSlice(
		constants.RoleErrorAdmin("aksi ini untuk admin/owner"),
		constants.AdminAndAbove,
	)

	/* =====================================================
	   CLASS MATERIALS (CSST level) - admin area
	===================================================== */

	// Semua endpoint di bawah ini wajib lewat guard (admin/dkm/owner)
	adminCSSTMaterials := admin.Group("/csst/:csst_id/materials", guard)

	adminCSSTMaterials.Get("/", classMaterialsCtrl.List)
	adminCSSTMaterials.Post("/", classMaterialsCtrl.TeacherCreate)
	adminCSSTMaterials.Patch("/:material_id", classMaterialsCtrl.TeacherUpdate)
	adminCSSTMaterials.Delete("/:material_id", classMaterialsCtrl.TeacherSoftDelete)

	/* =====================================================
	   SCHOOL MATERIALS (template per school) - admin area
	===================================================== */

	adminSchoolMaterials := admin.Group("/school-materials", guard)

	adminSchoolMaterials.Get("/", schoolMaterialsCtrl.ListSchoolMaterials)
	adminSchoolMaterials.Post("/", schoolMaterialsCtrl.CreateSchoolMaterial)
	adminSchoolMaterials.Patch("/:id", schoolMaterialsCtrl.UpdateSchoolMaterial)
	adminSchoolMaterials.Delete("/:id", schoolMaterialsCtrl.DeleteSchoolMaterial)
}
