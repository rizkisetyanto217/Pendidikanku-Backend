// file: internals/features/finance/spp/api/controllers.go
package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	// ==== Import yang benar ====
	dto "schoolku_backend/internals/features/finance/billings/dto"
	model "schoolku_backend/internals/features/finance/billings/model"
	"schoolku_backend/internals/features/finance/general_billings/snapshot"
	helper "schoolku_backend/internals/helpers"
	helperAuth "schoolku_backend/internals/helpers/auth"
)

/* =======================================================
   BOOTSTRAP & HELPERS
======================================================= */

type Handler struct {
	DB *gorm.DB
}

func parseUUID(c *fiber.Ctx, param string) (uuid.UUID, error) {
	idStr := c.Params(param)
	return uuid.Parse(idStr)
}

func mustSchoolID(c *fiber.Ctx) (uuid.UUID, error) {
	midStr := strings.TrimSpace(c.Params("school_id"))
	if midStr == "" {
		return uuid.Nil, fmt.Errorf("school_id missing in path")
	}
	return uuid.Parse(midStr)
}

/* =======================================================
   FEE RULES (AUTHORIZED + TENANT-SCOPED)
======================================================= */

// POST /:school_id/spp/fee-rules
// POST /:school_id/spp/fee-rules
func (h *Handler) CreateFeeRule(c *fiber.Ctx) error {
	schoolID, err := mustSchoolID(c)
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "invalid school_id")
	}
	if err := helperAuth.EnsureStaffSchool(c, schoolID); err != nil {
		return err
	}

	var in dto.FeeRuleCreateDTO
	if err := c.BodyParser(&in); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "invalid json")
	}

	// selalu set dari path (abaikan body)
	in.FeeRuleSchoolID = schoolID

	// rakit model dari DTO
	m := dto.FeeRuleCreateDTOToModel(in) // -> fee.FeeRule

	// transaksi: hydrate snapshot GBK (kalau ada) lalu create
	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		// Jika ada referensi General Billing Kind, ambil snapshot dan isi ke kolom *_snapshot
		if m.FeeRuleGeneralBillingKindID != nil {
			snap, err := snapshot.ValidateAndSnapshotGBK(tx, schoolID, *m.FeeRuleGeneralBillingKindID)
			if err != nil {
				// salah tenant / tidak ditemukan / dll.
				return err
			}
			if snap != nil {
				m.FeeRuleGBKCodeSnapshot = snap.Code
				m.FeeRuleGBKNameSnapshot = snap.Name
				m.FeeRuleGBKCategorySnapshot = snap.Category
				m.FeeRuleGBKIsGlobalSnapshot = snap.IsGlobal
				m.FeeRuleGBKVisibilitySnapshot = snap.Visibility
				m.FeeRuleGBKIsRecurringSnapshot = snap.IsRecurring
				m.FeeRuleGBKRequiresMonthYearSnapshot = snap.RequiresMonthYear
				m.FeeRuleGBKRequiresOptionCodeSnapshot = snap.RequiresOptionCode
				m.FeeRuleGBKDefaultAmountIDRSnapshot = snap.DefaultAmountIDR
				m.FeeRuleGBKIsActiveSnapshot = snap.IsActive

				// opsional: jika bill_code kosong, pakai code GBK
				if strings.TrimSpace(m.FeeRuleBillCode) == "" && snap.Code != nil {
					m.FeeRuleBillCode = *snap.Code
				}
			}
		}

		// insert
		return tx.Create(&m).Error
	}); err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	return helper.JsonCreated(c, "fee rule created", dto.ToFeeRuleResponse(m))
}

