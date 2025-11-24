// dto/ui_theme_preset_dto.go
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
UIThemePresetRequest
- Dipakai untuk CREATE, UPDATE, maupun PATCH.
- Aturan umum:
  - CREATE: wajib kirim Code, Name, Light, Dark (semua non-nil & valid)
  - UPDATE (replace-style): field pointer non-nil → REPLACE nilai (scalar/JSON)
  - PATCH: bisa campur:
  - REPLACE penuh JSON via Light/Dark (pointer JSON non-nil)
  - MERGE parsial JSON via LightPatch/DarkPatch (RFC 7386: null = hapus key)
  - Controller bebas menentukan "mode" dan validasi yang diperlukan
    (misal, pastikan field wajib terisi pada CREATE).
*/
type UIThemePresetRequest struct {
	// Scalar fields (pointer → optional)
	UIThemePresetCode *string `json:"ui_theme_preset_code,omitempty" validate:"omitempty,max=64"`
	UIThemePresetName *string `json:"ui_theme_preset_name,omitempty" validate:"omitempty,max=128"`

	// JSON REPLACE (pointer → optional, jika non-nil: replace total)
	UIThemePresetLight *datatypes.JSON `json:"ui_theme_preset_light,omitempty"`
	UIThemePresetDark  *datatypes.JSON `json:"ui_theme_preset_dark,omitempty"`

	// JSON PATCH (merge parsial; kirim object JSON)
	UIThemePresetLightPatch *json.RawMessage `json:"ui_theme_preset_light_patch,omitempty"`
	UIThemePresetDarkPatch  *json.RawMessage `json:"ui_theme_preset_dark_patch,omitempty"`
}

/*
UIThemePresetResponse
- Dipakai sebagai bentuk keluaran/API response.
*/
type UIThemePresetResponse struct {
	UIThemePresetID    uuid.UUID      `json:"ui_theme_preset_id"`
	UIThemePresetCode  string         `json:"ui_theme_preset_code"`
	UIThemePresetName  string         `json:"ui_theme_preset_name"`
	UIThemePresetLight datatypes.JSON `json:"ui_theme_preset_light"`
	UIThemePresetDark  datatypes.JSON `json:"ui_theme_preset_dark"`
}

/* =========================
   Helper Validations
========================= */

// Validasi sederhana untuk CREATE use-case.
func (r *UIThemePresetRequest) ValidateCreate() error {
	if r.UIThemePresetCode == nil || len(strings.TrimSpace(*r.UIThemePresetCode)) == 0 {
		return errors.New("ui_theme_preset_code is required")
	}
	if r.UIThemePresetName == nil || len(strings.TrimSpace(*r.UIThemePresetName)) == 0 {
		return errors.New("ui_theme_preset_name is required")
	}
	if r.UIThemePresetLight == nil {
		return errors.New("ui_theme_preset_light is required")
	}
	if r.UIThemePresetDark == nil {
		return errors.New("ui_theme_preset_dark is required")
	}
	return nil
}

// Cek apakah request benar-benar membawa sesuatu untuk diubah (berguna untuk PATCH/UPDATE).
func (r *UIThemePresetRequest) IsNoop() bool {
	return r.UIThemePresetCode == nil &&
		r.UIThemePresetName == nil &&
		r.UIThemePresetLight == nil &&
		r.UIThemePresetDark == nil &&
		r.UIThemePresetLightPatch == nil &&
		r.UIThemePresetDarkPatch == nil
}

/* =========================
   Mapper (model -> response)
========================= */

func ToUIThemePresetResponse(m *model.UIThemePreset) UIThemePresetResponse {
	return UIThemePresetResponse{
		UIThemePresetID:    m.UIThemePresetID,
		UIThemePresetCode:  m.UIThemePresetCode,
		UIThemePresetName:  m.UIThemePresetName,
		UIThemePresetLight: m.UIThemePresetLight,
		UIThemePresetDark:  m.UIThemePresetDark,
	}
}

/* =========================
   Util JSON Merge (RFC 7386)
   - null pada patch → hapus key
   - object bersarang → recursive merge
   - array/primitif → replace
========================= */

func EnsureJSONObject(raw []byte) error {
	var any interface{}
	if err := json.Unmarshal(raw, &any); err != nil {
		return err
	}
	if _, ok := any.(map[string]interface{}); !ok {
		return errors.New("patch must be a JSON object")
	}
	return nil
}

func JSONMerge(base datatypes.JSON, patch []byte) (datatypes.JSON, error) {
	var baseMap map[string]interface{}
	if len(base) == 0 {
		baseMap = map[string]interface{}{}
	} else if err := json.Unmarshal(base, &baseMap); err != nil {
		return nil, err
	}

	var patchMap map[string]interface{}
	if err := json.Unmarshal(patch, &patchMap); err != nil {
		return nil, err
	}

	merged := mergeMaps(baseMap, patchMap)
	out, err := json.Marshal(merged)
	if err != nil {
		return nil, err
	}
	return datatypes.JSON(out), nil
}

func mergeMaps(dst, src map[string]interface{}) map[string]interface{} {
	if dst == nil {
		dst = map[string]interface{}{}
	}
	for k, v := range src {
		// RFC-7386: null → delete key
		if v == nil {
			delete(dst, k)
			continue
		}
		// nested object → recursive merge
		if vmap, ok := v.(map[string]interface{}); ok {
			if cur, ok2 := dst[k].(map[string]interface{}); ok2 {
				dst[k] = mergeMaps(cur, vmap)
			} else {
				dst[k] = mergeMaps(map[string]interface{}{}, vmap)
			}
			continue
		}
		// array/primitives → replace
		dst[k] = v
	}
	return dst
}

/* =========================
   ApplyPatch Helper
   - Utility untuk langsung menerapkan request ke entity model
   - Cocok dipakai di controller PATCH
========================= */

func ApplyPatchToModel(entity *model.UIThemePreset, req *UIThemePresetRequest) error {
	// scalar
	if req.UIThemePresetCode != nil {
		entity.UIThemePresetCode = *req.UIThemePresetCode
	}
	if req.UIThemePresetName != nil {
		entity.UIThemePresetName = *req.UIThemePresetName
	}

	// JSON replace
	if req.UIThemePresetLight != nil {
		entity.UIThemePresetLight = *req.UIThemePresetLight
	}
	if req.UIThemePresetDark != nil {
		entity.UIThemePresetDark = *req.UIThemePresetDark
	}

	// JSON merge (parsial)
	if req.UIThemePresetLightPatch != nil {
		if err := EnsureJSONObject(*req.UIThemePresetLightPatch); err != nil {
			return err
		}
		merged, err := JSONMerge(entity.UIThemePresetLight, *req.UIThemePresetLightPatch)
		if err != nil {
			return err
		}
		entity.UIThemePresetLight = merged
	}
	if req.UIThemePresetDarkPatch != nil {
		if err := EnsureJSONObject(*req.UIThemePresetDarkPatch); err != nil {
			return err
		}
		merged, err := JSONMerge(entity.UIThemePresetDark, *req.UIThemePresetDarkPatch)
		if err != nil {
			return err
		}
		entity.UIThemePresetDark = merged
	}

	// update timestamp (biar aman kalau controller lupa)
	entity.UIThemePresetUpdatedAt = time.Now()
	return nil
}

/* =========================
   (Opsional) Helper Uniq Violation
   - Biar controller bisa handle 409 lebih mudah tanpa util terpisah
========================= */

func IsUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate key value violates unique constraint") ||
		strings.Contains(msg, "sqlstate 23505") ||
		strings.Contains(msg, "duplicate key")
}
