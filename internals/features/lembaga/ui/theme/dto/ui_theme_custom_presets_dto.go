// dto/ui_theme_custom_preset_dto.go
package dto

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"madinahsalam_backend/internals/features/lembaga/ui/theme/model"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

/*
UIThemeCustomPresetRequest
- Dipakai untuk CREATE & PATCH.
- CREATE: wajib kirim Code, Name, Light, Dark.
- PATCH: boleh scalar (pointer → optional), JSON replace, atau JSON merge.
*/
type UIThemeCustomPresetRequest struct {
	// Scalar fields
	UIThemeCustomPresetSchoolID *uuid.UUID `json:"ui_theme_custom_preset_school_id,omitempty"`
	UIThemeCustomPresetCode     *string    `json:"ui_theme_custom_preset_code,omitempty" validate:"omitempty,max=64"`
	UIThemeCustomPresetName     *string    `json:"ui_theme_custom_preset_name,omitempty" validate:"omitempty,max=128"`

	// JSON replace
	UIThemeCustomPresetLight *datatypes.JSON `json:"ui_theme_custom_preset_light,omitempty"`
	UIThemeCustomPresetDark  *datatypes.JSON `json:"ui_theme_custom_preset_dark,omitempty"`

	// JSON merge (parsial RFC 7386)
	UIThemeCustomPresetLightPatch *json.RawMessage `json:"ui_theme_custom_preset_light_patch,omitempty"`
	UIThemeCustomPresetDarkPatch  *json.RawMessage `json:"ui_theme_custom_preset_dark_patch,omitempty"`

	// Optional base preset id
	UIThemeCustomBasePresetID *uuid.UUID `json:"ui_theme_custom_base_preset_id,omitempty"`

	// Flag aktif/nonaktif
	UIThemeCustomPresetIsActive *bool `json:"ui_theme_custom_preset_is_active,omitempty"`
}

/*
UIThemeCustomPresetResponse
- Output API response.
*/
type UIThemeCustomPresetResponse struct {
	UIThemeCustomPresetID        uuid.UUID      `json:"ui_theme_custom_preset_id"`
	UIThemeCustomPresetSchoolID  uuid.UUID      `json:"ui_theme_custom_preset_school_id"`
	UIThemeCustomPresetCode      string         `json:"ui_theme_custom_preset_code"`
	UIThemeCustomPresetName      string         `json:"ui_theme_custom_preset_name"`
	UIThemeCustomPresetLight     datatypes.JSON `json:"ui_theme_custom_preset_light"`
	UIThemeCustomPresetDark      datatypes.JSON `json:"ui_theme_custom_preset_dark"`
	UIThemeCustomBasePresetID    *uuid.UUID     `json:"ui_theme_custom_base_preset_id,omitempty"`
	UIThemeCustomPresetIsActive  bool           `json:"ui_theme_custom_preset_is_active"`
	UIThemeCustomPresetCreatedAt time.Time      `json:"ui_theme_custom_preset_created_at"`
	UIThemeCustomPresetUpdatedAt time.Time      `json:"ui_theme_custom_preset_updated_at"`
}

/* =========================
   Validation
========================= */

func (r *UIThemeCustomPresetRequest) ValidateCreate() error {
	if r.UIThemeCustomPresetSchoolID == nil {
		return errors.New("ui_theme_custom_preset_school_id is required")
	}
	if r.UIThemeCustomPresetCode == nil || len(strings.TrimSpace(*r.UIThemeCustomPresetCode)) == 0 {
		return errors.New("ui_theme_custom_preset_code is required")
	}
	if r.UIThemeCustomPresetName == nil || len(strings.TrimSpace(*r.UIThemeCustomPresetName)) == 0 {
		return errors.New("ui_theme_custom_preset_name is required")
	}
	if r.UIThemeCustomPresetLight == nil {
		return errors.New("ui_theme_custom_preset_light is required")
	}
	if r.UIThemeCustomPresetDark == nil {
		return errors.New("ui_theme_custom_preset_dark is required")
	}
	return nil
}