// PATCH /:school_id/spp/fee-rules/:id
func (h *Handler) UpdateFeeRule(c *fiber.Ctx) error {
	schoolID, err := mustSchoolID(c)
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "invalid school_id")
	}
	if err := helperAuth.EnsureStaffSchool(c, schoolID); err != nil {
		return err
	}

	id, err := parseUUID(c, "id")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "invalid id")
	}

	var in dto.FeeRuleUpdateDTO
	if err := c.BodyParser(&in); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "invalid json")
	}

	var m model.FeeRule
	if err := h.DB.First(&m,
		"fee_rule_id = ? AND fee_rule_school_id = ? AND fee_rule_deleted_at IS NULL",
		id, schoolID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "fee_rule not found")
		}
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	// school_id tidak boleh diubah lewat update; cukup apply field lain
	dto.ApplyFeeRuleUpdate(&m, in)

	if err := h.DB.Save(&m).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}
	return helper.JsonOK(c, "fee rule updated", dto.ToFeeRuleResponse(m))
}

// =======================================================
// GENERATE STUDENT BILLS FROM BATCH (AUTHORIZED)
// =======================================================

func (h *Handler) GenerateStudentBills(c *fiber.Ctx) error {
	schoolID, err := mustSchoolID(c)
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "invalid school_id")
	}
	if err := helperAuth.EnsureStaffSchool(c, schoolID); err != nil {
		return err
	}

	var in dto.GenerateStudentBillsRequest
	if err := c.BodyParser(&in); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "invalid json")
	}

	// Normalize & guard
	in.StudentBillSchoolID = schoolID
	if in.BillBatchID == uuid.Nil {
		return helper.JsonError(c, http.StatusBadRequest, "bill_batch_id is required")
	}
	if strings.TrimSpace(in.Labeling.OptionCode) == "" {
		return helper.JsonError(c, http.StatusBadRequest, "labeling.option_code is required")
	}
	if in.Source.Type == "" {
		return helper.JsonError(c, http.StatusBadRequest, "source.type is required (section|class|students)")
	}
	switch in.AmountStrategy.Mode {
	case "fixed":
		if in.AmountStrategy.FixedAmountIDR == nil {
			return helper.JsonError(c, http.StatusBadRequest, "amount_strategy.fixed_amount_idr is required for mode=fixed")
		}
	case "rule_fallback_fixed":
		if in.AmountStrategy.PreferRule == nil || strings.TrimSpace(in.AmountStrategy.PreferRule.OptionCode) == "" {
			return helper.JsonError(c, http.StatusBadRequest, "amount_strategy.prefer_rule.option_code is required for mode=rule_fallback_fixed")
		}
		if in.AmountStrategy.PreferRule.By != "ym" && in.AmountStrategy.PreferRule.By != "term" {
			return helper.JsonError(c, http.StatusBadRequest, "amount_strategy.prefer_rule.by must be 'ym' or 'term'")
		}
	default:
		return helper.JsonError(c, http.StatusBadRequest, "amount_strategy.mode must be 'fixed' or 'rule_fallback_fixed'")
	}

	// 1) Load batch (tenant-scoped)
	var batch model.BillBatch
	if err := h.DB.First(&batch,
		"bill_batch_id = ? AND bill_batch_school_id = ? AND bill_batch_deleted_at IS NULL",
		in.BillBatchID, schoolID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "bill_batch not found")
		}
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	// 2) Resolve target students
	targetIDs, err := h.resolveTargetStudents(in)
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}
	if len(targetIDs) == 0 {
		return helper.JsonOK(c, "no target students", dto.GenerateStudentBillsResponse{
			BillBatchID: in.BillBatchID,
			Inserted:    0,
			Skipped:     0,
		})
	}

	// 3) Idempotency ringan (opsional)
	if in.IdempotencyKey != nil {
		var count int64
		if err := h.DB.Model(&model.StudentBill{}).
			Where("student_bill_batch_id = ? AND student_bill_school_id = ? AND student_bill_deleted_at IS NULL",
				in.BillBatchID, schoolID).
			Count(&count).Error; err == nil && int(count) >= len(targetIDs) {
			return helper.JsonOK(c, "already generated", dto.GenerateStudentBillsResponse{
				BillBatchID: in.BillBatchID,
				Inserted:    0,
				Skipped:     int(count),
			})
		}
	}

	// 4) Generate
	res := dto.GenerateStudentBillsResponse{BillBatchID: in.BillBatchID}
	err = h.DB.Transaction(func(tx *gorm.DB) error {
		for _, sid := range targetIDs {
			amount, err := h.resolveAmountWithContext(tx, in, batch, sid)
			if err != nil {
				return fmt.Errorf("student %s: %w", sid.String(), err)
			}
			usb := model.StudentBill{
				StudentBillBatchID:         in.BillBatchID,
				StudentBillSchoolID:        in.StudentBillSchoolID,
				StudentBillSchoolStudentID: &sid,
				StudentBillOptionCode:      &in.Labeling.OptionCode,
				StudentBillOptionLabel:     in.Labeling.OptionLabel,
				StudentBillAmountIDR:       amount,
				StudentBillStatus:          "unpaid",
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "student_bill_batch_id"}, {Name: "student_bill_school_student_id"}},
				DoNothing: true,
			}).Create(&usb).Error; err != nil {
				return err
			}
			if usb.StudentBillID != uuid.Nil {
				res.Inserted++
			} else {
				res.Skipped++
			}
		}
		return nil
	})
	if err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	return helper.JsonOK(c, "student bills generated", res)
}

