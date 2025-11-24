package dto

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	mModel "madinahsalam_backend/internals/features/lembaga/school_yayasans/schools/model"

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

type CreateSchoolServicePlanRequest struct {
	SchoolServicePlanCode        string  `json:"school_service_plan_code" form:"school_service_plan_code" validate:"required,min=1,max=30,alphanumdash"`
	SchoolServicePlanName        string  `json:"school_service_plan_name" form:"school_service_plan_name" validate:"required,min=2,max=100"`
	SchoolServicePlanDescription *string `json:"school_service_plan_description" form:"school_service_plan_description" validate:"omitempty,max=2000"`

	// Image (current). *_old & *_delete_pending_until dikelola service saat swap.
	SchoolServicePlanImageURL       *string `json:"school_service_plan_image_url" form:"school_service_plan_image_url" validate:"omitempty,url,max=2048"`
	SchoolServicePlanImageObjectKey *string `json:"school_service_plan_image_object_key" form:"school_service_plan_image_object_key" validate:"omitempty,max=512"`

	SchoolServicePlanMaxTeachers  *int     `json:"school_service_plan_max_teachers" form:"school_service_plan_max_teachers" validate:"omitempty,min=0"`
	SchoolServicePlanMaxStudents  *int     `json:"school_service_plan_max_students" form:"school_service_plan_max_students" validate:"omitempty,min=0"`
	SchoolServicePlanMaxStorageMB *int     `json:"school_service_plan_max_storage_mb" form:"school_service_plan_max_storage_mb" validate:"omitempty,min=0"`
	SchoolServicePlanPriceMonthly *float64 `json:"school_service_plan_price_monthly" form:"school_service_plan_price_monthly" validate:"omitempty,gte=0"`
	SchoolServicePlanPriceYearly  *float64 `json:"school_service_plan_price_yearly" form:"school_service_plan_price_yearly" validate:"omitempty,gte=0"`

	SchoolServicePlanAllowCustomTheme bool `json:"school_service_plan_allow_custom_theme" form:"school_service_plan_allow_custom_theme"`
	SchoolServicePlanMaxCustomThemes  *int `json:"school_service_plan_max_custom_themes" form:"school_service_plan_max_custom_themes" validate:"omitempty,min=0"`

	SchoolServicePlanIsActive *bool `json:"school_service_plan_is_active" form:"school_service_plan_is_active" validate:"omitempty"`
}

func (r *CreateSchoolServicePlanRequest) ToModel() *mModel.SchoolServicePlan {
	m := &mModel.SchoolServicePlan{
		SchoolServicePlanCode:        r.SchoolServicePlanCode,
		SchoolServicePlanName:        r.SchoolServicePlanName,
		SchoolServicePlanDescription: r.SchoolServicePlanDescription,

		SchoolServicePlanImageURL:       r.SchoolServicePlanImageURL,
		SchoolServicePlanImageObjectKey: r.SchoolServicePlanImageObjectKey,

		SchoolServicePlanMaxTeachers:  r.SchoolServicePlanMaxTeachers,
		SchoolServicePlanMaxStudents:  r.SchoolServicePlanMaxStudents,
		SchoolServicePlanMaxStorageMB: r.SchoolServicePlanMaxStorageMB,

		SchoolServicePlanPriceMonthly: r.SchoolServicePlanPriceMonthly,
		SchoolServicePlanPriceYearly:  r.SchoolServicePlanPriceYearly,

		SchoolServicePlanAllowCustomTheme: r.SchoolServicePlanAllowCustomTheme,
		SchoolServicePlanMaxCustomThemes:  r.SchoolServicePlanMaxCustomThemes,
	}

	if r.SchoolServicePlanIsActive != nil {
		m.SchoolServicePlanIsActive = *r.SchoolServicePlanIsActive
	} else {
		m.SchoolServicePlanIsActive = true
	}
	return m
}

/* =========================================================
   UPDATE (PATCH) REQUEST
   - support JSON patch (tri-state)
========================================================= */

