// file: internals/features/school/materials/route/materials_user_route.go
package route

import (
	cmctl "madinahsalam_backend/internals/features/school/classes/class_materials/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// MaterialsUserRoutes
//
// Contoh pemakaian di main router:
//
//	api := app.Group("/api", auth.JWTProtected())
//	u := api.Group("/u") // area user (student/teacher/dll, tergantung token)
//	route.MaterialsUserRoutes(u, db)
//
// Hasil endpoint:
//
//	// CLASS MATERIALS (level CSST) — konsumsi user (murid/guru):
//	GET  /api/u/csst/:csst_id/materials
//
//	// STUDENT PROGRESS (materi di level CSST):
//	GET  /api/u/student/class-material-progress
//	GET  /api/u/student/class-material-progress/by-material/:class_material_id
//	POST /api/u/student/class-material-progress/ping
func MaterialsUserRoutes(user fiber.Router, db *gorm.DB) {
	// Controller materi (class_materials)
	classMatCtrl := cmctl.NewClassMaterialsController(db)

	// Controller progress murid (student_class_material_progresses)
	progressCtrl := cmctl.NewStudentClassMaterialProgressController(db)

	/* =====================================================
	   CLASS MATERIALS (CSST level) - user area
	   - List materi dalam satu CSST (kelas x subject x guru)
	   - Controller.List sendiri sudah filter by school_id & csst_id
	===================================================== */

	csstMaterials := user.Group("/csst/:csst_id/materials")
	csstMaterials.Get("/", classMatCtrl.List)

	/* =====================================================
	   STUDENT CLASS MATERIAL PROGRESS - user (murid yang login)
	   - Semua handler di controller sudah enforce:
	     * ResolveSchoolIDFromContext
	     * ResolveStudentIDFromContext (hanya STUDENT di school tsb)
	===================================================== */

	studentProgress := user.Group("/student/class-material-progress")

	// List semua progress materi milik murid yang login (bisa difilter via query)
	// GET /api/u/student/class-material-progress?scsst_id=...&class_material_id=...&status=...
	studentProgress.Get("/", progressCtrl.ListMyClassMaterialProgress)

	// Ambil progress 1 materi spesifik milik murid yang login
	// GET /api/u/student/class-material-progress/by-material/:class_material_id
	studentProgress.Get("/by-material/:class_material_id", progressCtrl.GetMyClassMaterialProgressByMaterial)

	// Ping / upsert progress materi (article/video/pdf/dll)
	// POST /api/u/student/class-material-progress/ping
	studentProgress.Post("/ping", progressCtrl.PingMyClassMaterialProgress)
}
