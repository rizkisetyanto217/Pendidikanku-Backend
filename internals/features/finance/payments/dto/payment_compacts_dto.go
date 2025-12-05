// file: internals/features/finance/payments/dto/payment_compact.go
package dto

import (
	"encoding/json"
	"time"

	m "madinahsalam_backend/internals/features/finance/payments/model"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Response ringkas: fokus untuk UI list
type PaymentCompactResponse struct {
	PaymentID        uuid.UUID       `json:"payment_id"`
	PaymentNumber    *int64          `json:"payment_number,omitempty"` // ⬅️ nomor per sekolah
	PaymentStatus    m.PaymentStatus `json:"payment_status"`
	PaymentAmountIDR int             `json:"payment_amount_idr"` // ⬅️ samakan dengan model (int)

	PaymentMethod          m.PaymentMethod           `json:"payment_method"`                     // ⬅️ enum dari model
	PaymentGatewayProvider *m.PaymentGatewayProvider `json:"payment_gateway_provider,omitempty"` // ⬅️ pointer enum
	PaymentEntryType       m.PaymentEntryType        `json:"payment_entry_type"`                 // ⬅️ enum dari model

	PaymentInvoiceNumber    *string `json:"payment_invoice_number,omitempty"`
	PaymentExternalID       *string `json:"payment_external_id,omitempty"`
	PaymentGatewayReference *string `json:"payment_gateway_reference,omitempty"`
	PaymentManualReference  *string `json:"payment_manual_reference,omitempty"`
	PaymentDescription      *string `json:"payment_description,omitempty"`

	// ====== SNAPSHOT INFO (dari payment_meta / snapshots) ======
	// Siapa yang bayar
	PayerName *string `json:"payer_name,omitempty"`
	PayerRole *string `json:"payer_role,omitempty"` // student/parent/teacher/admin/dll

	// Info murid (kalau payment ini terkait class enrollment / siswa)
	StudentName *string `json:"student_name,omitempty"`
	StudentCode *string `json:"student_code,omitempty"`

	// Info kelas
	ClassName *string `json:"class_name,omitempty"`

	// Kategori pembayaran (registration / spp / dll)
	FeeRuleCategorySnapshot *string `json:"fee_rule_gbk_category_snapshot,omitempty"`

	PaymentCreatedAt time.Time `json:"payment_created_at"`
}

/* ================== helpers JSONB ================== */

func jsonStr(j datatypes.JSON, key string) *string {
	if len(j) == 0 || string(j) == "null" {
		return nil
	}
	var mm map[string]any
	if err := json.Unmarshal(j, &mm); err != nil {
		return nil
	}
	if v, ok := mm[key]; ok && v != nil {
		if s, ok := v.(string); ok && s != "" {
			return &s
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

		// ---- SNAPSHOT dari meta ----
		PayerName: jsonStr(meta, "payer_name_snapshot"),
		PayerRole: jsonStr(meta, "payer_role_snapshot"),

		StudentName: jsonStr(meta, "student_name_snapshot"),
		StudentCode: jsonStr(meta, "student_code_snapshot"),
		ClassName:   jsonStr(meta, "class_name_snapshot"),

		FeeRuleCategorySnapshot: jsonStr(meta, "fee_rule_gbk_category_snapshot"),

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
