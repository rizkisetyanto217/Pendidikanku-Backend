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

	// ==== Ganti path sesuai projectmu ====
	"masjidku_backend/internals/features/finance/billings/dto"
	billing "masjidku_backend/internals/features/finance/billings/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
)

/* =======================================================
   BOOTSTRAP & HELPERS
======================================================= */

func normalizeOrderFragment(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	up := strings.ToUpper(s)
	if strings.HasPrefix(up, "ORDER BY ") {
		return strings.TrimSpace(s[9:])
	}
	return s
}

type Handler struct {
	DB *gorm.DB
}

func parseUUID(c *fiber.Ctx, param string) (uuid.UUID, error) {
	idStr := c.Params(param)
	return uuid.Parse(idStr)
}

func mustMasjidID(c *fiber.Ctx) (uuid.UUID, error) {
	midStr := strings.TrimSpace(c.Params("masjid_id"))
	if midStr == "" {
		return uuid.Nil, fmt.Errorf("masjid_id missing in path")
	}
	return uuid.Parse(midStr)
}

/* =======================================================
   FEE RULES (AUTHORIZED + TENANT-SCOPED)
======================================================= */

// POST /:masjid_id/spp/fee-rules
func (h *Handler) CreateFeeRule(c *fiber.Ctx) error {
	masjidID, err := mustMasjidID(c)
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "invalid masjid_id")
	}
	if err := helperAuth.EnsureStaffMasjid(c, masjidID); err != nil {
		return err
	}

	var in dto.FeeRuleCreateDTO
	if err := c.BodyParser(&in); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "invalid json")
	}

	// >>> selalu set dari path (abaikan body)
	in.FeeRuleMasjidID = masjidID

	model := dto.FeeRuleCreateDTOToModel(in)
	if err := h.DB.Create(&model).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}
	return helper.JsonCreated(c, "fee rule created", dto.ToFeeRuleResponse(model))
}

// PATCH /:masjid_id/spp/fee-rules/:id
func (h *Handler) UpdateFeeRule(c *fiber.Ctx) error {
	masjidID, err := mustMasjidID(c)
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "invalid masjid_id")
	}
	if err := helperAuth.EnsureStaffMasjid(c, masjidID); err != nil {
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

	var m billing.FeeRule
	if err := h.DB.First(&m,
		"fee_rule_id = ? AND fee_rule_masjid_id = ? AND fee_rule_deleted_at IS NULL",
		id, masjidID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "fee_rule not found")
		}
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	// >>> masjid_id tidak boleh diubah lewat update; cukup apply field lain
	dto.ApplyFeeRuleUpdate(&m, in)

	if err := h.DB.Save(&m).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}
	return helper.JsonOK(c, "fee rule updated", dto.ToFeeRuleResponse(m))
}

// GET /:masjid_id/spp/fee-rules
func (h *Handler) ListFeeRules(c *fiber.Ctx) error {
	masjidID, err := mustMasjidID(c)
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "invalid masjid_id")
	}
	if err := helperAuth.EnsureMemberMasjid(c, masjidID); err != nil {
		return err
	}

	p := helper.ParseFiber(c, "created_at", "desc", helper.DefaultOpts)
	offset := (p.Page - 1) * p.PerPage

	q := h.DB.Model(&billing.FeeRule{}).
		Where("fee_rule_deleted_at IS NULL").
		Where("fee_rule_masjid_id = ?", masjidID)

	if oc := c.Query("option_code"); oc != "" {
		q = q.Where("LOWER(fee_rule_option_code) = ?", strings.ToLower(oc))
	}
	if sc := c.Query("scope"); sc != "" {
		q = q.Where("fee_rule_scope = ?", sc)
	}
	if tid := c.Query("term_id"); tid != "" {
		if id, err := uuid.Parse(tid); err == nil {
			q = q.Where("fee_rule_term_id = ?", id)
		}
	} else if ym := c.Query("ym"); ym != "" {
		var y, m int
		if _, err := fmt.Sscanf(ym, "%d-%d", &y, &m); err == nil && y > 0 && m >= 1 && m <= 12 {
			q = q.Where("fee_rule_year = ? AND fee_rule_month = ?", y, m)
		}
	}

	allowed := map[string]string{
		"created_at": "fee_rule_created_at",
		"updated_at": "fee_rule_updated_at",
		"amount":     "fee_rule_amount_idr",
		"option":     "fee_rule_option_code",
	}
	orderClause, _ := p.SafeOrderClause(allowed, "created_at")
	orderClause = normalizeOrderFragment(orderClause)

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	listQ := q
	if orderClause != "" {
		listQ = listQ.Order(orderClause)
	}
	if !p.All {
		listQ = listQ.Limit(p.Limit()).Offset(offset)
	}

	var list []billing.FeeRule
	if err := listQ.Find(&list).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	out := dto.ToFeeRuleResponses(list)
	meta := helper.BuildMeta(total, p)
	return helper.JsonList(c, out, meta)
}

