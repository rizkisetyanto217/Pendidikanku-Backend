// file: internals/helpers/snapshot/gbk_snapshot.go
package snapshot

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GBKSnapshot merepresentasikan nilai-nilai yang perlu "dibekukan"
// dari general_billing_kinds untuk disalin ke fee_rules.*_snapshot
type GBKSnapshot struct {
	Code               *string
	Name               *string
	Category           *string // "billing" | "campaign"
	IsGlobal           *bool
	Visibility         *string // "public" | "internal" | nil
	IsRecurring        *bool
	RequiresMonthYear  *bool
	RequiresOptionCode *bool
	DefaultAmountIDR   *int
	IsActive           *bool
}

// ValidateAndSnapshotGBK:
// - Memuat baris GBK yang hidup (deleted_at NULL)
// - Memastikan tenant sesuai (atau GLOBAL = masjid_id NULL)
// - Mengembalikan snapshot siap pakai
func ValidateAndSnapshotGBK(
	tx *gorm.DB,
	expectMasjidID uuid.UUID, // boleh uuid.Nil: lewati guard tenant
	gbkID uuid.UUID,
) (*GBKSnapshot, error) {
	var row struct {
		MasjidID         *string `gorm:"column:masjid_id"` // text agar aman saat NULL
		Code             *string `gorm:"column:code"`
		Name             *string `gorm:"column:name"`
		Category         *string `gorm:"column:category"`
		IsGlobal         *bool   `gorm:"column:is_global"`
		Visibility       *string `gorm:"column:visibility"`
		IsRecurring      *bool   `gorm:"column:is_recurring"`
		ReqYM            *bool   `gorm:"column:req_ym"`
		ReqOpt           *bool   `gorm:"column:req_opt"`
		DefaultAmountIDR *int    `gorm:"column:default_amount_idr"`
		IsActive         *bool   `gorm:"column:is_active"`
	}

	// SELECT eksplisit sesuai schema migrasi terbaru
	q := tx.Raw(`
		SELECT
		  gbk.general_billing_kind_masjid_id::text               AS masjid_id,
		  gbk.general_billing_kind_code                           AS code,
		  gbk.general_billing_kind_name                           AS name,
		  gbk.general_billing_kind_category::text                 AS category,
		  gbk.general_billing_kind_is_global                      AS is_global,
		  gbk.general_billing_kind_visibility::text               AS visibility,
		  gbk.general_billing_kind_is_recurring                   AS is_recurring,
		  gbk.general_billing_kind_requires_month_year            AS req_ym,
		  gbk.general_billing_kind_requires_option_code           AS req_opt,
		  gbk.general_billing_kind_default_amount_idr             AS default_amount_idr,
		  gbk.general_billing_kind_is_active                      AS is_active
		FROM general_billing_kinds gbk
		WHERE gbk.general_billing_kind_id = ?
		  AND gbk.general_billing_kind_deleted_at IS NULL
		LIMIT 1
	`, gbkID).Scan(&row)

	if q.Error != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal memuat General Billing Kind")
	}
	// not found
	if row.Code == nil && row.Name == nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "General Billing Kind tidak ditemukan")
	}

	// Guard tenant: jika GBK punya masjid_id dan bukan GLOBAL (masjid_id != NULL),
	// pastikan sama dengan expectMasjidID (kecuali expectMasjidID == uuid.Nil => skip guard)
	if expectMasjidID != uuid.Nil && row.MasjidID != nil && strings.TrimSpace(*row.MasjidID) != "" {
		rmz, perr := uuid.Parse(strings.TrimSpace(*row.MasjidID))
		if perr != nil {
			return nil, fiber.NewError(fiber.StatusInternalServerError, "Format masjid_id GBK tidak valid")
		}
		if rmz != uuid.Nil && rmz != expectMasjidID {
			return nil, fiber.NewError(fiber.StatusForbidden, "GBK bukan milik masjid Anda")
		}
	}

	trimPtr := func(p *string) *string {
		if p == nil {
			return nil
		}
		v := strings.TrimSpace(*p)
		if v == "" {
			return nil
		}
		return &v
	}

	return &GBKSnapshot{
		Code:               trimPtr(row.Code),
		Name:               trimPtr(row.Name),
		Category:           trimPtr(row.Category),
		IsGlobal:           row.IsGlobal,
		Visibility:         trimPtr(row.Visibility),
		IsRecurring:        row.IsRecurring,
		RequiresMonthYear:  row.ReqYM,
		RequiresOptionCode: row.ReqOpt,
		DefaultAmountIDR:   row.DefaultAmountIDR,
		IsActive:           row.IsActive,
	}, nil
}
