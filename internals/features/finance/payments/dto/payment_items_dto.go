package dto

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	m "madinahsalam_backend/internals/features/finance/payments/model"
)

/* =========================================================
   CREATE PAYMENT ITEM
========================================================= */

type CreatePaymentItemRequest struct {
	// Tenant & header
	PaymentItemSchoolID  uuid.UUID `json:"payment_item_school_id" validate:"required"`
	PaymentItemPaymentID uuid.UUID `json:"payment_item_payment_id" validate:"required"`
	PaymentItemIndex     int16     `json:"payment_item_index" validate:"required,min=1"`

	// === Target per item (mirror DDL baru) ===
	PaymentItemUserGeneralBillingID *uuid.UUID `json:"payment_item_user_general_billing_id"`
	PaymentItemGeneralBillingID     *uuid.UUID `json:"payment_item_general_billing_id"`
	PaymentItemBillBatchID          *uuid.UUID `json:"payment_item_bill_batch_id"`

	// Subjek murid per item
	PaymentItemSchoolStudentID *uuid.UUID `json:"payment_item_school_student_id"`
	PaymentItemClassID         *uuid.UUID `json:"payment_item_class_id"`
	PaymentItemEnrollmentID    *uuid.UUID `json:"payment_item_enrollment_id"`

	// Nominal
	PaymentItemAmountIDR int `json:"payment_item_amount_idr" validate:"required,min=0"`

	// Fee rule snapshots
	PaymentItemFeeRuleID                  *uuid.UUID     `json:"payment_item_fee_rule_id"`
	PaymentItemFeeRuleOptionCodeSnapshot  *string        `json:"payment_item_fee_rule_option_code_snapshot"`
	PaymentItemFeeRuleOptionIndexSnapshot *int16         `json:"payment_item_fee_rule_option_index_snapshot"`
	PaymentItemFeeRuleAmountSnapshot      *int           `json:"payment_item_fee_rule_amount_snapshot"`
	PaymentItemFeeRuleScopeSnapshot       *m.FeeScope    `json:"payment_item_fee_rule_scope_snapshot"`
	PaymentItemFeeRuleNoteSnapshot        *string        `json:"payment_item_fee_rule_note_snapshot"`
	PaymentItemMeta                       datatypes.JSON `json:"payment_item_meta"`

	// Academic term snapshots
	PaymentItemAcademicTermID           *uuid.UUID `json:"payment_item_academic_term_id"`
	PaymentItemAcademicTermAcademicYear *string    `json:"payment_item_academic_term_academic_year_cache"`
	PaymentItemAcademicTermName         *string    `json:"payment_item_academic_term_name_cache"`
	PaymentItemAcademicTermSlug         *string    `json:"payment_item_academic_term_slug_cache"`
	PaymentItemAcademicTermAngkatan     *string    `json:"payment_item_academic_term_angkatan_cache"`

	// Invoice per item
	PaymentItemInvoiceNumber *string    `json:"payment_item_invoice_number"`
	PaymentItemInvoiceTitle  *string    `json:"payment_item_invoice_title"`
	PaymentItemInvoiceDue    *time.Time `json:"payment_item_invoice_due_date"`

	// Line title / deskripsi
	PaymentItemTitle       *string `json:"payment_item_title"`
	PaymentItemDescription *string `json:"payment_item_description"`
}

func (r *CreatePaymentItemRequest) Validate() error {
	// minimal 1 target (mirror CK di DB baru)
	hasTarget := r.PaymentItemUserGeneralBillingID != nil ||
		r.PaymentItemGeneralBillingID != nil ||
		r.PaymentItemBillBatchID != nil ||
		r.PaymentItemSchoolStudentID != nil

	if !hasTarget {
		return errors.New("wajib menyertakan salah satu target: payment_item_user_general_billing_id / payment_item_general_billing_id / payment_item_bill_batch_id / payment_item_school_student_id")
	}

	if r.PaymentItemAmountIDR < 0 {
		return errors.New("payment_item_amount_idr tidak boleh negatif")
	}

	if r.PaymentItemFeeRuleOptionIndexSnapshot != nil && *r.PaymentItemFeeRuleOptionIndexSnapshot < 1 {
		return errors.New("payment_item_fee_rule_option_index_snapshot minimal 1 (1-based)")
	}

	if r.PaymentItemFeeRuleAmountSnapshot != nil && *r.PaymentItemFeeRuleAmountSnapshot < 0 {
		return errors.New("payment_item_fee_rule_amount_snapshot tidak boleh negatif")
	}

	if r.PaymentItemIndex < 1 {
		return errors.New("payment_item_index minimal 1 (1-based)")
	}

	return nil
}

