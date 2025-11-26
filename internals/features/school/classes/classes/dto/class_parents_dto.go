package dto

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/datatypes"

	// ⬅️ Perbaikan penting
	m "madinahsalam_backend/internals/features/school/classes/classes/model"
)

//
// =========================================================
// JSONMapFlexible — untuk menangani input JSON body / form-data
// =========================================================
//

type JSONMapFlexible datatypes.JSONMap

func (m *JSONMapFlexible) UnmarshalJSON(b []byte) error {
	bs := strings.TrimSpace(string(b))
	if len(bs) > 0 && bs[0] == '"' {
		var raw string
		if err := json.Unmarshal(b, &raw); err != nil {
			return err
		}
		if strings.TrimSpace(raw) == "" {
			*m = JSONMapFlexible{}
			return nil
		}
		tmp := datatypes.JSONMap{}
		if err := json.Unmarshal([]byte(raw), &tmp); err != nil {
			return err
		}
		*m = JSONMapFlexible(tmp)
		return nil
	}
	tmp := datatypes.JSONMap{}
	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}
	*m = JSONMapFlexible(tmp)
	return nil
}

func (m *JSONMapFlexible) UnmarshalText(text []byte) error {
	s := strings.TrimSpace(string(text))
	if s == "" {
		*m = JSONMapFlexible{}
		return nil
	}
	tmp := datatypes.JSONMap{}
	if err := json.Unmarshal([]byte(s), &tmp); err != nil {
		return err
	}
	*m = JSONMapFlexible(tmp)
	return nil
}

func (m JSONMapFlexible) ToJSONMap() datatypes.JSONMap {
	return datatypes.JSONMap(m)
}

//
// =========================================================
// PATCH FIELD (tri-state)
// =========================================================
//

type PatchFieldClassParent[T any] struct {
	Present bool
	Value   *T
}

