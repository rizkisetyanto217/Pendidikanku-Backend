package dto

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	mModel "masjidku_backend/internals/features/lembaga/masjids/model"

	"github.com/google/uuid"
)

/* =========================================================
   PATCH FIELD — tri-state (absent | null | value)
========================================================= */

type PatchField[T any] struct {
	Present bool
	Value   *T
}

func (p *PatchField[T]) UnmarshalJSON(b []byte) error {
	p.Present = true
	if string(b) == "null" {
		p.Value = nil
		return nil
	}
	var v T
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	p.Value = &v
	return nil
}

func applyPatchPtr[T any](dst **T, pf PatchField[T]) {
	if !pf.Present {
		return
	}
	if pf.Value == nil {
		*dst = nil
		return
	}
	*dst = pf.Value
}

func applyPatchScalar[T any](dst *T, pf PatchField[T]) {
	if !pf.Present || pf.Value == nil {
		return
	}
	*dst = *pf.Value
}

/* =========================================================
   DB error helpers
========================================================= */

func IsUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate key value violates unique constraint") ||
		strings.Contains(msg, "sqlstate 23505") ||
		strings.Contains(msg, "duplicate key")
}

/* =========================================================
   CREATE REQUEST (match model)
   - support JSON & multipart (form tag)
========================================================= */

type CreateMasjidServicePlanRequest struct {
	MasjidServicePlanCode        string  `json:"masjid_service_plan_code" form:"masjid_service_plan_code" validate:"required,min=1,max=30,alphanumdash"`
	MasjidServicePlanName        string  `json:"masjid_service_plan_name" form:"masjid_service_plan_name" validate:"required,min=2,max=100"`
	MasjidServicePlanDescription *string `json:"masjid_service_plan_description" form:"masjid_service_plan_description" validate:"omitempty,max=2000"`

	// Image (current). *_old & *_delete_pending_until dikelola service saat swap.
	MasjidServicePlanImageURL       *string `json:"masjid_service_plan_image_url" form:"masjid_service_plan_image_url" validate:"omitempty,url,max=2048"`
	MasjidServicePlanImageObjectKey *string `json:"masjid_service_plan_image_object_key" form:"masjid_service_plan_image_object_key" validate:"omitempty,max=512"`

	MasjidServicePlanMaxTeachers  *int     `json:"masjid_service_plan_max_teachers" form:"masjid_service_plan_max_teachers" validate:"omitempty,min=0"`
	MasjidServicePlanMaxStudents  *int     `json:"masjid_service_plan_max_students" form:"masjid_service_plan_max_students" validate:"omitempty,min=0"`
	MasjidServicePlanMaxStorageMB *int     `json:"masjid_service_plan_max_storage_mb" form:"masjid_service_plan_max_storage_mb" validate:"omitempty,min=0"`
	MasjidServicePlanPriceMonthly *float64 `json:"masjid_service_plan_price_monthly" form:"masjid_service_plan_price_monthly" validate:"omitempty,gte=0"`
	MasjidServicePlanPriceYearly  *float64 `json:"masjid_service_plan_price_yearly" form:"masjid_service_plan_price_yearly" validate:"omitempty,gte=0"`

	MasjidServicePlanAllowCustomTheme bool `json:"masjid_service_plan_allow_custom_theme" form:"masjid_service_plan_allow_custom_theme"`
	MasjidServicePlanMaxCustomThemes  *int `json:"masjid_service_plan_max_custom_themes" form:"masjid_service_plan_max_custom_themes" validate:"omitempty,min=0"`

	MasjidServicePlanIsActive *bool `json:"masjid_service_plan_is_active" form:"masjid_service_plan_is_active" validate:"omitempty"`
}

