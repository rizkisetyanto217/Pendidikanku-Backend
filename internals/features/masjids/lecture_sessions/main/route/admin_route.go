package route

import (
	"masjidku_backend/internals/features/masjids/lecture_sessions/main/controller"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// ✅ Admin Routes
func LectureSessionAdminRoutes(admin fiber.Router, db *gorm.DB) {
	lectureSessionCtrl := controller.NewLectureSessionController(db)
	userLectureSessionCtrl := controller.NewUserLectureSessionController(db)

	// 📚 Group: /lecture-sessions
	session := admin.Group("/lecture-sessions")
	session.Post("/", lectureSessionCtrl.CreateLectureSession)      // ➕ Buat sesi baru
	session.Get("/", lectureSessionCtrl.GetAllLectureSessions)      // 📄 Lihat semua sesi
	session.Get("/:id", lectureSessionCtrl.GetLectureSessionByID)   // 🔍 Detail sesi
	session.Put("/:id", lectureSessionCtrl.UpdateLectureSession)    // ✏️ Edit sesi
	session.Delete("/:id", lectureSessionCtrl.DeleteLectureSession) // ❌ Hapus sesi

	// 👥 Group: /user-lecture-sessions
	userSession := admin.Group("/user-lecture-sessions")
	userSession.Get("/", userLectureSessionCtrl.GetAllUserLectureSessions)      // 📄 Semua user sesi
	userSession.Get("/:id", userLectureSessionCtrl.GetUserLectureSessionByID)   // 🔍 Detail user sesi
	userSession.Put("/:id", userLectureSessionCtrl.UpdateUserLectureSession)    // ✏️ Edit user sesi
	userSession.Delete("/:id", userLectureSessionCtrl.DeleteUserLectureSession) // ❌ Hapus user sesi
}