func (p *PatchFieldClassParent[T]) UnmarshalJSON(b []byte) error {
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

func (p PatchFieldClassParent[T]) Get() (*T, bool) { return p.Value, p.Present }

//
// =========================================================
// CREATE REQUEST
// =========================================================
//

type ClassParentCreateRequest struct {
	ClassParentSchoolID uuid.UUID `json:"class_parent_school_id" form:"class_parent_school_id" validate:"required"`
	ClassParentName     string    `json:"class_parent_name" form:"class_parent_name" validate:"required,min=1,max=120"`

	ClassParentCode         *string         `json:"class_parent_code" form:"class_parent_code" validate:"omitempty,max=40"`
	ClassParentSlug         *string         `json:"class_parent_slug" form:"class_parent_slug" validate:"omitempty,max=160"`
	ClassParentDescription  *string         `json:"class_parent_description" form:"class_parent_description"`
	ClassParentLevel        *int16          `json:"class_parent_level" form:"class_parent_level" validate:"omitempty,min=0,max=100"`
	ClassParentIsActive     *bool           `json:"class_parent_is_active" form:"class_parent_is_active"`
	ClassParentRequirements JSONMapFlexible `json:"class_parent_requirements" form:"class_parent_requirements"`

	// Gambar opsional
	ClassParentImageURL       *string `json:"class_parent_image_url" form:"class_parent_image_url"`
	ClassParentImageObjectKey *string `json:"class_parent_image_object_key" form:"class_parent_image_object_key"`
}

func (r *ClassParentCreateRequest) Normalize() {
	trim := func(p **string, lower bool) {
		if p == nil || *p == nil {
			return
		}
		v := strings.TrimSpace(**p)
		if v == "" {
			*p = nil
			return
		}
		if lower {
			v = strings.ToLower(v)
		}
		*p = &v
	}

	trim(&r.ClassParentCode, false)
	trim(&r.ClassParentSlug, true)
	trim(&r.ClassParentDescription, false)
	trim(&r.ClassParentImageURL, false)
	trim(&r.ClassParentImageObjectKey, false)

	r.ClassParentName = strings.TrimSpace(r.ClassParentName)
}

func (r ClassParentCreateRequest) ToModel() *m.ClassParentModel {
	now := time.Now()

	reqMap := r.ClassParentRequirements.ToJSONMap()
	if reqMap == nil {
		reqMap = datatypes.JSONMap{}
	}

	cp := &m.ClassParentModel{
		ClassParentSchoolID:     r.ClassParentSchoolID,
		ClassParentName:         r.ClassParentName,
		ClassParentCode:         r.ClassParentCode,
		ClassParentSlug:         r.ClassParentSlug,
		ClassParentDescription:  r.ClassParentDescription,
		ClassParentLevel:        r.ClassParentLevel,
		ClassParentRequirements: reqMap,
		ClassParentCreatedAt:    now,
		ClassParentUpdatedAt:    now,
	}

	if r.ClassParentIsActive != nil {
		cp.ClassParentIsActive = *r.ClassParentIsActive
	} else {
		cp.ClassParentIsActive = true
	}

	cp.ClassParentImageURL = r.ClassParentImageURL
	cp.ClassParentImageObjectKey = r.ClassParentImageObjectKey

	return cp
}

//
// =========================================================
// RESPONSE
// =========================================================
//

type ClassParentResponse struct {
	ClassParentID          uuid.UUID `json:"class_parent_id"`
	ClassParentSchoolID    uuid.UUID `json:"class_parent_school_id"`
	ClassParentName        string    `json:"class_parent_name"`
	ClassParentCode        *string   `json:"class_parent_code"`
	ClassParentSlug        *string   `json:"class_parent_slug"`
	ClassParentDescription *string   `json:"class_parent_description"`
	ClassParentLevel       *int16    `json:"class_parent_level"`
	ClassParentIsActive    bool      `json:"class_parent_is_active"`

	// 🔥 Snapshot lengkap
	ClassParentTotalClasses        int32 `json:"class_parent_total_classes"`
	ClassParentTotalClassSections  int32 `json:"class_parent_total_class_sections"`
	ClassParentTotalStudents       int32 `json:"class_parent_total_students"`
	ClassParentTotalMaleStudents   int32 `json:"class_parent_total_male_students"`
	ClassParentTotalFemaleStudents int32 `json:"class_parent_total_female_students"`
	ClassParentTotalTeachers       int32 `json:"class_parent_total_teachers"`

	ClassParentRequirements            datatypes.JSONMap `json:"class_parent_requirements"`
	ClassParentImageURL                *string           `json:"class_parent_image_url"`
	ClassParentImageObjectKey          *string           `json:"class_parent_image_object_key"`
	ClassParentImageURLOld             *string           `json:"class_parent_image_url_old"`
	ClassParentImageObjectKeyOld       *string           `json:"class_parent_image_object_key_old"`
	ClassParentImageDeletePendingUntil *time.Time        `json:"class_parent_image_delete_pending_until"`

	ClassParentCreatedAt time.Time  `json:"class_parent_created_at"`
	ClassParentUpdatedAt time.Time  `json:"class_parent_updated_at"`
	ClassParentDeletedAt *time.Time `json:"class_parent_deleted_at,omitempty"`
}

func FromModelClassParent(cp *m.ClassParentModel) ClassParentResponse {
	var deletedAt *time.Time
	if cp.ClassParentDeletedAt.Valid {
		t := cp.ClassParentDeletedAt.Time
		deletedAt = &t
	}
	return ClassParentResponse{
		ClassParentID:          cp.ClassParentID,
		ClassParentSchoolID:    cp.ClassParentSchoolID,
		ClassParentName:        cp.ClassParentName,
		ClassParentCode:        cp.ClassParentCode,
		ClassParentSlug:        cp.ClassParentSlug,
		ClassParentDescription: cp.ClassParentDescription,
		ClassParentLevel:       cp.ClassParentLevel,
		ClassParentIsActive:    cp.ClassParentIsActive,

		// 🔥 Snapshot
		ClassParentTotalClasses:        cp.ClassParentTotalClasses,
		ClassParentTotalClassSections:  cp.ClassParentTotalClassSections,
		ClassParentTotalStudents:       cp.ClassParentTotalStudents,
		ClassParentTotalMaleStudents:   cp.ClassParentTotalMaleStudents,
		ClassParentTotalFemaleStudents: cp.ClassParentTotalFemaleStudents,
		ClassParentTotalTeachers:       cp.ClassParentTotalTeachers,

		ClassParentRequirements:            cp.ClassParentRequirements,
		ClassParentImageURL:                cp.ClassParentImageURL,
		ClassParentImageObjectKey:          cp.ClassParentImageObjectKey,
		ClassParentImageURLOld:             cp.ClassParentImageURLOld,
		ClassParentImageObjectKeyOld:       cp.ClassParentImageObjectKeyOld,
		ClassParentImageDeletePendingUntil: cp.ClassParentImageDeletePendingUntil,
		ClassParentCreatedAt:               cp.ClassParentCreatedAt,
		ClassParentUpdatedAt:               cp.ClassParentUpdatedAt,
		ClassParentDeletedAt:               deletedAt,
	}
}

//
// =========================================================
// PATCH REQUEST
// =========================================================
//

type ClassParentPatchRequest struct {
	ClassParentName        PatchFieldClassParent[string]  `json:"class_parent_name"`
	ClassParentCode        PatchFieldClassParent[*string] `json:"class_parent_code"`
	ClassParentSlug        PatchFieldClassParent[*string] `json:"class_parent_slug"`
	ClassParentDescription PatchFieldClassParent[*string] `json:"class_parent_description"`
	ClassParentLevel       PatchFieldClassParent[*int16]  `json:"class_parent_level"`
	ClassParentIsActive    PatchFieldClassParent[*bool]   `json:"class_parent_is_active"`

	// hanya total_classes yang boleh di-patch (opsional)
	ClassParentTotalClasses PatchFieldClassParent[*int32] `json:"class_parent_total_classes"`

	ClassParentRequirements PatchFieldClassParent[JSONMapFlexible] `json:"class_parent_requirements"`

	// Image
	ClassParentImageURL                PatchFieldClassParent[*string]    `json:"class_parent_image_url"`
	ClassParentImageObjectKey          PatchFieldClassParent[*string]    `json:"class_parent_image_object_key"`
	ClassParentImageURLOld             PatchFieldClassParent[*string]    `json:"class_parent_image_url_old"`
	ClassParentImageObjectKeyOld       PatchFieldClassParent[*string]    `json:"class_parent_image_object_key_old"`
	ClassParentImageDeletePendingUntil PatchFieldClassParent[*time.Time] `json:"class_parent_image_delete_pending_until"`
}

func (p *ClassParentPatchRequest) Normalize() {
	trim := func(pp **string, lower bool) {
		if pp == nil || *pp == nil {
			return
		}
		v := strings.TrimSpace(**pp)
		if v == "" {
			*pp = nil
			return
		}
		if lower {
			v = strings.ToLower(v)
		}
		*pp = &v
	}

	if p.ClassParentName.Present && p.ClassParentName.Value != nil {
		v := strings.TrimSpace(*p.ClassParentName.Value)
		p.ClassParentName.Value = &v
	}
	if p.ClassParentCode.Present {
		trim(p.ClassParentCode.Value, false)
	}
	if p.ClassParentSlug.Present {
		trim(p.ClassParentSlug.Value, true)
	}
	if p.ClassParentDescription.Present {
		trim(p.ClassParentDescription.Value, false)
	}
	if p.ClassParentImageURL.Present {
		trim(p.ClassParentImageURL.Value, false)
	}
	if p.ClassParentImageObjectKey.Present {
		trim(p.ClassParentImageObjectKey.Value, false)
	}
	if p.ClassParentImageURLOld.Present {
		trim(p.ClassParentImageURLOld.Value, false)
	}
	if p.ClassParentImageObjectKeyOld.Present {
		trim(p.ClassParentImageObjectKeyOld.Value, false)
	}
}

func (p ClassParentPatchRequest) Apply(cp *m.ClassParentModel) {
	if p.ClassParentName.Present && p.ClassParentName.Value != nil {
		cp.ClassParentName = *p.ClassParentName.Value
	}
	if p.ClassParentCode.Present {
		if p.ClassParentCode.Value == nil {
			cp.ClassParentCode = nil
		} else {
			cp.ClassParentCode = *p.ClassParentCode.Value
		}
	}
	if p.ClassParentSlug.Present {
		if p.ClassParentSlug.Value == nil {
			cp.ClassParentSlug = nil
		} else {
			cp.ClassParentSlug = *p.ClassParentSlug.Value
		}
	}
	if p.ClassParentDescription.Present {
		if p.ClassParentDescription.Value == nil {
			cp.ClassParentDescription = nil
		} else {
			cp.ClassParentDescription = *p.ClassParentDescription.Value
		}
	}
	if p.ClassParentLevel.Present {
		if p.ClassParentLevel.Value == nil {
			cp.ClassParentLevel = nil
		} else {
			cp.ClassParentLevel = *p.ClassParentLevel.Value
		}
	}
	if p.ClassParentIsActive.Present && p.ClassParentIsActive.Value != nil {
		cp.ClassParentIsActive = **p.ClassParentIsActive.Value
	}

	// total_classes saja yang boleh di-patch
	if p.ClassParentTotalClasses.Present {
		if p.ClassParentTotalClasses.Value == nil {
			cp.ClassParentTotalClasses = 0
		} else {
			cp.ClassParentTotalClasses = **p.ClassParentTotalClasses.Value
		}
	}

	if p.ClassParentRequirements.Present {
		if p.ClassParentRequirements.Value == nil {
			cp.ClassParentRequirements = datatypes.JSONMap{}
		} else {
			cp.ClassParentRequirements = p.ClassParentRequirements.Value.ToJSONMap()
			if cp.ClassParentRequirements == nil {
				cp.ClassParentRequirements = datatypes.JSONMap{}
			}
		}
	}

	if p.ClassParentImageURL.Present {
		if p.ClassParentImageURL.Value == nil {
			cp.ClassParentImageURL = nil
		} else {
			cp.ClassParentImageURL = *p.ClassParentImageURL.Value
		}
	}
	if p.ClassParentImageObjectKey.Present {
		if p.ClassParentImageObjectKey.Value == nil {
			cp.ClassParentImageObjectKey = nil
		} else {
			cp.ClassParentImageObjectKey = *p.ClassParentImageObjectKey.Value
		}
	}
	if p.ClassParentImageURLOld.Present {
		if p.ClassParentImageURLOld.Value == nil {
			cp.ClassParentImageURLOld = nil
		} else {
			cp.ClassParentImageURLOld = *p.ClassParentImageURLOld.Value
		}
	}
	if p.ClassParentImageObjectKeyOld.Present {
		if p.ClassParentImageObjectKeyOld.Value == nil {
			cp.ClassParentImageObjectKeyOld = nil
		} else {
			cp.ClassParentImageObjectKeyOld = *p.ClassParentImageObjectKeyOld.Value
		}
	}
	if p.ClassParentImageDeletePendingUntil.Present {
		if p.ClassParentImageDeletePendingUntil.Value == nil {
			cp.ClassParentImageDeletePendingUntil = nil
		} else {
			cp.ClassParentImageDeletePendingUntil = *p.ClassParentImageDeletePendingUntil.Value
		}
	}

	cp.ClassParentUpdatedAt = time.Now()
}

//
// =========================================================
// LIST QUERY + HELPERS
// =========================================================
//

type ListClassParentQuery struct {
	Limit     int        `query:"limit"`
	Offset    int        `query:"offset"`
	Q         string     `query:"q"`
	Name      string     `query:"name"`
	Active    *bool      `query:"active"`
	LevelMin  *int16     `query:"level_min"`
	LevelMax  *int16     `query:"level_max"`
	CreatedGt *time.Time `query:"created_gt"`
	CreatedLt *time.Time `query:"created_lt"`
}

func ToClassParentResponses(rows []m.ClassParentModel) []ClassParentResponse {
	out := make([]ClassParentResponse, 0, len(rows))
	for i := range rows {
		out = append(out, FromModelClassParent(&rows[i]))
	}
	return out
}

type PaginationMeta struct {
	Total      int64 `json:"total"`
	Limit      int   `json:"limit"`
	Offset     int   `json:"offset"`
	Count      int   `json:"count"`
	NextOffset *int  `json:"next_offset,omitempty"`
	PrevOffset *int  `json:"prev_offset,omitempty"`
	HasMore    bool  `json:"has_more"`
}

func NewPaginationMeta(total int64, limit, offset, count int) PaginationMeta {
	if limit <= 0 {
		limit = 20
	}
	meta := PaginationMeta{
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		Count:   count,
		HasMore: int64(offset+count) < total,
	}
	if offset > 0 {
		prev := offset - limit
		if prev < 0 {
			prev = 0
		}
		meta.PrevOffset = &prev
	}
	if meta.HasMore {
		next := offset + count
		meta.NextOffset = &next
	}
	return meta
}

// Decode untuk PATCH
func DecodePatchClassParentFromRequest(c *fiber.Ctx, out *ClassParentPatchRequest) error {
	ct := strings.ToLower(c.Get(fiber.HeaderContentType))
	if strings.Contains(ct, "multipart/form-data") {
		if s := strings.TrimSpace(c.FormValue("payload")); s != "" {
			if err := json.Unmarshal([]byte(s), out); err != nil {
				return err
			}
		} else {
			if err := decodePatchClassParentMultipart(c, out); err != nil {
				return err
			}
		}
	} else {
		if err := c.BodyParser(out); err != nil {
			return err
		}
	}
	out.Normalize()
	return nil
}

func decodePatchClassParentMultipart(c *fiber.Ctx, r *ClassParentPatchRequest) error {
	form, err := c.MultipartForm()
	if err != nil || form == nil {
		return fiber.NewError(fiber.StatusBadRequest, "form-data tidak ditemukan")
	}

	get := func(k string) (string, bool) {
		if vs, ok := form.Value[k]; ok {
			if len(vs) == 0 {
				return "", true
			}
			return vs[0], true
		}
		return "", false
	}

	setStr := func(f *PatchFieldClassParent[string], key string) {
		if v, ok := get(key); ok {
			f.Present = true
			v = strings.TrimSpace(v)
			f.Value = &v
		}
	}

	setStrPtr := func(f *PatchFieldClassParent[*string], key string, lower bool) {
		if v, ok := get(key); ok {
			f.Present = true
			v = strings.TrimSpace(v)
			if v == "" {
				f.Value = nil
			} else {
				if lower {
					v = strings.ToLower(v)
				}
				ptr := &v
				f.Value = &ptr
			}
		}
	}

	setInt16Ptr := func(f *PatchFieldClassParent[*int16], key string) error {
		if v, ok := get(key); ok {
			f.Present = true
			if strings.TrimSpace(v) == "" {
				f.Value = nil
				return nil
			}
			x, err := strconv.ParseInt(strings.TrimSpace(v), 10, 16)
			if err != nil {
				return fiber.NewError(fiber.StatusBadRequest, key+" harus int16")
			}
			tmp := int16(x)
			pp := new(*int16)
			*pp = &tmp
			f.Value = pp
		}
		return nil
	}

	setBoolPtr := func(f *PatchFieldClassParent[*bool], key string) error {
		if v, ok := get(key); ok {
			f.Present = true
			if strings.TrimSpace(v) == "" {
				f.Value = nil
				return nil
			}
			x, err := strconv.ParseBool(strings.TrimSpace(v))
			if err != nil {
				return fiber.NewError(fiber.StatusBadRequest, key+" harus boolean")
			}
			pp := new(*bool)
			*pp = &x
			f.Value = pp
		}
		return nil
	}

	setInt32Ptr := func(f *PatchFieldClassParent[*int32], key string) error {
		if v, ok := get(key); ok {
			f.Present = true
			if strings.TrimSpace(v) == "" {
				f.Value = nil
				return nil
			}
			x, err := strconv.ParseInt(strings.TrimSpace(v), 10, 32)
			if err != nil {
				return fiber.NewError(fiber.StatusBadRequest, key+" harus int32")
			}
			tmp := int32(x)
			pp := new(*int32)
			*pp = &tmp
			f.Value = pp
		}
		return nil
	}

	setTimePtr := func(f *PatchFieldClassParent[*time.Time], key string) error {
		if v, ok := get(key); ok {
			f.Present = true
			s := strings.TrimSpace(v)
			if s == "" {
				f.Value = nil
				return nil
			}
			var t time.Time
			var e error
			if t, e = time.Parse(time.RFC3339, s); e != nil {
				if t, e = time.Parse("2006-01-02", s); e != nil {
					return fiber.NewError(fiber.StatusBadRequest, key+" format invalid (RFC3339 atau YYYY-MM-DD)")
				}
			}
			pp := new(*time.Time)
			*pp = &t
			f.Value = pp
		}
		return nil
	}

	setJSONMapFlexible := func(f *PatchFieldClassParent[JSONMapFlexible], key string) error {
		if v, ok := get(key); ok {
			f.Present = true
			s := strings.TrimSpace(v)
			if s == "" {
				empty := JSONMapFlexible(datatypes.JSONMap{})
				f.Value = &empty
				return nil
			}
			tmp := datatypes.JSONMap{}
			if err := json.Unmarshal([]byte(s), &tmp); err != nil {
				return fiber.NewError(fiber.StatusBadRequest, key+" harus JSON valid")
			}
			val := JSONMapFlexible(tmp)
			f.Value = &val
		}
		return nil
	}

	setStr(&r.ClassParentName, "class_parent_name")
	if err := setInt16Ptr(&r.ClassParentLevel, "class_parent_level"); err != nil {
		return err
	}
	if err := setBoolPtr(&r.ClassParentIsActive, "class_parent_is_active"); err != nil {
		return err
	}
	if err := setInt32Ptr(&r.ClassParentTotalClasses, "class_parent_total_classes"); err != nil {
		return err
	}

	setStrPtr(&r.ClassParentCode, "class_parent_code", false)
	setStrPtr(&r.ClassParentSlug, "class_parent_slug", true)
	setStrPtr(&r.ClassParentDescription, "class_parent_description", false)

	if err := setJSONMapFlexible(&r.ClassParentRequirements, "class_parent_requirements"); err != nil {
		return err
	}

	setStrPtr(&r.ClassParentImageURL, "class_parent_image_url", false)
	setStrPtr(&r.ClassParentImageObjectKey, "class_parent_image_object_key", false)
	setStrPtr(&r.ClassParentImageURLOld, "class_parent_image_url_old", false)
	setStrPtr(&r.ClassParentImageObjectKeyOld, "class_parent_image_object_key_old", false)

	if err := setTimePtr(&r.ClassParentImageDeletePendingUntil, "class_parent_image_delete_pending_until"); err != nil {
		return err
	}

	return nil
}