func (r *CreateMasjidServicePlanRequest) ToModel() *mModel.MasjidServicePlan {
	m := &mModel.MasjidServicePlan{
		MasjidServicePlanCode:        r.MasjidServicePlanCode,
		MasjidServicePlanName:        r.MasjidServicePlanName,
		MasjidServicePlanDescription: r.MasjidServicePlanDescription,

		MasjidServicePlanImageURL:       r.MasjidServicePlanImageURL,
		MasjidServicePlanImageObjectKey: r.MasjidServicePlanImageObjectKey,

		MasjidServicePlanMaxTeachers:  r.MasjidServicePlanMaxTeachers,
		MasjidServicePlanMaxStudents:  r.MasjidServicePlanMaxStudents,
		MasjidServicePlanMaxStorageMB: r.MasjidServicePlanMaxStorageMB,

		MasjidServicePlanPriceMonthly: r.MasjidServicePlanPriceMonthly,
		MasjidServicePlanPriceYearly:  r.MasjidServicePlanPriceYearly,

		MasjidServicePlanAllowCustomTheme: r.MasjidServicePlanAllowCustomTheme,
		MasjidServicePlanMaxCustomThemes:  r.MasjidServicePlanMaxCustomThemes,
	}

	if r.MasjidServicePlanIsActive != nil {
		m.MasjidServicePlanIsActive = *r.MasjidServicePlanIsActive
	} else {
		m.MasjidServicePlanIsActive = true
	}
	return m
}

/* =========================================================
   UPDATE (PATCH) REQUEST
   - support JSON patch (tri-state)
========================================================= */

type UpdateMasjidServicePlanRequest struct {
	MasjidServicePlanCode        PatchField[string] `json:"masjid_service_plan_code" form:"masjid_service_plan_code"`
	MasjidServicePlanName        PatchField[string] `json:"masjid_service_plan_name" form:"masjid_service_plan_name"`
	MasjidServicePlanDescription PatchField[string] `json:"masjid_service_plan_description" form:"masjid_service_plan_description"`

	MasjidServicePlanImageURL       PatchField[string] `json:"masjid_service_plan_image_url" form:"masjid_service_plan_image_url"`
	MasjidServicePlanImageObjectKey PatchField[string] `json:"masjid_service_plan_image_object_key" form:"masjid_service_plan_image_object_key"`

	MasjidServicePlanMaxTeachers  PatchField[int]     `json:"masjid_service_plan_max_teachers" form:"masjid_service_plan_max_teachers"`
	MasjidServicePlanMaxStudents  PatchField[int]     `json:"masjid_service_plan_max_students" form:"masjid_service_plan_max_students"`
	MasjidServicePlanMaxStorageMB PatchField[int]     `json:"masjid_service_plan_max_storage_mb" form:"masjid_service_plan_max_storage_mb"`
	MasjidServicePlanPriceMonthly PatchField[float64] `json:"masjid_service_plan_price_monthly" form:"masjid_service_plan_price_monthly"`
	MasjidServicePlanPriceYearly  PatchField[float64] `json:"masjid_service_plan_price_yearly" form:"masjid_service_plan_price_yearly"`

	MasjidServicePlanAllowCustomTheme PatchField[bool] `json:"masjid_service_plan_allow_custom_theme" form:"masjid_service_plan_allow_custom_theme"`
	MasjidServicePlanMaxCustomThemes  PatchField[int]  `json:"masjid_service_plan_max_custom_themes" form:"masjid_service_plan_max_custom_themes"`

	MasjidServicePlanIsActive PatchField[bool] `json:"masjid_service_plan_is_active" form:"masjid_service_plan_is_active"`
}

var (
	ErrImagePairMismatch = errors.New("if you patch image_url you must also patch image_object_key (both null or both non-null)")
)