func (r *CreatePaymentItemRequest) ToModel() *m.PaymentItemModel {
	now := time.Now()

	return &m.PaymentItemModel{
		PaymentItemSchoolID:  r.PaymentItemSchoolID,
		PaymentItemPaymentID: r.PaymentItemPaymentID,
		PaymentItemIndex:     r.PaymentItemIndex,

		// target (baru)
		PaymentItemUserGeneralBillingID: r.PaymentItemUserGeneralBillingID,
		PaymentItemGeneralBillingID:     r.PaymentItemGeneralBillingID,
		PaymentItemBillBatchID:          r.PaymentItemBillBatchID,

		PaymentItemSchoolStudentID: r.PaymentItemSchoolStudentID,
		PaymentItemClassID:         r.PaymentItemClassID,
		PaymentItemEnrollmentID:    r.PaymentItemEnrollmentID,

		PaymentItemAmountIDR: r.PaymentItemAmountIDR,

		PaymentItemFeeRuleID:                  r.PaymentItemFeeRuleID,
		PaymentItemFeeRuleOptionCodeSnapshot:  r.PaymentItemFeeRuleOptionCodeSnapshot,
		PaymentItemFeeRuleOptionIndexSnapshot: r.PaymentItemFeeRuleOptionIndexSnapshot,
		PaymentItemFeeRuleAmountSnapshot:      r.PaymentItemFeeRuleAmountSnapshot,

		PaymentItemFeeRuleScopeSnapshot: r.PaymentItemFeeRuleScopeSnapshot,
		PaymentItemFeeRuleNoteSnapshot:  r.PaymentItemFeeRuleNoteSnapshot,

		PaymentItemAcademicTermID:           r.PaymentItemAcademicTermID,
		PaymentItemAcademicTermAcademicYear: r.PaymentItemAcademicTermAcademicYear,
		PaymentItemAcademicTermName:         r.PaymentItemAcademicTermName,
		PaymentItemAcademicTermSlug:         r.PaymentItemAcademicTermSlug,
		PaymentItemAcademicTermAngkatan:     r.PaymentItemAcademicTermAngkatan,

		PaymentItemInvoiceNumber: r.PaymentItemInvoiceNumber,
		PaymentItemInvoiceTitle:  r.PaymentItemInvoiceTitle,
		PaymentItemInvoiceDue:    r.PaymentItemInvoiceDue,

		PaymentItemTitle:       r.PaymentItemTitle,
		PaymentItemDescription: r.PaymentItemDescription,
		PaymentItemMeta:        r.PaymentItemMeta,

		PaymentItemCreatedAt: now,
		PaymentItemUpdatedAt: now,
	}
}

/*
	=========================================================
	  UPDATE (PATCH) PAYMENT ITEM

=========================================================
*/
type UpdatePaymentItemRequest struct {
	PaymentItemSchoolID  PatchField[uuid.UUID] `json:"payment_item_school_id"`
	PaymentItemPaymentID PatchField[uuid.UUID] `json:"payment_item_payment_id"`
	PaymentItemIndex     PatchField[int16]     `json:"payment_item_index"`

	// targets (baru)
	PaymentItemUserGeneralBillingID PatchField[uuid.UUID] `json:"payment_item_user_general_billing_id"`
	PaymentItemGeneralBillingID     PatchField[uuid.UUID] `json:"payment_item_general_billing_id"`
	PaymentItemBillBatchID          PatchField[uuid.UUID] `json:"payment_item_bill_batch_id"`

	PaymentItemSchoolStudentID PatchField[uuid.UUID] `json:"payment_item_school_student_id"`
	PaymentItemClassID         PatchField[uuid.UUID] `json:"payment_item_class_id"`
	PaymentItemEnrollmentID    PatchField[uuid.UUID] `json:"payment_item_enrollment_id"`

	PaymentItemAmountIDR PatchField[int] `json:"payment_item_amount_idr"`

	PaymentItemFeeRuleID                  PatchField[uuid.UUID]      `json:"payment_item_fee_rule_id"`
	PaymentItemFeeRuleOptionCodeSnapshot  PatchField[string]         `json:"payment_item_fee_rule_option_code_snapshot"`
	PaymentItemFeeRuleOptionIndexSnapshot PatchField[int16]          `json:"payment_item_fee_rule_option_index_snapshot"`
	PaymentItemFeeRuleAmountSnapshot      PatchField[int]            `json:"payment_item_fee_rule_amount_snapshot"`
	PaymentItemFeeRuleScopeSnapshot       PatchField[m.FeeScope]     `json:"payment_item_fee_rule_scope_snapshot"`
	PaymentItemFeeRuleNoteSnapshot        PatchField[string]         `json:"payment_item_fee_rule_note_snapshot"`
	PaymentItemMeta                       PatchField[datatypes.JSON] `json:"payment_item_meta"`

	PaymentItemAcademicTermID           PatchField[uuid.UUID] `json:"payment_item_academic_term_id"`
	PaymentItemAcademicTermAcademicYear PatchField[string]    `json:"payment_item_academic_term_academic_year_cache"`
	PaymentItemAcademicTermName         PatchField[string]    `json:"payment_item_academic_term_name_cache"`
	PaymentItemAcademicTermSlug         PatchField[string]    `json:"payment_item_academic_term_slug_cache"`
	PaymentItemAcademicTermAngkatan     PatchField[string]    `json:"payment_item_academic_term_angkatan_cache"`

	PaymentItemInvoiceNumber PatchField[string]    `json:"payment_item_invoice_number"`
	PaymentItemInvoiceTitle  PatchField[string]    `json:"payment_item_invoice_title"`
	PaymentItemInvoiceDue    PatchField[time.Time] `json:"payment_item_invoice_due_date"`

	PaymentItemTitle       PatchField[string] `json:"payment_item_title"`
	PaymentItemDescription PatchField[string] `json:"payment_item_description"`
}