/*
=======================================================

	Target resolver berbasis relasi kelas/section

=======================================================
*/
type idRow struct{ ID uuid.UUID }

func (h *Handler) resolveTargetStudents(in dto.GenerateStudentBillsRequest) ([]uuid.UUID, error) {
	switch in.Source.Type {
	case "students":
		return in.Source.SchoolStudentIDs, nil

	case "section":
		if in.Source.SectionID == nil {
			return nil, fmt.Errorf("source.section_id is required")
		}
		var rows []idRow
		q := h.DB.Raw(`
			SELECT ms.school_student_id AS id
			FROM student_class_sections scs
			JOIN school_students ms
			  ON ms.school_student_id = scs.student_class_section_school_student_id
			WHERE scs.student_class_section_school_id = ?
			  AND scs.student_class_section_section_id = ?
			  AND (COALESCE(?, false) IS FALSE
			       OR scs.student_class_section_status = 'active')
			  AND ms.school_student_deleted_at IS NULL
		`,
			in.StudentBillSchoolID,
			*in.Source.SectionID,
			in.Filters != nil && in.Filters.OnlyActiveStudents,
		).Scan(&rows)
		if q.Error != nil {
			return nil, q.Error
		}

		out := make([]uuid.UUID, 0, len(rows))
		for _, r := range rows {
			out = append(out, r.ID)
		}
		return out, nil

	case "class":
		if in.Source.ClassID == nil {
			return nil, fmt.Errorf("source.class_id is required")
		}
		var rows []idRow
		q := h.DB.Raw(`
			SELECT ms.school_student_id AS id
			FROM student_class_sections scs
			JOIN class_sections cs
			  ON cs.class_section_id = scs.student_class_section_section_id
			JOIN school_students ms
			  ON ms.school_student_id = scs.student_class_section_school_student_id
			WHERE scs.student_class_section_school_id = ?
			  AND cs.class_section_class_id = ?
			  AND (COALESCE(?, false) IS FALSE
			       OR scs.student_class_section_status = 'active')
			  AND ms.school_student_deleted_at IS NULL
		`,
			in.StudentBillSchoolID,
			*in.Source.ClassID,
			in.Filters != nil && in.Filters.OnlyActiveStudents,
		).Scan(&rows)
		if q.Error != nil {
			return nil, q.Error
		}

		out := make([]uuid.UUID, 0, len(rows))
		for _, r := range rows {
			out = append(out, r.ID)
		}
		return out, nil

	default:
		return nil, fmt.Errorf("unsupported source.type: %s", in.Source.Type)
	}
}

