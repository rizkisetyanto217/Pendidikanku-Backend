// file: internals/features/finance/spp/api/bill_batch_controller.go
package controller

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	// ganti sesuai struktur projectmu
	"masjidku_backend/internals/features/finance/billings/dto"
	billing "masjidku_backend/internals/features/finance/billings/model"
	helper "masjidku_backend/internals/helpers"
)

// =======================================================
// BOOTSTRAP
// =======================================================

type BillBatchHandler struct {
	DB *gorm.DB
}

// =======================================================
// HELPERS
// =======================================================

func parseUUIDParam(c *fiber.Ctx, name string) (uuid.UUID, error) {
	idStr := c.Params(name)
	return uuid.Parse(idStr)
}

func xorValid(classID, sectionID *uuid.UUID) bool {
	return (classID != nil && sectionID == nil) || (classID == nil && sectionID != nil)
}

func isUniqueViolation(err error) bool {
	return err != nil &&
		(strings.Contains(err.Error(), "duplicate key value") ||
			strings.Contains(err.Error(), "unique constraint"))
}

func isOneOff(optionCode *string) bool {
	return optionCode != nil && strings.TrimSpace(*optionCode) != ""
}

func normalizeBillCode(code string) string {
	code = strings.TrimSpace(code)
	if code == "" {
		return "SPP"
	}
	return code
}

func strPtrOrNil(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

// =======================================================
// INTERNAL: Recalc totals (tanpa trigger DB)
// =======================================================

func recalcBillBatchTotals(tx *gorm.DB, batchID uuid.UUID) error {
	type agg struct {
		TotalAmount  int64
		TotalPaid    int64
		TotalStu     int64
		TotalStuPaid int64
	}
	var a agg

	if err := tx.Table("student_bills").
		Select(`
			COALESCE(SUM(student_bill_amount_idr), 0) AS total_amount,
			COALESCE(SUM(CASE WHEN student_bill_status = 'paid' THEN student_bill_amount_idr ELSE 0 END), 0) AS total_paid,
			COALESCE(COUNT(1), 0) AS total_stu,
			COALESCE(SUM(CASE WHEN student_bill_status = 'paid' THEN 1 ELSE 0 END), 0) AS total_stu_paid
		`).
		Where("student_bill_batch_id = ? AND student_bill_deleted_at IS NULL", batchID).
		Scan(&a).Error; err != nil {
		return err
	}

	return tx.Model(&billing.BillBatch{}).
		Where("bill_batch_id = ?", batchID).
		Updates(map[string]any{
			"bill_batch_total_amount_idr":    int(a.TotalAmount),
			"bill_batch_total_paid_idr":      int(a.TotalPaid),
			"bill_batch_total_students":      int(a.TotalStu),
			"bill_batch_total_students_paid": int(a.TotalStuPaid),
			"bill_batch_updated_at":          time.Now(),
		}).Error
}

// =======================================================
// INTERNAL: resolve target students (subset / scope)
// =======================================================

func (h *BillBatchHandler) listTargetStudentIDs(tx *gorm.DB, masjidID uuid.UUID, classID, sectionID *uuid.UUID, selected []uuid.UUID, onlyActive bool) ([]uuid.UUID, error) {
	// Jika admin sudah memilih subset siswa → validasi & gunakan itu saja
	if len(selected) > 0 {
		type row struct{ ID uuid.UUID }
		var rows []row
		q := tx.Table("masjid_students").
			Select("masjid_student_id AS id").
			Where("masjid_student_masjid_id = ?", masjidID).
			Where("masjid_student_id IN ?", selected)
		if onlyActive {
			q = q.Where("is_active = TRUE")
		}
		if err := q.Find(&rows).Error; err != nil {
			return nil, err
		}
		out := make([]uuid.UUID, 0, len(rows))
		for _, r := range rows {
			out = append(out, r.ID)
		}
		return out, nil
	}

	// Jika tidak, ambil semua siswa pada scope (class/section)
	type row struct{ ID uuid.UUID }
	var rows []row
	q := tx.Table("masjid_students").
		Select("masjid_student_id AS id").
		Where("masjid_student_masjid_id = ?", masjidID)

	if classID != nil {
		q = q.Where("class_id = ?", *classID)
	}
	if sectionID != nil {
		q = q.Where("section_id = ?", *sectionID)
	}
	if onlyActive {
		q = q.Where("is_active = TRUE")
	}
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]uuid.UUID, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.ID)
	}
	return out, nil
}

// =======================================================
// INTERNAL: resolve nominal dari fee_rules (spesifisitas & periode)
// =======================================================