func (p *UpdatePaymentItemRequest) Apply(mo *m.PaymentItemModel) error {
	// basic FK: value-type → pakai applyVal
	applyVal(&mo.PaymentItemSchoolID, p.PaymentItemSchoolID)
	applyVal(&mo.PaymentItemPaymentID, p.PaymentItemPaymentID)

	if p.PaymentItemIndex.Set {
		if p.PaymentItemIndex.Null || p.PaymentItemIndex.Value == nil {
			return errors.New("payment_item_index tidak boleh null")
		}
		if *p.PaymentItemIndex.Value < 1 {
			return errors.New("payment_item_index minimal 1 (1-based)")
		}
		mo.PaymentItemIndex = *p.PaymentItemIndex.Value
	}

	// targets
	targetPatched := p.PaymentItemUserGeneralBillingID.Set ||
		p.PaymentItemGeneralBillingID.Set ||
		p.PaymentItemBillBatchID.Set ||
		p.PaymentItemSchoolStudentID.Set

	applyPtr(&mo.PaymentItemUserGeneralBillingID, p.PaymentItemUserGeneralBillingID)
	applyPtr(&mo.PaymentItemGeneralBillingID, p.PaymentItemGeneralBillingID)
	applyPtr(&mo.PaymentItemBillBatchID, p.PaymentItemBillBatchID)

	applyPtr(&mo.PaymentItemSchoolStudentID, p.PaymentItemSchoolStudentID)
	applyPtr(&mo.PaymentItemClassID, p.PaymentItemClassID)
	applyPtr(&mo.PaymentItemEnrollmentID, p.PaymentItemEnrollmentID)

	// amount
	if p.PaymentItemAmountIDR.Set {
		if p.PaymentItemAmountIDR.Null || p.PaymentItemAmountIDR.Value == nil {
			return errors.New("payment_item_amount_idr tidak boleh null")
		}
		if *p.PaymentItemAmountIDR.Value < 0 {
			return errors.New("payment_item_amount_idr tidak boleh negatif")
		}
		mo.PaymentItemAmountIDR = *p.PaymentItemAmountIDR.Value
	}

	// fee_rule snapshots
	applyPtr(&mo.PaymentItemFeeRuleID, p.PaymentItemFeeRuleID)
	applyPtr(&mo.PaymentItemFeeRuleOptionCodeSnapshot, p.PaymentItemFeeRuleOptionCodeSnapshot)

	if p.PaymentItemFeeRuleOptionIndexSnapshot.Set {
		if !p.PaymentItemFeeRuleOptionIndexSnapshot.Null &&
			p.PaymentItemFeeRuleOptionIndexSnapshot.Value != nil &&
			*p.PaymentItemFeeRuleOptionIndexSnapshot.Value < 1 {
			return errors.New("payment_item_fee_rule_option_index_snapshot minimal 1 (1-based)")
		}
		applyPtr(&mo.PaymentItemFeeRuleOptionIndexSnapshot, p.PaymentItemFeeRuleOptionIndexSnapshot)
	}

	if p.PaymentItemFeeRuleAmountSnapshot.Set {
		if !p.PaymentItemFeeRuleAmountSnapshot.Null &&
			p.PaymentItemFeeRuleAmountSnapshot.Value != nil &&
			*p.PaymentItemFeeRuleAmountSnapshot.Value < 0 {
			return errors.New("payment_item_fee_rule_amount_snapshot tidak boleh negatif")
		}
		applyPtr(&mo.PaymentItemFeeRuleAmountSnapshot, p.PaymentItemFeeRuleAmountSnapshot)
	}

	applyPtr(&mo.PaymentItemFeeRuleScopeSnapshot, p.PaymentItemFeeRuleScopeSnapshot)
	applyPtr(&mo.PaymentItemFeeRuleNoteSnapshot, p.PaymentItemFeeRuleNoteSnapshot)
	applyVal(&mo.PaymentItemMeta, p.PaymentItemMeta)

	// academic term snapshots
	applyPtr(&mo.PaymentItemAcademicTermID, p.PaymentItemAcademicTermID)
	applyPtr(&mo.PaymentItemAcademicTermAcademicYear, p.PaymentItemAcademicTermAcademicYear)
	applyPtr(&mo.PaymentItemAcademicTermName, p.PaymentItemAcademicTermName)
	applyPtr(&mo.PaymentItemAcademicTermSlug, p.PaymentItemAcademicTermSlug)
	applyPtr(&mo.PaymentItemAcademicTermAngkatan, p.PaymentItemAcademicTermAngkatan)

	// invoice
	applyPtr(&mo.PaymentItemInvoiceNumber, p.PaymentItemInvoiceNumber)
	applyPtr(&mo.PaymentItemInvoiceTitle, p.PaymentItemInvoiceTitle)
	applyPtr(&mo.PaymentItemInvoiceDue, p.PaymentItemInvoiceDue)

	// title/desc
	applyPtr(&mo.PaymentItemTitle, p.PaymentItemTitle)
	applyPtr(&mo.PaymentItemDescription, p.PaymentItemDescription)

	// jaga constraint target-any (mirror ck_payment_item_target_any)
	if targetPatched {
		hasTarget := mo.PaymentItemUserGeneralBillingID != nil ||
			mo.PaymentItemGeneralBillingID != nil ||
			mo.PaymentItemBillBatchID != nil ||
			mo.PaymentItemSchoolStudentID != nil

		if !hasTarget {
			return errors.New("setidaknya satu target harus diisi: payment_item_user_general_billing_id / payment_item_general_billing_id / payment_item_bill_batch_id / payment_item_school_student_id")
		}
	}

	return nil
}