func (r *UpdateMasjidServicePlanRequest) ApplyToModelWithImageSwap(m *mModel.MasjidServicePlan, retention time.Duration) error {
	// scalar & pointer fields
	applyPatchScalar(&m.MasjidServicePlanCode, r.MasjidServicePlanCode)
	applyPatchScalar(&m.MasjidServicePlanName, r.MasjidServicePlanName)
	applyPatchPtr(&m.MasjidServicePlanDescription, r.MasjidServicePlanDescription)

	applyPatchPtr(&m.MasjidServicePlanMaxTeachers, r.MasjidServicePlanMaxTeachers)
	applyPatchPtr(&m.MasjidServicePlanMaxStudents, r.MasjidServicePlanMaxStudents)
	applyPatchPtr(&m.MasjidServicePlanMaxStorageMB, r.MasjidServicePlanMaxStorageMB)

	applyPatchPtr(&m.MasjidServicePlanPriceMonthly, r.MasjidServicePlanPriceMonthly)
	applyPatchPtr(&m.MasjidServicePlanPriceYearly, r.MasjidServicePlanPriceYearly)

	applyPatchScalar(&m.MasjidServicePlanAllowCustomTheme, r.MasjidServicePlanAllowCustomTheme)
	applyPatchPtr(&m.MasjidServicePlanMaxCustomThemes, r.MasjidServicePlanMaxCustomThemes)

	applyPatchScalar(&m.MasjidServicePlanIsActive, r.MasjidServicePlanIsActive)

	// image pair
	imgURLPatched := r.MasjidServicePlanImageURL.Present
	imgKeyPatched := r.MasjidServicePlanImageObjectKey.Present
	if imgURLPatched != imgKeyPatched {
		return ErrImagePairMismatch
	}
	if imgURLPatched && imgKeyPatched {
		newURLPtr := r.MasjidServicePlanImageURL.Value
		newKeyPtr := r.MasjidServicePlanImageObjectKey.Value

		var curURL, curKey string
		if m.MasjidServicePlanImageURL != nil {
			curURL = *m.MasjidServicePlanImageURL
		}
		if m.MasjidServicePlanImageObjectKey != nil {
			curKey = *m.MasjidServicePlanImageObjectKey
		}

		// clear
		if newURLPtr == nil && newKeyPtr == nil {
			if m.MasjidServicePlanImageURL != nil && m.MasjidServicePlanImageObjectKey != nil {
				m.MasjidServicePlanImageURLOld = m.MasjidServicePlanImageURL
				m.MasjidServicePlanImageObjectKeyOld = m.MasjidServicePlanImageObjectKey
				if retention > 0 {
					t := time.Now().Add(retention)
					m.MasjidServicePlanImageDeletePendingUntil = &t
				}
			}
			m.MasjidServicePlanImageURL = nil
			m.MasjidServicePlanImageObjectKey = nil
		} else {
			// set baru
			newURL := *newURLPtr
			newKey := *newKeyPtr
			if newURL != curURL || newKey != curKey {
				if m.MasjidServicePlanImageURL != nil && m.MasjidServicePlanImageObjectKey != nil {
					m.MasjidServicePlanImageURLOld = m.MasjidServicePlanImageURL
					m.MasjidServicePlanImageObjectKeyOld = m.MasjidServicePlanImageObjectKey
					if retention > 0 {
						t := time.Now().Add(retention)
						m.MasjidServicePlanImageDeletePendingUntil = &t
					}
				}
				m.MasjidServicePlanImageURL = &newURL
				m.MasjidServicePlanImageObjectKey = &newKey
			}
		}
	}

	m.MasjidServicePlanUpdatedAt = time.Now()
	return nil
}

/* =========================================================
   LIST QUERY
========================================================= */

type ListMasjidServicePlanQuery struct {
	Code   *string `query:"code"`
	Name   *string `query:"name"`
	Active *bool   `query:"active"`

	AllowCustomTheme *bool    `query:"allow_custom_theme"`
	PriceMonthlyMin  *float64 `query:"price_monthly_min"`
	PriceMonthlyMax  *float64 `query:"price_monthly_max"`

	Limit  int     `query:"limit" validate:"omitempty,min=1,max=200"`
	Offset int     `query:"offset" validate:"omitempty,min=0"`
	Sort   *string `query:"sort"`
}

/* =========================================================
   RESPONSE
========================================================= */

