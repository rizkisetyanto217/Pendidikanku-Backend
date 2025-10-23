// file: internals/features/finance/spp/api/controllers.go
package api

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

/* =======================================================
   BILL BATCHES (AUTHORIZED + TENANT-SCOPED)
======================================================= */

// POST /:masjid_id/spp/bill-batches
func (h *Handler) CreateBillBatch(c *fiber.Ctx) error {
	masjidID, err := mustMasjidID(c)
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "invalid masjid_id")
	}
	if err := helperAuth.EnsureStaffMasjid(c, masjidID); err != nil {
		return err
	}

	var in dto.BillBatchCreateDTO
	if err := c.BodyParser(&in); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "invalid json")
	}
	if !xorValid(in.BillBatchClassID, in.BillBatchSectionID) {
		return helper.JsonError(c, http.StatusBadRequest, "exactly one of bill_batch_class_id or bill_batch_section_id must be set")
	}

	// >>> selalu set dari path (abaikan body)
	in.BillBatchMasjidID = masjidID

	m := dto.BillBatchCreateDTOToModel(in)
	if err := h.DB.Create(&m).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, http.StatusConflict, "duplicate bill batch for the given scope and period")
		}
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}
	return helper.JsonCreated(c, "bill batch created", dto.ToBillBatchResponse(m))
}

// PATCH /:masjid_id/spp/bill-batches/:id
func (h *Handler) UpdateBillBatch(c *fiber.Ctx) error {
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

	var in dto.BillBatchUpdateDTO
	if err := c.BodyParser(&in); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "invalid json")
	}

	var m billing.BillBatch
	if err := h.DB.First(&m,
		"bill_batch_id = ? AND bill_batch_masjid_id = ? AND bill_batch_deleted_at IS NULL",
		id, masjidID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "bill_batch not found")
		}
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	if err := dto.ApplyBillBatchUpdate(&m, in); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	if err := h.DB.Save(&m).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, http.StatusConflict, "duplicate bill batch for the given scope and period")
		}
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}
	return helper.JsonOK(c, "bill batch updated", dto.ToBillBatchResponse(m))
}

// GETs untuk BillBatch: (sudah tenant scoped via WHERE bill_batch_masjid_id = ?)
// — tidak perlu perubahan dari versi kamu —

/* =======================================================
   STUDENT BILLS (AUTHORIZED + TENANT-SCOPED)
======================================================= */

// GET /:masjid_id/spp/student-bills/:id
func (h *Handler) GetStudentBill(c *fiber.Ctx) error {
	masjidID, err := mustMasjidID(c)
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "invalid masjid_id")
	}
	if err := helperAuth.EnsureMemberMasjid(c, masjidID); err != nil {
		return err
	}

	id, err := parseUUID(c, "id")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "invalid id")
	}
	var m billing.StudentBill
	if err := h.DB.First(&m, "student_bill_id = ? AND student_bill_masjid_id = ? AND student_bill_deleted_at IS NULL", id, masjidID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "student_bill not found")
		}
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}
	return helper.JsonOK(c, "ok", dto.ToStudentBillResponse(m))
}

// POST /:masjid_id/spp/student-bills/:id/mark-paid
func (h *Handler) MarkStudentBillPaid(c *fiber.Ctx) error {
	masjidID, err := mustMasjidID(c)
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "invalid masjid_id")
	}
	if err := helperAuth.EnsureStaffMasjid(c, masjidID); err != nil { // kasir/staff
		return err
	}

	id, err := parseUUID(c, "id")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "invalid id")
	}
	var in dto.StudentBillMarkPaidDTO
	if err := c.BodyParser(&in); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "invalid json")
	}
	var m billing.StudentBill
	if err := h.DB.First(&m, "student_bill_id = ? AND student_bill_masjid_id = ? AND student_bill_deleted_at IS NULL", id, masjidID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "student_bill not found")
		}
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}
	now := time.Now()
	if in.PaidAt == nil {
		in.PaidAt = &now
	}
	m.StudentBillStatus = "paid"
	m.StudentBillPaidAt = in.PaidAt
	m.StudentBillNote = in.Note
	if err := h.DB.Save(&m).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}
	return helper.JsonOK(c, "student bill marked paid", dto.ToStudentBillResponse(m))
}

// POST /:masjid_id/spp/student-bills/:id/mark-unpaid
func (h *Handler) MarkStudentBillUnpaid(c *fiber.Ctx) error {
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
	var in dto.StudentBillMarkUnpaidDTO
	if err := c.BodyParser(&in); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "invalid json")
	}
	var m billing.StudentBill
	if err := h.DB.First(&m, "student_bill_id = ? AND student_bill_masjid_id = ? AND student_bill_deleted_at IS NULL", id, masjidID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "student_bill not found")
		}
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}
	m.StudentBillStatus = "unpaid"
	m.StudentBillPaidAt = nil
	m.StudentBillNote = in.Note
	if err := h.DB.Save(&m).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}
	return helper.JsonOK(c, "student bill marked unpaid", dto.ToStudentBillResponse(m))
}

// POST /:masjid_id/spp/student-bills/:id/cancel
func (h *Handler) CancelStudentBill(c *fiber.Ctx) error {
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
	var in dto.StudentBillCancelDTO
	if err := c.BodyParser(&in); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "invalid json")
	}
	var m billing.StudentBill
	if err := h.DB.First(&m, "student_bill_id = ? AND student_bill_masjid_id = ? AND student_bill_deleted_at IS NULL", id, masjidID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "student_bill not found")
		}
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}
	m.StudentBillStatus = "canceled"
	m.StudentBillNote = in.Note
	if err := h.DB.Save(&m).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}
	return helper.JsonOK(c, "student bill canceled", dto.ToStudentBillResponse(m))
}