/*
	=========================================================
	  RESPONSE DTO

=========================================================
*/
type PaymentItemResponse struct {
	PaymentItemID uuid.UUID `json:"payment_item_id"`

	PaymentItemSchoolID  uuid.UUID `json:"payment_item_school_id"`
	PaymentItemPaymentID uuid.UUID `json:"payment_item_payment_id"`
	PaymentItemIndex     int16     `json:"payment_item_index"`

	PaymentItemUserGeneralBillingID *uuid.UUID `json:"payment_item_user_general_billing_id"`
	PaymentItemGeneralBillingID     *uuid.UUID `json:"payment_item_general_billing_id"`
	PaymentItemBillBatchID          *uuid.UUID `json:"payment_item_bill_batch_id"`

	PaymentItemSchoolStudentID *uuid.UUID `json:"payment_item_school_student_id"`
	PaymentItemClassID         *uuid.UUID `json:"payment_item_class_id"`
	PaymentItemEnrollmentID    *uuid.UUID `json:"payment_item_enrollment_id"`

	PaymentItemAmountIDR int `json:"payment_item_amount_idr"`

	PaymentItemFeeRuleID                  *uuid.UUID     `json:"payment_item_fee_rule_id"`
	PaymentItemFeeRuleOptionCodeSnapshot  *string        `json:"payment_item_fee_rule_option_code_snapshot"`
	PaymentItemFeeRuleOptionIndexSnapshot *int16         `json:"payment_item_fee_rule_option_index_snapshot"`
	PaymentItemFeeRuleAmountSnapshot      *int           `json:"payment_item_fee_rule_amount_snapshot"`

	PaymentItemFeeRuleScopeSnapshot       *m.FeeScope    `json:"payment_item_fee_rule_scope_snapshot"`
	PaymentItemFeeRuleNoteSnapshot        *string        `json:"payment_item_fee_rule_note_snapshot"`
	PaymentItemMeta                       datatypes.JSON `json:"payment_item_meta"`

	PaymentItemAcademicTermID           *uuid.UUID `json:"payment_item_academic_term_id"`
	PaymentItemAcademicTermAcademicYear *string    `json:"payment_item_academic_term_academic_year_cache"`
	PaymentItemAcademicTermName         *string    `json:"payment_item_academic_term_name_cache"`
	PaymentItemAcademicTermSlug         *string    `json:"payment_item_academic_term_slug_cache"`
	PaymentItemAcademicTermAngkatan     *string    `json:"payment_item_academic_term_angkatan_cache"`

	PaymentItemInvoiceNumber *string    `json:"payment_item_invoice_number"`
	PaymentItemInvoiceTitle  *string    `json:"payment_item_invoice_title"`
	PaymentItemInvoiceDue    *time.Time `json:"payment_item_invoice_due_date"`

	PaymentItemTitle       *string    `json:"payment_item_title"`
	PaymentItemDescription *string    `json:"payment_item_description"`
	PaymentItemCreatedAt   time.Time  `json:"payment_item_created_at"`
	PaymentItemUpdatedAt   time.Time  `json:"payment_item_updated_at"`
	PaymentItemDeletedAt   *time.Time `json:"payment_item_deleted_at"`
}