type MasjidServicePlanResponse struct {
	MasjidServicePlanID uuid.UUID `json:"masjid_service_plan_id"`

	MasjidServicePlanCode        string  `json:"masjid_service_plan_code"`
	MasjidServicePlanName        string  `json:"masjid_service_plan_name"`
	MasjidServicePlanDescription *string `json:"masjid_service_plan_description,omitempty"`

	MasjidServicePlanImageURL                *string    `json:"masjid_service_plan_image_url,omitempty"`
	MasjidServicePlanImageObjectKey          *string    `json:"masjid_service_plan_image_object_key,omitempty"`
	MasjidServicePlanImageURLOld             *string    `json:"masjid_service_plan_image_url_old,omitempty"`
	MasjidServicePlanImageObjectKeyOld       *string    `json:"masjid_service_plan_image_object_key_old,omitempty"`
	MasjidServicePlanImageDeletePendingUntil *time.Time `json:"masjid_service_plan_image_delete_pending_until,omitempty"`

	MasjidServicePlanMaxTeachers  *int     `json:"masjid_service_plan_max_teachers,omitempty"`
	MasjidServicePlanMaxStudents  *int     `json:"masjid_service_plan_max_students,omitempty"`
	MasjidServicePlanMaxStorageMB *int     `json:"masjid_service_plan_max_storage_mb,omitempty"`
	MasjidServicePlanPriceMonthly *float64 `json:"masjid_service_plan_price_monthly,omitempty"`
	MasjidServicePlanPriceYearly  *float64 `json:"masjid_service_plan_price_yearly,omitempty"`

	MasjidServicePlanAllowCustomTheme bool `json:"masjid_service_plan_allow_custom_theme"`
	MasjidServicePlanMaxCustomThemes  *int `json:"masjid_service_plan_max_custom_themes,omitempty"`

	MasjidServicePlanIsActive bool `json:"masjid_service_plan_is_active"`

	MasjidServicePlanCreatedAt time.Time  `json:"masjid_service_plan_created_at"`
	MasjidServicePlanUpdatedAt time.Time  `json:"masjid_service_plan_updated_at"`
	MasjidServicePlanDeletedAt *time.Time `json:"masjid_service_plan_deleted_at,omitempty"`
}

func NewMasjidServicePlanResponse(m *mModel.MasjidServicePlan) *MasjidServicePlanResponse {
	if m == nil {
		return nil
	}
	resp := &MasjidServicePlanResponse{
		MasjidServicePlanID:                      m.MasjidServicePlanID,
		MasjidServicePlanCode:                    m.MasjidServicePlanCode,
		MasjidServicePlanName:                    m.MasjidServicePlanName,
		MasjidServicePlanDescription:             m.MasjidServicePlanDescription,
		MasjidServicePlanImageURL:                m.MasjidServicePlanImageURL,
		MasjidServicePlanImageObjectKey:          m.MasjidServicePlanImageObjectKey,
		MasjidServicePlanImageURLOld:             m.MasjidServicePlanImageURLOld,
		MasjidServicePlanImageObjectKeyOld:       m.MasjidServicePlanImageObjectKeyOld,
		MasjidServicePlanImageDeletePendingUntil: m.MasjidServicePlanImageDeletePendingUntil,
		MasjidServicePlanMaxTeachers:             m.MasjidServicePlanMaxTeachers,
		MasjidServicePlanMaxStudents:             m.MasjidServicePlanMaxStudents,
		MasjidServicePlanMaxStorageMB:            m.MasjidServicePlanMaxStorageMB,
		MasjidServicePlanPriceMonthly:            m.MasjidServicePlanPriceMonthly,
		MasjidServicePlanPriceYearly:             m.MasjidServicePlanPriceYearly,
		MasjidServicePlanAllowCustomTheme:        m.MasjidServicePlanAllowCustomTheme,
		MasjidServicePlanMaxCustomThemes:         m.MasjidServicePlanMaxCustomThemes,
		MasjidServicePlanIsActive:                m.MasjidServicePlanIsActive,
		MasjidServicePlanCreatedAt:               m.MasjidServicePlanCreatedAt,
		MasjidServicePlanUpdatedAt:               m.MasjidServicePlanUpdatedAt,
	}
	if m.MasjidServicePlanDeletedAt.Valid {
		t := m.MasjidServicePlanDeletedAt.Time
		resp.MasjidServicePlanDeletedAt = &t
	}
	return resp
}
