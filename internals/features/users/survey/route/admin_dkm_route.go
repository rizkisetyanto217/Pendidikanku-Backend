package route

import (
	"masjidku_backend/internals/constants"
	surveyController "masjidku_backend/internals/features/users/survey/controller"
	authMiddleware "masjidku_backend/internals/middlewares/auth"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func SurveyAdminRoutes(api fiber.Router, db *gorm.DB) {
	surveyQuestionCtrl := surveyController.NewSurveyQuestionController(db)

	// ✅ Admin (admin/dkm) & Owner boleh kelola pertanyaan survei
	adminOrOwner := api.Group("/survey-questions",
		authMiddleware.OnlyRolesSlice(
			constants.RoleErrorAdmin("mengelola pertanyaan survei"),
			constants.AdminAndAbove,
		),
	)

	adminOrOwner.Post("/",   surveyQuestionCtrl.Create)
	adminOrOwner.Put("/:id", surveyQuestionCtrl.Update)
	adminOrOwner.Delete("/:id", surveyQuestionCtrl.Delete)

	// (Opsional) Sub-group khusus owner untuk aksi sangat sensitif
	// ownerOnly := api.Group("/survey-questions",
	// 	authMiddleware.OnlyRolesSlice(
	// 		constants.RoleErrorOwner("aksi ini khusus owner"),
	// 		constants.OwnerOnly,
	// 	),
	// )
	// ownerOnly.Post("/import", surveyQuestionCtrl.BulkImport)
}