/*
=======================================================

	Ambil konteks aktif siswa (section_id, class_id)

=======================================================
*/
type studentContext struct {
	SectionID *uuid.UUID
	ClassID   *uuid.UUID
}

func (h *Handler) getStudentActiveContext(tx *gorm.DB, schoolID, studentID uuid.UUID) (studentContext, error) {
	var r struct {
		SectionID *uuid.UUID
		ClassID   *uuid.UUID
	}
	q := tx.Raw(`
		SELECT
		  scs.student_class_section_section_id AS section_id,
		  cs.class_section_class_id            AS class_id
		FROM student_class_sections scs
		LEFT JOIN class_sections cs
		  ON cs.class_section_id = scs.student_class_section_section_id
		WHERE scs.student_class_section_school_id = ?
		  AND scs.student_class_section_school_student_id = ?
		  AND scs.student_class_section_status = 'active'
		LIMIT 1
	`, schoolID, studentID).Scan(&r)
	if q.Error != nil {
		return studentContext{}, q.Error
	}
	return studentContext{SectionID: r.SectionID, ClassID: r.ClassID}, nil
}

/* =======================================================
   Resolve nominal (rule → fallback fixed)
======================================================= */
/* =======================================================
   Resolve nominal (rule → fallback fixed)
   - Konsisten DTO: returns int
   - Konsisten Model: ambil dari FeeRuleAmountOptions by option_code
======================================================= */

