package route

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	sppapi "schoolku_backend/internals/features/finance/billings/controller"
)

/*
Admin routes (CRUD & actions)
Diproteksi IsSchoolAdmin() — sesuaikan jika ada varian ByParam.
*/
func BillingsAdminRoutes(admin fiber.Router, db *gorm.DB) {
	h := &sppapi.Handler{DB: db}
	billBatch := &sppapi.BillBatchHandler{DB: db}
	studentBill := &sppapi.StudentBillHandler{DB: db}

	// Jika kamu punya resolver konteks school berbasis param, aktifkan di sini
	// contoh: ResolveSchoolContextByParam("school_id")
	grp := admin.Group(
		"")

	{
		// =========================
		// Fee Rules
		// =========================
		grp.Post("/fee-rules", h.CreateFeeRule)
		grp.Patch("/fee-rules/:id", h.UpdateFeeRule)
		// grp.Get("/fee-rules/list", h.ListFeeRules)

		// =========================
		// Bill Batches
		// =========================
		grp.Post("/bill-batches", billBatch.CreateBillBatch)
		grp.Patch("/bill-batches/:id", billBatch.UpdateBillBatch)
		// ---- Bill Batches (readonly)
		grp.Get("/bill-batches", billBatch.ListBillBatches)

		// =========================
		// Student Bills (list/detail + status actions)
		// =========================
		grp.Post("/student-bills/:id/cancel", studentBill.Cancel)

		// =========================
		// Generate Student Bills from Batch
		// =========================
		// Body request mengikuti dto.GenerateStudentBillsRequest (BillBatchID, dll)
		// Jika mau versi path-based (bill-batches/:id/generate), tinggal buat handler terpisah yang inject BillBatchID dari param.
		grp.Post("/generate", h.GenerateStudentBills)
	}
}