type UpdateSchoolServicePlanRequest struct {
	SchoolServicePlanCode        PatchField[string] `json:"school_service_plan_code" form:"school_service_plan_code"`
	SchoolServicePlanName        PatchField[string] `json:"school_service_plan_name" form:"school_service_plan_name"`
	SchoolServicePlanDescription PatchField[string] `json:"school_service_plan_description" form:"school_service_plan_description"`

	SchoolServicePlanImageURL       PatchField[string] `json:"school_service_plan_image_url" form:"school_service_plan_image_url"`
	SchoolServicePlanImageObjectKey PatchField[string] `json:"school_service_plan_image_object_key" form:"school_service_plan_image_object_key"`

	SchoolServicePlanMaxTeachers  PatchField[int]     `json:"school_service_plan_max_teachers" form:"school_service_plan_max_teachers"`
	SchoolServicePlanMaxStudents  PatchField[int]     `json:"school_service_plan_max_students" form:"school_service_plan_max_students"`
	SchoolServicePlanMaxStorageMB PatchField[int]     `json:"school_service_plan_max_storage_mb" form:"school_service_plan_max_storage_mb"`
	SchoolServicePlanPriceMonthly PatchField[float64] `json:"school_service_plan_price_monthly" form:"school_service_plan_price_monthly"`
	SchoolServicePlanPriceYearly  PatchField[float64] `json:"school_service_plan_price_yearly" form:"school_service_plan_price_yearly"`

	SchoolServicePlanAllowCustomTheme PatchField[bool] `json:"school_service_plan_allow_custom_theme" form:"school_service_plan_allow_custom_theme"`
	SchoolServicePlanMaxCustomThemes  PatchField[int]  `json:"school_service_plan_max_custom_themes" form:"school_service_plan_max_custom_themes"`

	SchoolServicePlanIsActive PatchField[bool] `json:"school_service_plan_is_active" form:"school_service_plan_is_active"`
}

var (
	ErrImagePairMismatch = errors.New("if you patch image_url you must also patch image_object_key (both null or both non-null)")
)

func (r *UpdateSchoolServicePlanRequest) ApplyToModelWithImageSwap(m *mModel.SchoolServicePlan, retention time.Duration) error {
	// scalar & pointer fields
	applyPatchScalar(&m.SchoolServicePlanCode, r.SchoolServicePlanCode)
	applyPatchScalar(&m.SchoolServicePlanName, r.SchoolServicePlanName)
	applyPatchPtr(&m.SchoolServicePlanDescription, r.SchoolServicePlanDescription)

	applyPatchPtr(&m.SchoolServicePlanMaxTeachers, r.SchoolServicePlanMaxTeachers)
	applyPatchPtr(&m.SchoolServicePlanMaxStudents, r.SchoolServicePlanMaxStudents)
	applyPatchPtr(&m.SchoolServicePlanMaxStorageMB, r.SchoolServicePlanMaxStorageMB)

	applyPatchPtr(&m.SchoolServicePlanPriceMonthly, r.SchoolServicePlanPriceMonthly)
	applyPatchPtr(&m.SchoolServicePlanPriceYearly, r.SchoolServicePlanPriceYearly)

	applyPatchScalar(&m.SchoolServicePlanAllowCustomTheme, r.SchoolServicePlanAllowCustomTheme)
	applyPatchPtr(&m.SchoolServicePlanMaxCustomThemes, r.SchoolServicePlanMaxCustomThemes)

	applyPatchScalar(&m.SchoolServicePlanIsActive, r.SchoolServicePlanIsActive)

	// image pair
	imgURLPatched := r.SchoolServicePlanImageURL.Present
	imgKeyPatched := r.SchoolServicePlanImageObjectKey.Present
	if imgURLPatched != imgKeyPatched {
		return ErrImagePairMismatch
	}
	if imgURLPatched && imgKeyPatched {
		newURLPtr := r.SchoolServicePlanImageURL.Value
		newKeyPtr := r.SchoolServicePlanImageObjectKey.Value

		var curURL, curKey string
		if m.SchoolServicePlanImageURL != nil {
			curURL = *m.SchoolServicePlanImageURL
		}
		if m.SchoolServicePlanImageObjectKey != nil {
			curKey = *m.SchoolServicePlanImageObjectKey
		}

		// clear
		if newURLPtr == nil && newKeyPtr == nil {
			if m.SchoolServicePlanImageURL != nil && m.SchoolServicePlanImageObjectKey != nil {
				m.SchoolServicePlanImageURLOld = m.SchoolServicePlanImageURL
				m.SchoolServicePlanImageObjectKeyOld = m.SchoolServicePlanImageObjectKey
				if retention > 0 {
					t := time.Now().Add(retention)
					m.SchoolServicePlanImageDeletePendingUntil = &t
				}
			}
			m.SchoolServicePlanImageURL = nil
			m.SchoolServicePlanImageObjectKey = nil
		} else {
			// set baru
			newURL := *newURLPtr
			newKey := *newKeyPtr
			if newURL != curURL || newKey != curKey {
				if m.SchoolServicePlanImageURL != nil && m.SchoolServicePlanImageObjectKey != nil {
					m.SchoolServicePlanImageURLOld = m.SchoolServicePlanImageURL
					m.SchoolServicePlanImageObjectKeyOld = m.SchoolServicePlanImageObjectKey
					if retention > 0 {
						t := time.Now().Add(retention)
						m.SchoolServicePlanImageDeletePendingUntil = &t
					}
				}
				m.SchoolServicePlanImageURL = &newURL
				m.SchoolServicePlanImageObjectKey = &newKey
			}
		}
	}

	m.SchoolServicePlanUpdatedAt = time.Now()
	return nil
}

