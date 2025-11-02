// dto/ui_theme_choice_dto.go
package dto

import (
	"errors"
	"time"

	"schoolku_backend/internals/features/lembaga/ui/theme/model"

	"github.com/google/uuid"
)

/*
UIThemeChoiceRequest
- Dipakai untuk CREATE & PATCH.
- CREATE:
  - wajib kirim: school_id
  - pilih tepat salah satu: preset_id ATAU custom_preset_id (exactly-one)

- PATCH:
  - boleh ubah school_id (jarang), preset_id, custom_preset_id, is_default, is_enabled
  - Aturan switch:
  - Jika Anda mengirim preset_id pada PATCH → custom_preset_id otomatis di-clear (jadi nil)
  - Jika Anda mengirim custom_preset_id → preset_id otomatis di-clear (jadi nil)
*/
type UIThemeChoiceRequest struct {
	UIThemeChoiceSchoolID *uuid.UUID `json:"ui_theme_choice_school_id,omitempty"`

	// Exactly-one-of (lihat aturan di atas)
	UIThemeChoicePresetID       *uuid.UUID `json:"ui_theme_choice_preset_id,omitempty"`
	UIThemeChoiceCustomPresetID *uuid.UUID `json:"ui_theme_choice_custom_preset_id,omitempty"`

	UIThemeChoiceIsDefault *bool `json:"ui_theme_choice_is_default,omitempty"`
	UIThemeChoiceIsEnabled *bool `json:"ui_theme_choice_is_enabled,omitempty"`
}

/*
UIThemeChoiceResponse
*/
type UIThemeChoiceResponse struct {
	UIThemeChoiceID             uuid.UUID  `json:"ui_theme_choice_id"`
	UIThemeChoiceSchoolID       uuid.UUID  `json:"ui_theme_choice_school_id"`
	UIThemeChoicePresetID       *uuid.UUID `json:"ui_theme_choice_preset_id,omitempty"`
	UIThemeChoiceCustomPresetID *uuid.UUID `json:"ui_theme_choice_custom_preset_id,omitempty"`
	UIThemeChoiceIsDefault      bool       `json:"ui_theme_choice_is_default"`
	UIThemeChoiceIsEnabled      bool       `json:"ui_theme_choice_is_enabled"`
	UIThemeChoiceCreatedAt      time.Time  `json:"ui_theme_choice_created_at"`
	UIThemeChoiceUpdatedAt      time.Time  `json:"ui_theme_choice_updated_at"`
}

/* =========================
   Validation
========================= */

func (r *UIThemeChoiceRequest) ValidateCreate() error {
	if r.UIThemeChoiceSchoolID == nil {
		return errors.New("ui_theme_choice_school_id is required")
	}
	// exactly-one: preset xor custom
	hasPreset := r.UIThemeChoicePresetID != nil
	hasCustom := r.UIThemeChoiceCustomPresetID != nil

	if hasPreset == hasCustom {
		// true,true or false,false -> invalid
		return errors.New("exactly one of ui_theme_choice_preset_id or ui_theme_choice_custom_preset_id is required")
	}
	return nil
}

func (r *UIThemeChoiceRequest) IsNoop() bool {
	return r.UIThemeChoiceSchoolID == nil &&
		r.UIThemeChoicePresetID == nil &&
		r.UIThemeChoiceCustomPresetID == nil &&
		r.UIThemeChoiceIsDefault == nil &&
		r.UIThemeChoiceIsEnabled == nil
}

/* =========================
   Mapper
========================= */

func ToUIThemeChoiceResponse(m *model.UIThemeChoice) UIThemeChoiceResponse {
	return UIThemeChoiceResponse{
		UIThemeChoiceID:             m.UIThemeChoiceID,
		UIThemeChoiceSchoolID:       m.UIThemeChoiceSchoolID,
		UIThemeChoicePresetID:       m.UIThemeChoicePresetID,
		UIThemeChoiceCustomPresetID: m.UIThemeChoiceCustomPresetID,
		UIThemeChoiceIsDefault:      m.UIThemeChoiceIsDefault,
		UIThemeChoiceIsEnabled:      m.UIThemeChoiceIsEnabled,
		UIThemeChoiceCreatedAt:      m.UIThemeChoiceCreatedAt,
		UIThemeChoiceUpdatedAt:      m.UIThemeChoiceUpdatedAt,
	}
}

/* =========================
   ApplyPatch Helper
   - Terapkan PATCH ke entity dengan aturan:
     * Jika request membawa preset_id -> custom_id di-clear (nil)
     * Jika request membawa custom_id -> preset_id di-clear (nil)
     * Setelah terapan, pastikan EXACTLY-ONE terpenuhi.
========================= */

func ApplyPatchToChoiceModel(entity *model.UIThemeChoice, req *UIThemeChoiceRequest) error {
	// School ID (jarang diubah, tapi diizinkan jika diperlukan)
	if req.UIThemeChoiceSchoolID != nil {
		entity.UIThemeChoiceSchoolID = *req.UIThemeChoiceSchoolID
	}

	// Siapkan target copy
	var (
		targetPreset *uuid.UUID = entity.UIThemeChoicePresetID
		targetCustom *uuid.UUID = entity.UIThemeChoiceCustomPresetID
	)

	// Aturan switch:
	// - kalau kirim preset_id → pakai itu & kosongkan custom
	if req.UIThemeChoicePresetID != nil {
		targetPreset = req.UIThemeChoicePresetID
		targetCustom = nil
	}

	// - kalau kirim custom_preset_id → pakai itu & kosongkan preset
	if req.UIThemeChoiceCustomPresetID != nil {
		targetCustom = req.UIThemeChoiceCustomPresetID
		targetPreset = nil
	}

	// Exactly-one setelah patch
	hasPreset := targetPreset != nil
	hasCustom := targetCustom != nil
	if hasPreset == hasCustom { // true,true atau false,false
		return errors.New("exactly one of ui_theme_choice_preset_id or ui_theme_choice_custom_preset_id must be set after patch")
	}

	// Assign target
	entity.UIThemeChoicePresetID = targetPreset
	entity.UIThemeChoiceCustomPresetID = targetCustom

	// Flags
	if req.UIThemeChoiceIsDefault != nil {
		entity.UIThemeChoiceIsDefault = *req.UIThemeChoiceIsDefault
	}
	if req.UIThemeChoiceIsEnabled != nil {
		entity.UIThemeChoiceIsEnabled = *req.UIThemeChoiceIsEnabled
	}

	// Update timestamp
	entity.UIThemeChoiceUpdatedAt = time.Now()
	return nil
}