func (h *BillBatchHandler) resolveAmountFromRules(tx *gorm.DB, masjidID uuid.UUID, optionCode string, batch billing.BillBatch, studentID uuid.UUID) (int, error) {
	eff := time.Now()
	if batch.BillBatchDueDate != nil {
		eff = *batch.BillBatchDueDate
	}

	q := tx.Model(&billing.FeeRule{}).
		Where("fee_rule_masjid_id = ?", masjidID).
		Where("LOWER(fee_rule_option_code) = ?", strings.ToLower(optionCode)).
		Where("fee_rule_deleted_at IS NULL").
		Where("?::date >= COALESCE(fee_rule_effective_from, '-infinity'::date) AND ?::date <= COALESCE(fee_rule_effective_to, 'infinity'::date)", eff, eff)

	// match periode:
	// - Jika term ada ⇒ match by term
	// - Jika YM ada ⇒ match by YM
	// - Jika one-off tanpa YM ⇒ cari rule general (term NULL, year NULL, month NULL) jika skemamu mendukung
	if batch.BillBatchTermID != nil {
		q = q.Where("fee_rule_term_id = ?", *batch.BillBatchTermID)
	} else if batch.BillBatchYear != nil && batch.BillBatchMonth != nil {
		q = q.Where("fee_rule_term_id IS NULL AND fee_rule_year = ? AND fee_rule_month = ?", *batch.BillBatchYear, *batch.BillBatchMonth)
	} else {
		q = q.Where("fee_rule_term_id IS NULL AND fee_rule_year IS NULL AND fee_rule_month IS NULL")
	}

	// pilih rule paling spesifik
	var rule billing.FeeRule
	err := q.Where(`
		(fee_rule_scope = 'student' AND fee_rule_masjid_student_id = ?)
		OR (fee_rule_scope = 'section' AND fee_rule_section_id = (SELECT section_id FROM masjid_students WHERE masjid_student_id = ? LIMIT 1))
		OR (fee_rule_scope = 'class'   AND fee_rule_class_id   = (SELECT class_id   FROM masjid_students WHERE masjid_student_id = ? LIMIT 1))
		OR (fee_rule_scope = 'class_parent' AND fee_rule_class_parent_id IS NOT NULL)
		OR (fee_rule_scope = 'tenant')
	`, studentID, studentID, studentID).
		Order(`
			CASE fee_rule_scope
				WHEN 'student' THEN 1
				WHEN 'section' THEN 2
				WHEN 'class' THEN 3
				WHEN 'class_parent' THEN 4
				WHEN 'tenant' THEN 5
				ELSE 99
			END, fee_rule_is_default DESC, fee_rule_created_at DESC
		`).
		Limit(1).
		First(&rule).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, fmt.Errorf("no matching fee_rule for option_code=%s", optionCode)
		}
		return 0, err
	}
	return rule.FeeRuleAmountIDR, nil
}

// =======================================================
// CREATE (hanya buat batch; masjid_id dari path)
// =======================================================

func (h *BillBatchHandler) CreateBillBatch(c *fiber.Ctx) error {
	masjidID, err := mustMasjidID(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid masjid_id")
	}

	var in dto.BillBatchCreateDTO
	if err := c.BodyParser(&in); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid json")
	}

	// override dari path
	in.BillBatchMasjidID = masjidID
	in.BillBatchBillCode = normalizeBillCode(in.BillBatchBillCode)
	in.BillBatchOptionCode = strPtrOrNil(in.BillBatchOptionCode)

	// XOR guard
	if !xorValid(in.BillBatchClassID, in.BillBatchSectionID) {
		return helper.JsonError(c, fiber.StatusBadRequest, "exactly one of bill_batch_class_id or bill_batch_section_id must be set")
	}

	// Periodic vs One-off validation:
	if isOneOff(in.BillBatchOptionCode) {
		// one-off: YM opsional (boleh nil)
	} else {
		// periodic: YM wajib
		if in.BillBatchMonth == nil || in.BillBatchYear == nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "periodic batch requires bill_batch_month and bill_batch_year")
		}
	}

	m := dto.BillBatchCreateDTOToModel(in)

	if err := h.DB.Create(&m).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "duplicate bill batch for the given scope and period")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonCreated(c, "bill batch created", dto.ToBillBatchResponse(m))
}

// =======================================================
// UPDATE (partial; tenant-guard)
// =======================================================

func (h *BillBatchHandler) UpdateBillBatch(c *fiber.Ctx) error {
	masjidID, err := mustMasjidID(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid masjid_id")
	}

	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid id")
	}

	var in dto.BillBatchUpdateDTO
	if err := c.BodyParser(&in); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid json")
	}

	var m billing.BillBatch
	if err := h.DB.First(&m, "bill_batch_id = ? AND bill_batch_masjid_id = ? AND bill_batch_deleted_at IS NULL", id, masjidID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "bill batch not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	if err := dto.ApplyBillBatchUpdate(&m, in); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Re-validate periodic vs one-off setelah apply:
	if isOneOff(m.BillBatchOptionCode) {
		// one-off: YM opsional (no-op)
	} else {
		// periodic: YM wajib & valid
		if m.BillBatchMonth == nil || m.BillBatchYear == nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "periodic batch requires bill_batch_month and bill_batch_year")
		}
	}

	if err := h.DB.Save(&m).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "duplicate bill batch for the given scope and period")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonOK(c, "bill batch updated", dto.ToBillBatchResponse(m))
}

// =======================================================
// DELETE (soft delete; tenant-scoped)
// =======================================================