func FromPaymentItemModel(mo *m.PaymentItemModel) *PaymentItemResponse {
	if mo == nil {
		return nil
	}

	return &PaymentItemResponse{
		PaymentItemID: mo.PaymentItemID,

		PaymentItemSchoolID:  mo.PaymentItemSchoolID,
		PaymentItemPaymentID: mo.PaymentItemPaymentID,
		PaymentItemIndex:     mo.PaymentItemIndex,

		PaymentItemUserGeneralBillingID: mo.PaymentItemUserGeneralBillingID,
		PaymentItemGeneralBillingID:     mo.PaymentItemGeneralBillingID,
		PaymentItemBillBatchID:          mo.PaymentItemBillBatchID,

		PaymentItemSchoolStudentID: mo.PaymentItemSchoolStudentID,
		PaymentItemClassID:         mo.PaymentItemClassID,
		PaymentItemEnrollmentID:    mo.PaymentItemEnrollmentID,

		PaymentItemAmountIDR: mo.PaymentItemAmountIDR,

		PaymentItemFeeRuleID:                  mo.PaymentItemFeeRuleID,
		PaymentItemFeeRuleOptionCodeSnapshot:  mo.PaymentItemFeeRuleOptionCodeSnapshot,
		PaymentItemFeeRuleOptionIndexSnapshot: mo.PaymentItemFeeRuleOptionIndexSnapshot,
		PaymentItemFeeRuleAmountSnapshot:      mo.PaymentItemFeeRuleAmountSnapshot,
		PaymentItemFeeRuleScopeSnapshot:       mo.PaymentItemFeeRuleScopeSnapshot,
		PaymentItemFeeRuleNoteSnapshot:        mo.PaymentItemFeeRuleNoteSnapshot,
		PaymentItemMeta:                       mo.PaymentItemMeta,

		PaymentItemAcademicTermID:           mo.PaymentItemAcademicTermID,
		PaymentItemAcademicTermAcademicYear: mo.PaymentItemAcademicTermAcademicYear,
		PaymentItemAcademicTermName:         mo.PaymentItemAcademicTermName,
		PaymentItemAcademicTermSlug:         mo.PaymentItemAcademicTermSlug,
		PaymentItemAcademicTermAngkatan:     mo.PaymentItemAcademicTermAngkatan,

		PaymentItemInvoiceNumber: mo.PaymentItemInvoiceNumber,
		PaymentItemInvoiceTitle:  mo.PaymentItemInvoiceTitle,
		PaymentItemInvoiceDue:    mo.PaymentItemInvoiceDue,

		PaymentItemTitle:       mo.PaymentItemTitle,
		PaymentItemDescription: mo.PaymentItemDescription,

		PaymentItemCreatedAt: mo.PaymentItemCreatedAt,
		PaymentItemUpdatedAt: mo.PaymentItemUpdatedAt,
		PaymentItemDeletedAt: mo.PaymentItemDeletedAt,
	}
}

func FromPaymentItemModels(src []m.PaymentItemModel) []*PaymentItemResponse {
	out := make([]*PaymentItemResponse, 0, len(src))
	for i := range src {
		out = append(out, FromPaymentItemModel(&src[i]))
	}
	return out
}