// =======================================================
// GENERATE STUDENT BILLS FROM BATCH (AUTHORIZED)
// =======================================================

func (h *Handler) GenerateStudentBills(c *fiber.Ctx) error {
	masjidID, err := mustMasjidID(c)
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "invalid masjid_id")
	}
	if err := helperAuth.EnsureStaffMasjid(c, masjidID); err != nil {
		return err
	}

	var in dto.GenerateStudentBillsRequest
	if err := c.BodyParser(&in); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "invalid json")
	}

	// Normalize & guard
	in.StudentBillMasjidID = masjidID
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
	var batch billing.BillBatch
	if err := h.DB.First(&batch,
		"bill_batch_id = ? AND bill_batch_masjid_id = ? AND bill_batch_deleted_at IS NULL",
		in.BillBatchID, masjidID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "bill_batch not found")
		}
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	// 2) Resolve target students
	targetIDs, err := h.resolveTargetStudents(in)
	if err != nil {
		// kembali 400 untuk kesalahan input/query yang bisa diprediksi
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
		if err := h.DB.Model(&billing.StudentBill{}).
			Where("student_bill_batch_id = ? AND student_bill_masjid_id = ? AND student_bill_deleted_at IS NULL",
				in.BillBatchID, masjidID).
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
			usb := billing.StudentBill{
				StudentBillBatchID:         in.BillBatchID,
				StudentBillMasjidID:        in.StudentBillMasjidID,
				StudentBillMasjidStudentID: &sid,
				StudentBillOptionCode:      &in.Labeling.OptionCode,
				StudentBillOptionLabel:     in.Labeling.OptionLabel,
				StudentBillAmountIDR:       amount,
				StudentBillStatus:          "unpaid",
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "student_bill_batch_id"}, {Name: "student_bill_masjid_student_id"}},
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
		// (opsional) update totals batch di sini
		return nil
	})
	if err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	return helper.JsonOK(c, "student bills generated", res)
}

/*
	=======================================================
	  Target resolver berbasis JSONB masjid_student_sections

=======================================================
*/
type idRow struct{ ID uuid.UUID }