/* =======================================================
   GENERATE STUDENT BILLS FROM BATCH (AUTHORIZED)
======================================================= */

// POST /:masjid_id/spp/generate
/* =======================================================
   GENERATE STUDENT BILLS FROM BATCH (AUTHORIZED)
======================================================= */

// POST /:masjid_id/spp/generate
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

	// >>> selalu set dari path (abaikan body)
	in.StudentBillMasjidID = masjidID

	// 1) Validasi batch (tenant scoped)
	var batch billing.BillBatch
	if err := h.DB.First(&batch,
		"bill_batch_id = ? AND bill_batch_masjid_id = ? AND bill_batch_deleted_at IS NULL",
		in.BillBatchID, masjidID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "bill_batch not found")
		}
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	// 2) Target siswa (query sudah pakai in.StudentBillMasjidID yang barusan di-set)
	targetIDs, err := h.resolveTargetStudents(in)
	if err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}
	if len(targetIDs) == 0 {
		return helper.JsonOK(c, "no target students", dto.GenerateStudentBillsResponse{
			BillBatchID: in.BillBatchID,
			Inserted:    0,
			Skipped:     0,
		})
	}

	// 3) Idempotency ringan
	if in.IdempotencyKey != nil {
		var count int64
		if err := h.DB.Model(&billing.StudentBill{}).
			Where("student_bill_batch_id = ?", in.BillBatchID).
			Where("student_bill_masjid_id = ?", masjidID).
			Where("student_bill_deleted_at IS NULL").
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
			amount, err := h.resolveAmount(tx, in, batch, sid)
			if err != nil {
				return err
			}
			usb := billing.StudentBill{
				StudentBillBatchID:         in.BillBatchID,
				StudentBillMasjidID:        in.StudentBillMasjidID, // sudah = masjidID
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
		return nil
	})
	if err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	return helper.JsonOK(c, "student bills generated", res)
}

// resolveTargetStudents: ambil daftar masjid_student_id (tenant already in request)
func (h *Handler) resolveTargetStudents(in dto.GenerateStudentBillsRequest) ([]uuid.UUID, error) {
	switch in.Source.Type {
	case "class":
		if in.Source.ClassID == nil {
			return nil, fmt.Errorf("class_id required")
		}
		type row struct{ ID uuid.UUID }
		var rows []row
		q := h.DB.Table("masjid_students").
			Select("masjid_student_id AS id").
			Where("masjid_student_masjid_id = ?", in.StudentBillMasjidID).
			Where("class_id = ?", *in.Source.ClassID)
		if in.Filters != nil && in.Filters.OnlyActiveStudents {
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

	case "students":
		return in.Source.MasjidStudentIDs, nil
	default:
		return nil, fmt.Errorf("unsupported source type")
	}
}

// resolveAmount: pilih nominal dari rules (jika diminta) atau fixed (tenant-safe)
func (h *Handler) resolveAmount(tx *gorm.DB, in dto.GenerateStudentBillsRequest, batch billing.BillBatch, studentID uuid.UUID) (int, error) {
	// Fixed only
	if in.AmountStrategy.Mode == "fixed" {
		if in.AmountStrategy.FixedAmountIDR == nil {
			return 0, fmt.Errorf("fixed_amount_idr required for mode=fixed")
		}
		return *in.AmountStrategy.FixedAmountIDR, nil
	}

	// rule_fallback_fixed
	if in.AmountStrategy.PreferRule == nil {
		return 0, fmt.Errorf("prefer_rule required for mode=rule_fallback_fixed")
	}
	optionCode := in.AmountStrategy.PreferRule.OptionCode

	// tanggal efektif referensi
	dueDate := batch.BillBatchDueDate
	eff := time.Now()
	if dueDate != nil {
		eff = *dueDate
	}

	q := tx.Model(&billing.FeeRule{}).
		Where("fee_rule_masjid_id = ?", in.StudentBillMasjidID).
		Where("LOWER(fee_rule_option_code) = ?", strings.ToLower(optionCode)).
		Where("fee_rule_deleted_at IS NULL").
		Where("?::date >= COALESCE(fee_rule_effective_from, '-infinity'::date) AND ?::date <= COALESCE(fee_rule_effective_to, 'infinity'::date)", eff, eff)

	switch in.AmountStrategy.PreferRule.By {
	case "term":
		if batch.BillBatchTermID == nil {
			q = q.Where("fee_rule_term_id IS NULL AND fee_rule_year = ? AND fee_rule_month = ?", batch.BillBatchYear, batch.BillBatchMonth)
		} else {
			q = q.Where("fee_rule_term_id = ?", *batch.BillBatchTermID)
		}
	case "ym":
		q = q.Where("fee_rule_term_id IS NULL AND fee_rule_year = ? AND fee_rule_month = ?", batch.BillBatchYear, batch.BillBatchMonth)
	default:
		return 0, fmt.Errorf("prefer_rule.by invalid")
	}

	// scope specificity ordering
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

	if err == nil {
		return rule.FeeRuleAmountIDR, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}

	// fallback to fixed if provided
	if in.AmountStrategy.FixedAmountIDR != nil {
		return *in.AmountStrategy.FixedAmountIDR, nil
	}
	return 0, fmt.Errorf("no matching rule and no fixed fallback for option_code=%s", optionCode)
}
