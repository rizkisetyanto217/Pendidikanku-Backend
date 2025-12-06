// file: internals/features/finance/payments/dto/payment_compact.go
package dto

import (
	"encoding/json"
	"strings"
	"time"

	m "madinahsalam_backend/internals/features/finance/payments/model"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Response ringkas: fokus untuk UI list
type PaymentCompactResponse struct {
	PaymentID        uuid.UUID       `json:"payment_id"`
	PaymentNumber    *int64          `json:"payment_number,omitempty"` // nomor per sekolah
	PaymentStatus    m.PaymentStatus `json:"payment_status"`
	PaymentAmountIDR int             `json:"payment_amount_idr"` // samakan dengan model (int)

	PaymentMethod          m.PaymentMethod           `json:"payment_method"`                     // enum dari model
	PaymentGatewayProvider *m.PaymentGatewayProvider `json:"payment_gateway_provider,omitempty"` // pointer enum
	PaymentEntryType       m.PaymentEntryType        `json:"payment_entry_type"`                 // enum dari model

	PaymentInvoiceNumber    *string `json:"payment_invoice_number,omitempty"`
	PaymentExternalID       *string `json:"payment_external_id,omitempty"`
	PaymentGatewayReference *string `json:"payment_gateway_reference,omitempty"`
	PaymentManualReference  *string `json:"payment_manual_reference,omitempty"`
	PaymentDescription      *string `json:"payment_description,omitempty"`

	// ====== SNAPSHOT INFO PAYER ======
	// Diisi dari kolom snapshot di table payments (bukan lagi dari meta)
	// PayerName: kalau ada full_name_snapshot pakai itu, fallback ke user_name_snapshot
	PayerName *string `json:"payer_name,omitempty"`
	PayerRole *string `json:"payer_role,omitempty"` // masih diambil dari meta: payer_role_snapshot (kalau ada)

	// Info murid (kalau payment ini terkait class enrollment / siswa)
	// Sekarang tetap diambil dari meta (student_*_snapshot) kalau suatu saat kamu isi.
	StudentName *string `json:"student_name,omitempty"`
	StudentCode *string `json:"student_code,omitempty"`

	// Info kelas (opsional, dari meta kalau diisi)
	ClassName *string `json:"class_name,omitempty"`

	// Kategori pembayaran (registration / spp / dll)
	FeeRuleCategorySnapshot *string `json:"fee_rule_gbk_category_snapshot,omitempty"`

	// ====== SNAPSHOT ACADEMIC TERM (dari kolom di payments) ======
	AcademicTermID            *uuid.UUID `json:"academic_term_id,omitempty"`
	AcademicTermName          *string    `json:"academic_term_name,omitempty"`
	AcademicTermAcademicYear  *string    `json:"academic_term_academic_year,omitempty"`
	AcademicTermSlug          *string    `json:"academic_term_slug,omitempty"`
	AcademicTermAngkatanCache *string    `json:"academic_term_angkatan_cache,omitempty"`

	// ====== VA / channel snapshot (ringkas untuk list) ======
	PaymentChannelSnapshot  *string `json:"payment_channel_snapshot,omitempty"`
	PaymentBankSnapshot     *string `json:"payment_bank_snapshot,omitempty"`
	PaymentVANumberSnapshot *string `json:"payment_va_number_snapshot,omitempty"`
	PaymentVANameSnapshot   *string `json:"payment_va_name_snapshot,omitempty"`

	PaymentCreatedAt time.Time `json:"payment_created_at"`
}

/* ================== helpers JSONB (untuk meta lama) ================== */

func jsonStr(j datatypes.JSON, key string) *string {
	if len(j) == 0 || string(j) == "null" {
		return nil
	}
	var mm map[string]any
	if err := json.Unmarshal(j, &mm); err != nil {
		return nil
	}
	if v, ok := mm[key]; ok && v != nil {
		if s, ok := v.(string); ok {
			s = strings.TrimSpace(s)
			if s != "" {
				return &s
			}
		}
	}
	return nil
}

// helper kecil buat pilih nama yang paling cakep
func pickPayerName(fullName, userName *string) *string {
	for _, src := range []*string{fullName, userName} {
		if src != nil {
			if s := strings.TrimSpace(*src); s != "" {
				v := s
				return &v
			}
		}
	}
	return nil
}

// Single model → compact DTO
func FromModelCompact(src *m.Payment) *PaymentCompactResponse {
	if src == nil {
		return nil
	}

	meta := src.PaymentMeta

	return &PaymentCompactResponse{
		PaymentID:        src.PaymentID,
		PaymentNumber:    src.PaymentNumber,
		PaymentStatus:    src.PaymentStatus,
		PaymentAmountIDR: src.PaymentAmountIDR,

		PaymentMethod:          src.PaymentMethod,
		PaymentGatewayProvider: src.PaymentGatewayProvider,
		PaymentEntryType:       src.PaymentEntryType,

		PaymentInvoiceNumber:    src.PaymentInvoiceNumber,
		PaymentExternalID:       src.PaymentExternalID,
		PaymentGatewayReference: src.PaymentGatewayReference,
		PaymentManualReference:  src.PaymentManualReference,
		PaymentDescription:      src.PaymentDescription,

		// ---- SNAPSHOT PAYER (pakai kolom snapshot di table) ----
		PayerName: pickPayerName(src.PaymentFullNameSnapshot, src.PaymentUserNameSnapshot),
		// Role masih dari meta (kalau suatu saat kamu isi payer_role_snapshot)
		PayerRole: jsonStr(meta, "payer_role_snapshot"),

		// ---- SNAPSHOT SISWA & KELAS (opsional dari meta) ----
		StudentName: jsonStr(meta, "student_name_snapshot"),
		StudentCode: jsonStr(meta, "student_code_snapshot"),
		ClassName:   jsonStr(meta, "class_name_snapshot"),

		// Kategori fee rule (ini sudah dipakai di flow registration)
		FeeRuleCategorySnapshot: jsonStr(meta, "fee_rule_gbk_category_snapshot"),

		// ---- SNAPSHOT ACADEMIC TERM (dari kolom di payments) ----
		AcademicTermID:            src.PaymentAcademicTermID,
		AcademicTermName:          src.PaymentAcademicTermNameCache,
		AcademicTermAcademicYear:  src.PaymentAcademicTermAcademicYearCache,
		AcademicTermSlug:          src.PaymentAcademicTermSlugCache,
		AcademicTermAngkatanCache: src.PaymentAcademicTermAngkatanCache,

		// ---- VA / channel snapshot (dari kolom di payments) ----
		PaymentChannelSnapshot:  src.PaymentChannelSnapshot,
		PaymentBankSnapshot:     src.PaymentBankSnapshot,
		PaymentVANumberSnapshot: src.PaymentVANumberSnapshot,
		PaymentVANameSnapshot:   src.PaymentVANameSnapshot,

		PaymentCreatedAt: src.PaymentCreatedAt,
	}
}

// Slice model → slice compact DTO
func FromModelsCompact(src []m.Payment) []*PaymentCompactResponse {
	out := make([]*PaymentCompactResponse, 0, len(src))
	for i := range src {
		out = append(out, FromModelCompact(&src[i]))
	}
	return out
}
