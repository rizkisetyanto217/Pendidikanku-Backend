// file: internals/features/users/route/user_teachers_route.go
package route

import (
	userTeachersCtl "masjidku_backend/internals/features/users/user_teachers/controller"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Mounted di group /api/u
func UserTeachersRoute(userRoute fiber.Router, db *gorm.DB) {
	v := validator.New()

	utc := userTeachersCtl.NewUserTeacherController(db, v, nil)

	// ==== USER_TEACHERS CRUD ====
	ut := userRoute.Group("/user-teachers")
	ut.Get("/me", utc.GetMe)                // GET    /api/u/user-teachers/me
	ut.Get("/list", utc.List)               // GET    /api/u/user-teachers
	ut.Post("/", utc.Create)                // POST   /api/u/user-teachers
	ut.Patch("/me", utc.PatchMe)            // PATCH  /api/u/user-teachers/me
	ut.Patch("/:id", utc.Patch)             // PATCH  /api/u/user-teachers/:id
	ut.Delete("/:id", utc.DeleteFile) // DELETE /api/u/user-teachers/:id/files

}