func (h *BillBatchHandler) DeleteBillBatch(c *fiber.Ctx) error {
	masjidID, err := mustMasjidID(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid masjid_id")
	}

	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid id")
	}

	err = h.DB.Transaction(func(tx *gorm.DB) error {
		var m billing.BillBatch
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&m, "bill_batch_id = ? AND bill_batch_masjid_id = ? AND bill_batch_deleted_at IS NULL", id, masjidID).Error; err != nil {
			return err
		}
		now := time.Now()
		m.BillBatchDeletedAt = gorm.DeletedAt{Time: now, Valid: true}
		return tx.Save(&m).Error
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "bill batch not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonDeleted(c, "bill batch deleted", fiber.Map{"bill_batch_id": id})
}

// =======================================================
// CREATE + GENERATE student_bills dari fee_rules (sekali jalan)
// POST /api/a/:masjid_id/spp/bill-batches/generate
// =======================================================

func (h *BillBatchHandler) CreateBillBatchAndGenerate(c *fiber.Ctx) error {
	masjidID, err := mustMasjidID(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid masjid_id")
	}

	var in dto.BillBatchGenerateDTO
	if err := c.BodyParser(&in); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid json")
	}
	// Normalisasi code & option
	in.BillBatchBillCode = normalizeBillCode(in.BillBatchBillCode)
	in.BillBatchOptionCode = strPtrOrNil(in.BillBatchOptionCode)

	// XOR guard
	if !xorValid(in.BillBatchClassID, in.BillBatchSectionID) {
		return helper.JsonError(c, fiber.StatusBadRequest, "exactly one of bill_batch_class_id or bill_batch_section_id must be set")
	}

	// Periodic vs One-off validation utk batch yg dibuat:
	if isOneOff(in.BillBatchOptionCode) {
		// one-off: YM opsional
	} else {
		// periodic: YM wajib
		if in.BillBatchMonth == nil || in.BillBatchYear == nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "periodic batch requires bill_batch_month and bill_batch_year")
		}
	}

	var out dto.BillBatchGenerateResponse

	err = h.DB.Transaction(func(tx *gorm.DB) error {
		// 1) Buat batch
		batch := billing.BillBatch{
			BillBatchMasjidID:             masjidID,
			BillBatchClassID:              in.BillBatchClassID,
			BillBatchSectionID:            in.BillBatchSectionID,
			BillBatchMonth:                in.BillBatchMonth,
			BillBatchYear:                 in.BillBatchYear,
			BillBatchTermID:               in.BillBatchTermID,
			BillBatchGeneralBillingKindID: in.BillBatchGeneralBillingKindID,
			BillBatchBillCode:             in.BillBatchBillCode,
			BillBatchOptionCode:           in.BillBatchOptionCode,
			BillBatchTitle:                in.BillBatchTitle,
			BillBatchDueDate:              in.BillBatchDueDate,
			BillBatchNote:                 in.BillBatchNote,
		}
		if err := tx.Create(&batch).Error; err != nil {
			if isUniqueViolation(err) {
				return helper.JsonError(c, fiber.StatusConflict, "duplicate bill batch for the given scope and period")
			}
			return err
		}

		// 2) Ambil target siswa
		targetIDs, err := h.listTargetStudentIDs(tx, masjidID, in.BillBatchClassID, in.BillBatchSectionID, in.SelectedStudentIDs, in.OnlyActiveStudents)
		if err != nil {
			return err
		}

		// 3) Generate student_bills sesuai fee_rules
		ins, skip := 0, 0
		for _, sid := range targetIDs {
			amount, err := h.resolveAmountFromRules(tx, masjidID, in.Labeling.OptionCode, batch, sid)
			if err != nil {
				// Kalau ingin strict, ganti ke: return err
				skip++
				continue
			}
			sb := billing.StudentBill{
				StudentBillBatchID:         batch.BillBatchID,
				StudentBillMasjidID:        masjidID,
				StudentBillMasjidStudentID: &sid,
				StudentBillOptionCode:      &in.Labeling.OptionCode, // labeling utk student_bills
				StudentBillOptionLabel:     in.Labeling.OptionLabel,
				StudentBillAmountIDR:       amount,
				StudentBillStatus:          "unpaid",
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "student_bill_batch_id"}, {Name: "student_bill_masjid_student_id"}},
				DoNothing: true,
			}).Create(&sb).Error; err != nil {
				return err
			}
			if sb.StudentBillID != uuid.Nil {
				ins++
			} else {
				skip++
			}
		}

		// 4) Recalc totals batch
		if err := recalcBillBatchTotals(tx, batch.BillBatchID); err != nil {
			return err
		}

		out = dto.BillBatchGenerateResponse{
			BillBatch: dto.ToBillBatchResponse(batch),
			Inserted:  ins,
			Skipped:   skip,
		}
		return nil
	})
	if err != nil {
		// helper.JsonError biasanya sudah mengatur HTTP status; jika bukan,
		// fallback 500.
		if he, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, he.Code, he.Message)
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonCreated(c, "bill batch created & student bills generated", out)
}