func (h *Handler) resolveTargetStudents(in dto.GenerateStudentBillsRequest) ([]uuid.UUID, error) {
	switch in.Source.Type {
	case "students":
		return in.Source.MasjidStudentIDs, nil

	case "section":
		if in.Source.SectionID == nil {
			return nil, fmt.Errorf("source.section_id is required")
		}
		var rows []idRow
		q := h.DB.Raw(`
			SELECT ms.masjid_student_id AS id
			FROM student_class_sections scs
			JOIN masjid_students ms
			  ON ms.masjid_student_id = scs.student_class_section_masjid_student_id
			WHERE scs.student_class_section_masjid_id = ?
			  AND scs.student_class_section_section_id = ?
			  AND (COALESCE(?, false) IS FALSE
			       OR scs.student_class_section_status = 'active')
			  AND ms.masjid_student_deleted_at IS NULL
		`,
			in.StudentBillMasjidID,
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
			SELECT ms.masjid_student_id AS id
			FROM student_class_sections scs
			JOIN class_sections cs
			  ON cs.class_section_id = scs.student_class_section_section_id
			JOIN masjid_students ms
			  ON ms.masjid_student_id = scs.student_class_section_masjid_student_id
			WHERE scs.student_class_section_masjid_id = ?
			  AND cs.class_section_class_id = ?
			  AND (COALESCE(?, false) IS FALSE
			       OR scs.student_class_section_status = 'active')
			  AND ms.masjid_student_deleted_at IS NULL
		`,
			in.StudentBillMasjidID,
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

func (h *Handler) getStudentActiveContext(tx *gorm.DB, masjidID, studentID uuid.UUID) (studentContext, error) {
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
		WHERE scs.student_class_section_masjid_id = ?
		  AND scs.student_class_section_masjid_student_id = ?
		  AND scs.student_class_section_status = 'active'
		LIMIT 1
	`, masjidID, studentID).Scan(&r)
	if q.Error != nil {
		return studentContext{}, q.Error
	}
	return studentContext{SectionID: r.SectionID, ClassID: r.ClassID}, nil
}

/* =======================================================
   Resolve nominal (rule → fallback fixed)
======================================================= */

func (h *Handler) resolveAmountWithContext(tx *gorm.DB, in dto.GenerateStudentBillsRequest, batch billing.BillBatch, studentID uuid.UUID) (int, error) {
	// Mode fixed langsung
	if in.AmountStrategy.Mode == "fixed" {
		return *in.AmountStrategy.FixedAmountIDR, nil
	}

	// Mode rule_fallback_fixed
	ctx, err := h.getStudentActiveContext(tx, in.StudentBillMasjidID, studentID)
	if err != nil {
		return 0, err
	}

	optionCode := strings.ToLower(in.AmountStrategy.PreferRule.OptionCode)

	// Tanggal efektif referensi (pakai due_date batch jika ada)
	eff := time.Now()
	if batch.BillBatchDueDate != nil {
		eff = *batch.BillBatchDueDate
	}

	q := tx.Model(&billing.FeeRule{}).
		Where("fee_rule_masjid_id = ?", in.StudentBillMasjidID).
		Where("LOWER(fee_rule_option_code) = ?", optionCode).
		Where("fee_rule_deleted_at IS NULL").
		Where("?::date >= COALESCE(fee_rule_effective_from, '-infinity'::date) AND ?::date <= COALESCE(fee_rule_effective_to, 'infinity'::date)", eff, eff)

	switch in.AmountStrategy.PreferRule.By {
	case "term":
		if batch.BillBatchTermID != nil {
			q = q.Where("fee_rule_term_id = ?", *batch.BillBatchTermID)
		} else {
			q = q.Where("fee_rule_term_id IS NULL AND fee_rule_year = ? AND fee_rule_month = ?", batch.BillBatchYear, batch.BillBatchMonth)
		}
	case "ym":
		q = q.Where("fee_rule_term_id IS NULL AND fee_rule_year = ? AND fee_rule_month = ?", batch.BillBatchYear, batch.BillBatchMonth)
	}

	// Urutan spesifisitas (gunakan konteks yang ada saja)
	var rule billing.FeeRule
	q = q.Where(`
		(fee_rule_scope = 'student' AND fee_rule_masjid_student_id = ?)
		OR (fee_rule_scope = 'section' AND fee_rule_section_id = ?)
		OR (fee_rule_scope = 'class'   AND fee_rule_class_id   = ?)
		OR (fee_rule_scope = 'class_parent' AND fee_rule_class_parent_id IS NOT NULL)
		OR (fee_rule_scope = 'tenant')
	`, studentID, ctx.SectionID, ctx.ClassID).
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
		First(&rule)

	if q.Error == nil {
		return rule.FeeRuleAmountIDR, nil
	}
	if !errors.Is(q.Error, gorm.ErrRecordNotFound) {
		return 0, q.Error
	}

	// Fallback ke fixed jika disediakan
	if in.AmountStrategy.FixedAmountIDR != nil {
		return *in.AmountStrategy.FixedAmountIDR, nil
	}
	return 0, fmt.Errorf("no matching rule and no fixed fallback for option_code=%s", optionCode)
}
