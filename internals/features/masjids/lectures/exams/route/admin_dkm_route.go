package route

import (
	"masjidku_backend/internals/constants"
	examController "masjidku_backend/internals/features/masjids/lectures/exams/controller"
	authMiddleware "masjidku_backend/internals/middlewares/auth"
	masjidkuMiddleware "masjidku_backend/internals/middlewares/features"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// 🔒 Admin/DKM/Owner only
func LectureExamsAdminRoutes(router fiber.Router, db *gorm.DB) {
	// Group besar: wajib login + role admin/dkm/owner + scope masjid
	adminOrOwner := router.Group("/",
		authMiddleware.OnlyRolesSlice(
			constants.RoleErrorAdmin("mengelola ujian"),
			constants.AdminAndAbove, // admin, dkm, owner
		),
		masjidkuMiddleware.IsMasjidAdmin(), // inject masjid_id scope
	)

	examCtrl := examController.NewLectureExamController(db)
	userExamCtrl := examController.NewUserLectureExamController(db)

	// =========================
	// 📚 Lecture Exams (CRUD)
	// =========================
	exam := adminOrOwner.Group("/lecture-exams")
	exam.Post("/", examCtrl.CreateLectureExam)      // ➕ Buat ujian
	exam.Get("/", examCtrl.GetAllLectureExams)      // 📄 Lihat semua ujian
	exam.Get("/:id", examCtrl.GetLectureExamByID)   // 🔍 Detail ujian
	exam.Put("/:id", examCtrl.UpdateLectureExam)    // ✏️ Edit ujian
	exam.Delete("/:id", examCtrl.DeleteLectureExam) // ❌ Hapus ujian

	// =========================
	// 👥 User Lecture Exams (Read-only untuk admin melihat hasil)
	// =========================
	userExam := adminOrOwner.Group("/user-lecture-exams")
	userExam.Get("/", userExamCtrl.GetAllUserLectureExams)    // 📄 Lihat semua hasil user (scoped by masjid_id)
	userExam.Get("/:id", userExamCtrl.GetUserLectureExamByID) // 🔍 Detail hasil user
}