func (h *Handler) resolveAmountWithContext(
	tx *gorm.DB,
	in dto.GenerateStudentBillsRequest,
	batch model.BillBatch,
	studentID uuid.UUID,
) (int, error) {
	// 1) Mode fixed → langsung return (hindari panic saat nil)
	if strings.EqualFold(strings.TrimSpace(in.AmountStrategy.Mode), "fixed") {
		if in.AmountStrategy.FixedAmountIDR == nil {
			return 0, fmt.Errorf("fixed mode but fixed_amount_idr is nil")
		}
		return *in.AmountStrategy.FixedAmountIDR, nil
	}

	// Hanya dukung 2 mode
	if !strings.EqualFold(in.AmountStrategy.Mode, "rule_fallback_fixed") {
		return 0, fmt.Errorf("unsupported amount strategy mode: %s", in.AmountStrategy.Mode)
	}

	// 2) Ambil konteks aktif (class/section) siswa
	ctx, err := h.getStudentActiveContext(tx, in.StudentBillSchoolID, studentID)
	if err != nil {
		return 0, err
	}

	// 3) Normalisasi option_code
	if in.AmountStrategy.PreferRule == nil {
		return 0, fmt.Errorf("prefer_rule is required for rule_fallback_fixed")
	}
	optionCode := strings.ToLower(strings.TrimSpace(in.AmountStrategy.PreferRule.OptionCode))
	if optionCode == "" {
		return 0, fmt.Errorf("prefer_rule.option_code is required")
	}

	// 4) Tentukan tanggal efektif (pakai due_date batch jika ada)
	eff := time.Now()
	if batch.BillBatchDueDate != nil {
		eff = *batch.BillBatchDueDate
	}

	// 5) Base query: tenant, option_code, soft-delete, effective range
	q := tx.Model(&model.FeeRule{}).
		Where("fee_rule_school_id = ?", in.StudentBillSchoolID).
		Where("LOWER(fee_rule_option_code) = ?", optionCode).
		Where("fee_rule_deleted_at IS NULL").
		Where(
			"?::date >= COALESCE(fee_rule_effective_from, '-infinity'::date) AND ?::date <= COALESCE(fee_rule_effective_to, 'infinity'::date)",
			eff, eff,
		)

	// 6) Filter periode (term vs year-month)
	switch strings.ToLower(strings.TrimSpace(in.AmountStrategy.PreferRule.By)) {
	case "term":
		if batch.BillBatchTermID != nil {
			q = q.Where("fee_rule_term_id = ?", *batch.BillBatchTermID)
		} else {
			q = q.Where("fee_rule_term_id IS NULL AND fee_rule_year = ? AND fee_rule_month = ?", batch.BillBatchYear, batch.BillBatchMonth)
		}
	case "ym":
		q = q.Where("fee_rule_term_id IS NULL AND fee_rule_year = ? AND fee_rule_month = ?", batch.BillBatchYear, batch.BillBatchMonth)
	default:
		return 0, fmt.Errorf("unsupported prefer_rule.by: %s", in.AmountStrategy.PreferRule.By)
	}

	// 7) Rakit kondisi spesifisitas hanya bila konteks tersedia
	conds := make([]string, 0, 5)
	args := make([]any, 0, 5)

	// student: value (uuid.UUID)
	if studentID != uuid.Nil {
		conds = append(conds, "(fee_rule_scope = 'student' AND fee_rule_school_student_id = ?)")
		args = append(args, studentID)
	}

	// section: pointer (*uuid.UUID)
	if ctx.SectionID != nil && *ctx.SectionID != uuid.Nil {
		conds = append(conds, "(fee_rule_scope = 'section' AND fee_rule_section_id = ?)")
		args = append(args, *ctx.SectionID) // deref
	}

	// class: pointer (*uuid.UUID)
	if ctx.ClassID != nil && *ctx.ClassID != uuid.Nil {
		conds = append(conds, "(fee_rule_scope = 'class' AND fee_rule_class_id = ?)")
		args = append(args, *ctx.ClassID) // deref
	}

	// class_parent & tenant selalu disertakan (lebih umum)
	conds = append(conds, "(fee_rule_scope = 'class_parent' AND fee_rule_class_parent_id IS NOT NULL)")
	conds = append(conds, "(fee_rule_scope = 'tenant')")

	q = q.Where(strings.Join(conds, " OR "), args...)

	// 8) Ordering prioritas
	q = q.Order(`
		CASE fee_rule_scope
			WHEN 'student' THEN 1
			WHEN 'section' THEN 2
			WHEN 'class' THEN 3
			WHEN 'class_parent' THEN 4
			WHEN 'tenant' THEN 5
			ELSE 99
		END,
		fee_rule_is_default DESC,
		fee_rule_created_at DESC
	`).Limit(1)

	var rule model.FeeRule
	if err := q.First(&rule).Error; err != nil {
		// Kalau error bukan not found, return error
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, err
		}
		// Not found → Fallback ke fixed (jika ada)
		if in.AmountStrategy.FixedAmountIDR != nil {
			return *in.AmountStrategy.FixedAmountIDR, nil
		}
		return 0, fmt.Errorf("no matching rule and no fixed fallback for option_code=%s", optionCode)
	}

	// 9) Ambil nominal dari FeeRuleAmountOptions berdasarkan option_code
	amount, ok := findAmountFromOptions(rule.FeeRuleAmountOptions, optionCode)
	if ok {
		return amount, nil
	}

	// Opsi tidak ditemukan di array — terakhir: fallback ke fixed (jika ada)
	if in.AmountStrategy.FixedAmountIDR != nil {
		return *in.AmountStrategy.FixedAmountIDR, nil
	}

	return 0, fmt.Errorf("rule found but no amount option matched for option_code=%s", optionCode)
}

// Helper kecil: cari nominal pada slice options berdasarkan code (case-insensitive)
func findAmountFromOptions(opts []model.AmountOption, code string) (int, bool) {
	if len(opts) == 0 {
		return 0, false
	}
	lc := strings.ToLower(strings.TrimSpace(code))
	for _, it := range opts {
		if strings.ToLower(strings.TrimSpace(it.Code)) == lc {
			return it.Amount, true
		}
	}
	return 0, false
}