func (r *UIThemeCustomPresetRequest) IsNoop() bool {
	return r.UIThemeCustomPresetSchoolID == nil &&
		r.UIThemeCustomPresetCode == nil &&
		r.UIThemeCustomPresetName == nil &&
		r.UIThemeCustomPresetLight == nil &&
		r.UIThemeCustomPresetDark == nil &&
		r.UIThemeCustomPresetLightPatch == nil &&
		r.UIThemeCustomPresetDarkPatch == nil &&
		r.UIThemeCustomBasePresetID == nil &&
		r.UIThemeCustomPresetIsActive == nil
}

/* =========================
   Mapper
========================= */

func ToUIThemeCustomPresetResponse(m *model.UIThemeCustomPreset) UIThemeCustomPresetResponse {
	return UIThemeCustomPresetResponse{
		UIThemeCustomPresetID:        m.UIThemeCustomPresetID,
		UIThemeCustomPresetSchoolID:  m.UIThemeCustomPresetSchoolID,
		UIThemeCustomPresetCode:      m.UIThemeCustomPresetCode,
		UIThemeCustomPresetName:      m.UIThemeCustomPresetName,
		UIThemeCustomPresetLight:     m.UIThemeCustomPresetLight,
		UIThemeCustomPresetDark:      m.UIThemeCustomPresetDark,
		UIThemeCustomBasePresetID:    m.UIThemeCustomBasePresetID,
		UIThemeCustomPresetIsActive:  m.UIThemeCustomPresetIsActive,
		UIThemeCustomPresetCreatedAt: m.UIThemeCustomPresetCreatedAt,
		UIThemeCustomPresetUpdatedAt: m.UIThemeCustomPresetUpdatedAt,
	}
}

/* =========================
   ApplyPatch Helper
========================= */

func ApplyPatchToCustomModel(entity *model.UIThemeCustomPreset, req *UIThemeCustomPresetRequest) error {
	// Scalars
	if req.UIThemeCustomPresetSchoolID != nil {
		entity.UIThemeCustomPresetSchoolID = *req.UIThemeCustomPresetSchoolID
	}
	if req.UIThemeCustomPresetCode != nil {
		entity.UIThemeCustomPresetCode = *req.UIThemeCustomPresetCode
	}
	if req.UIThemeCustomPresetName != nil {
		entity.UIThemeCustomPresetName = *req.UIThemeCustomPresetName
	}
	if req.UIThemeCustomBasePresetID != nil {
		entity.UIThemeCustomBasePresetID = req.UIThemeCustomBasePresetID
	}
	if req.UIThemeCustomPresetIsActive != nil {
		entity.UIThemeCustomPresetIsActive = *req.UIThemeCustomPresetIsActive
	}

	// JSON replace
	if req.UIThemeCustomPresetLight != nil {
		entity.UIThemeCustomPresetLight = *req.UIThemeCustomPresetLight
	}
	if req.UIThemeCustomPresetDark != nil {
		entity.UIThemeCustomPresetDark = *req.UIThemeCustomPresetDark
	}

	// JSON merge
	if req.UIThemeCustomPresetLightPatch != nil {
		if err := EnsureJSONObject(*req.UIThemeCustomPresetLightPatch); err != nil {
			return err
		}
		merged, err := JSONMerge(entity.UIThemeCustomPresetLight, *req.UIThemeCustomPresetLightPatch)
		if err != nil {
			return err
		}
		entity.UIThemeCustomPresetLight = merged
	}
	if req.UIThemeCustomPresetDarkPatch != nil {
		if err := EnsureJSONObject(*req.UIThemeCustomPresetDarkPatch); err != nil {
			return err
		}
		merged, err := JSONMerge(entity.UIThemeCustomPresetDark, *req.UIThemeCustomPresetDarkPatch)
		if err != nil {
			return err
		}
		entity.UIThemeCustomPresetDark = merged
	}

	// updated_at
	entity.UIThemeCustomPresetUpdatedAt = time.Now()
	return nil
}