/* =========================================================
   LIST QUERY
========================================================= */

type ListSchoolServicePlanQuery struct {
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

type SchoolServicePlanResponse struct {
	SchoolServicePlanID uuid.UUID `json:"school_service_plan_id"`

	SchoolServicePlanCode        string  `json:"school_service_plan_code"`
	SchoolServicePlanName        string  `json:"school_service_plan_name"`
	SchoolServicePlanDescription *string `json:"school_service_plan_description,omitempty"`

	SchoolServicePlanImageURL                *string    `json:"school_service_plan_image_url,omitempty"`
	SchoolServicePlanImageObjectKey          *string    `json:"school_service_plan_image_object_key,omitempty"`
	SchoolServicePlanImageURLOld             *string    `json:"school_service_plan_image_url_old,omitempty"`
	SchoolServicePlanImageObjectKeyOld       *string    `json:"school_service_plan_image_object_key_old,omitempty"`
	SchoolServicePlanImageDeletePendingUntil *time.Time `json:"school_service_plan_image_delete_pending_until,omitempty"`

	SchoolServicePlanMaxTeachers  *int     `json:"school_service_plan_max_teachers,omitempty"`
	SchoolServicePlanMaxStudents  *int     `json:"school_service_plan_max_students,omitempty"`
	SchoolServicePlanMaxStorageMB *int     `json:"school_service_plan_max_storage_mb,omitempty"`
	SchoolServicePlanPriceMonthly *float64 `json:"school_service_plan_price_monthly,omitempty"`
	SchoolServicePlanPriceYearly  *float64 `json:"school_service_plan_price_yearly,omitempty"`

	SchoolServicePlanAllowCustomTheme bool `json:"school_service_plan_allow_custom_theme"`
	SchoolServicePlanMaxCustomThemes  *int `json:"school_service_plan_max_custom_themes,omitempty"`

	SchoolServicePlanIsActive bool `json:"school_service_plan_is_active"`

	SchoolServicePlanCreatedAt time.Time  `json:"school_service_plan_created_at"`
	SchoolServicePlanUpdatedAt time.Time  `json:"school_service_plan_updated_at"`
	SchoolServicePlanDeletedAt *time.Time `json:"school_service_plan_deleted_at,omitempty"`
}

func NewSchoolServicePlanResponse(m *mModel.SchoolServicePlan) *SchoolServicePlanResponse {
	if m == nil {
		return nil
	}
	resp := &SchoolServicePlanResponse{
		SchoolServicePlanID:                      m.SchoolServicePlanID,
		SchoolServicePlanCode:                    m.SchoolServicePlanCode,
		SchoolServicePlanName:                    m.SchoolServicePlanName,
		SchoolServicePlanDescription:             m.SchoolServicePlanDescription,
		SchoolServicePlanImageURL:                m.SchoolServicePlanImageURL,
		SchoolServicePlanImageObjectKey:          m.SchoolServicePlanImageObjectKey,
		SchoolServicePlanImageURLOld:             m.SchoolServicePlanImageURLOld,
		SchoolServicePlanImageObjectKeyOld:       m.SchoolServicePlanImageObjectKeyOld,
		SchoolServicePlanImageDeletePendingUntil: m.SchoolServicePlanImageDeletePendingUntil,
		SchoolServicePlanMaxTeachers:             m.SchoolServicePlanMaxTeachers,
		SchoolServicePlanMaxStudents:             m.SchoolServicePlanMaxStudents,
		SchoolServicePlanMaxStorageMB:            m.SchoolServicePlanMaxStorageMB,
		SchoolServicePlanPriceMonthly:            m.SchoolServicePlanPriceMonthly,
		SchoolServicePlanPriceYearly:             m.SchoolServicePlanPriceYearly,
		SchoolServicePlanAllowCustomTheme:        m.SchoolServicePlanAllowCustomTheme,
		SchoolServicePlanMaxCustomThemes:         m.SchoolServicePlanMaxCustomThemes,
		SchoolServicePlanIsActive:                m.SchoolServicePlanIsActive,
		SchoolServicePlanCreatedAt:               m.SchoolServicePlanCreatedAt,
		SchoolServicePlanUpdatedAt:               m.SchoolServicePlanUpdatedAt,
	}
	if m.SchoolServicePlanDeletedAt.Valid {
		t := m.SchoolServicePlanDeletedAt.Time
		resp.SchoolServicePlanDeletedAt = &t
	}
	return resp
}
