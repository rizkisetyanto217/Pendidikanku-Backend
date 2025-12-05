// file: internals/features/school/materials/route/materials_teacher_route.go
package route

import (
	"madinahsalam_backend/internals/constants"
	"madinahsalam_backend/internals/middlewares/auth"

	cmctl "madinahsalam_backend/internals/features/school/class_others/class_materials/controller"


	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// MaterialsTeacherRoutes
//
// Contoh pemakaian di main router:
//
//	api := app.Group("/api", auth.JWTProtected())
//	t := api.Group("/t") // area teacher/staff
//	route.MaterialsTeacherRoutes(t, db)
//
// Hasil endpoint:
//
//	// CLASS MATERIALS (level CSST)
//	GET    /api/t/csst/:csst_id/materials
//	POST   /api/t/csst/:csst_id/materials
//	PATCH  /api/t/csst/:csst_id/materials/:material_id
//	DELETE /api/t/csst/:csst_id/materials/:material_id
//
//	// SCHOOL MATERIALS (template per school)
//	GET    /api/t/school-materials
//	POST   /api/t/school-materials
//	PATCH  /api/t/school-materials/:id
//	DELETE /api/t/school-materials/:id
func MaterialsTeacherRoutes(teacher fiber.Router, db *gorm.DB) {
	classMatCtrl := cmctl.NewClassMaterialsController(db)
	schoolMatCtrl := cmctl.NewSchoolMaterialController(db)

	/* =====================================================
	   Guard untuk endpoint yang butuh Admin/DKM
	   (class_materials create/update/delete)
	===================================================== */

	adminDkmGuard := auth.OnlyRolesSlice(
		constants.RoleErrorAdmin("aksi ini untuk admin/DKM"),
		constants.AdminAndAbove,
	)

	/* =====================================================
	   CLASS MATERIALS (CSST level)
	   - List: boleh diakses semua member school yang punya token valid
	   - POST/PATCH/DELETE: hanya Admin/DKM (via guard + controller checks)
	===================================================== */

	// Base group: /csst/:csst_id/materials
	csstMaterials := teacher.Group("/csst/:csst_id/materials")

	// List materi di 1 CSST
	csstMaterials.Get("/", classMatCtrl.List)

	// Mutating endpoints → wajib Admin/DKM
	protectedClass := csstMaterials.Group("/", adminDkmGuard)
	protectedClass.Post("/", classMatCtrl.TeacherCreate)
	protectedClass.Patch("/:material_id", classMatCtrl.TeacherUpdate)
	protectedClass.Delete("/:material_id", classMatCtrl.TeacherSoftDelete)

	/* =====================================================
	   SCHOOL MATERIALS (template per school)
	   - GET   : member school (guard ada di dalam controller via EnsureMemberSchool)
	   - POST  : DKM/Teacher/Admin (guard di dalam controller via ResolveSchoolForDKMOrTeacher)
	   - PATCH : DKM/Teacher/Admin
	   - DELETE: DKM/Teacher/Admin
	   Jadi di router ini tidak perlu guard tambahan lagi.
	===================================================== */

	schoolMaterials := teacher.Group("/school-materials")

	schoolMaterials.Get("/", schoolMatCtrl.ListSchoolMaterials)
	schoolMaterials.Post("/", schoolMatCtrl.CreateSchoolMaterial)
	schoolMaterials.Patch("/:id", schoolMatCtrl.UpdateSchoolMaterial)
	schoolMaterials.Delete("/:id", schoolMatCtrl.DeleteSchoolMaterial)
}
