// file: internals/features/school/subjects/dto/subject_dto.go
package dto

import (
	"encoding/json"
	"errors"
	"mime/multipart"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	m "masjidku_backend/internals/features/school/academics/subjects/model"
	helper "masjidku_backend/internals/helpers"
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
	if err := jsonUnmarshal(b, &v); err != nil {
		return err
	}
	p.Value = &v
	return nil
}

func (p PatchField[T]) Get() (*T, bool) { return p.Value, p.Present }

/* =========================================================
   CREATE
   ========================================================= */

type CreateSubjectRequest struct {
	MasjidID uuid.UUID `json:"subject_masjid_id" form:"subject_masjid_id" validate:"required"`

	Code string `json:"subject_code" form:"subject_code" validate:"required,min=1,max=40"`
	Name string `json:"subject_name" form:"subject_name" validate:"required,min=1,max=120"`

	Desc *string `json:"subject_desc" form:"subject_desc"`

	// Slug: NOT NULL di DB — controller biasanya auto-generate
	Slug *string `json:"subject_slug" form:"subject_slug" validate:"omitempty,min=1,max=160"`

	IsActive *bool `json:"subject_is_active" form:"subject_is_active"`

	ImageURL                *string    `json:"subject_image_url"                 form:"subject_image_url"`
	ImageObjectKey          *string    `json:"subject_image_object_key"          form:"subject_image_object_key"`
	ImageURLOld             *string    `json:"subject_image_url_old"             form:"subject_image_url_old"`
	ImageObjectKeyOld       *string    `json:"subject_image_object_key_old"      form:"subject_image_object_key_old"`
	ImageDeletePendingUntil *time.Time `json:"subject_image_delete_pending_until" form:"subject_image_delete_pending_until"`
}

func (r *CreateSubjectRequest) Normalize() {
	trimPtr := func(pp **string, lower bool) {
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

	r.Code = strings.TrimSpace(r.Code)
	r.Name = strings.TrimSpace(r.Name)
	trimPtr(&r.Desc, false)

	// slug → sanitasi pakai Slugify(…, 160)
	if r.Slug != nil {
		s := helper.Slugify(strings.TrimSpace(*r.Slug), 160)
		if s == "" {
			r.Slug = nil
		} else {
			r.Slug = &s
		}
	}

	trimPtr(&r.ImageURL, false)
	trimPtr(&r.ImageObjectKey, false)
	trimPtr(&r.ImageURLOld, false)
	trimPtr(&r.ImageObjectKeyOld, false)
}

func (r CreateSubjectRequest) ToModel() m.SubjectModel {
	now := time.Now()

	// pastikan slug terisi (fallback dari name) — pakai Slugify
	slug := ""
	if r.Slug != nil && strings.TrimSpace(*r.Slug) != "" {
		slug = helper.Slugify(*r.Slug, 160)
	} else {
		slug = helper.Slugify(r.Name, 160)
		if slug == "" {
			slug = "subject"
		}
	}

	mm := m.SubjectModel{
		SubjectMasjidID:                r.MasjidID,
		SubjectCode:                    r.Code,
		SubjectName:                    r.Name,
		SubjectDesc:                    r.Desc,
		SubjectSlug:                    slug,
		SubjectImageURL:                r.ImageURL,
		SubjectImageObjectKey:          r.ImageObjectKey,
		SubjectImageURLOld:             r.ImageURLOld,
		SubjectImageObjectKeyOld:       r.ImageObjectKeyOld,
		SubjectImageDeletePendingUntil: r.ImageDeletePendingUntil,
		SubjectIsActive:                true,
		SubjectCreatedAt:               now,
		SubjectUpdatedAt:               now,
	}
	if r.IsActive != nil {
		mm.SubjectIsActive = *r.IsActive
	}
	return mm
}

// BindMultipartCreate: ambil text fields + file (opsional)
func BindMultipartCreate(c *fiber.Ctx) (CreateSubjectRequest, *multipart.FileHeader, error) {
	var req CreateSubjectRequest

	// text fields
	req.Code = strings.TrimSpace(c.FormValue("subject_code"))
	req.Name = strings.TrimSpace(c.FormValue("subject_name"))

	if v := strings.TrimSpace(c.FormValue("subject_desc")); v != "" {
		req.Desc = &v
	}
	if v := strings.TrimSpace(c.FormValue("subject_slug")); v != "" {
		s := helper.Slugify(v, 160)
		if s != "" {
			req.Slug = &s
		}
	}
	if v := strings.TrimSpace(c.FormValue("subject_is_active")); v != "" {
		if b, err := parseBoolLoose(v); err == nil {
			req.IsActive = &b
		}
	}
	if v := strings.TrimSpace(c.FormValue("subject_masjid_id")); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			req.MasjidID = id
		}
	}

	// image columns (opsional)
	if v := strings.TrimSpace(c.FormValue("subject_image_url")); v != "" {
		req.ImageURL = &v
	}
	if v := strings.TrimSpace(c.FormValue("subject_image_object_key")); v != "" {
		req.ImageObjectKey = &v
	}
	if v := strings.TrimSpace(c.FormValue("subject_image_url_old")); v != "" {
		req.ImageURLOld = &v
	}
	if v := strings.TrimSpace(c.FormValue("subject_image_object_key_old")); v != "" {
		req.ImageObjectKeyOld = &v
	}
	if v := strings.TrimSpace(c.FormValue("subject_image_delete_pending_until")); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			req.ImageDeletePendingUntil = &t
		}
	}

	// file
	var fh *multipart.FileHeader
	if f, err := c.FormFile("image"); err == nil && f != nil {
		fh = f
	} else if f2, err2 := c.FormFile("file"); err2 == nil && f2 != nil {
		fh = f2
	}

	return req, fh, nil
}

/* =========================================================
   UPDATE (PATCH) — tri-state
   ========================================================= */

type UpdateSubjectRequest struct {
	MasjidID *uuid.UUID `json:"subject_masjid_id" form:"subject_masjid_id"`

	Code     PatchField[string]  `json:"subject_code"`
	Name     PatchField[string]  `json:"subject_name"`
	Desc     PatchField[*string] `json:"subject_desc"`
	Slug     PatchField[string]  `json:"subject_slug"`
	IsActive PatchField[bool]    `json:"subject_is_active"`

	ImageURL                PatchField[*string]    `json:"subject_image_url"`
	ImageObjectKey          PatchField[*string]    `json:"subject_image_object_key"`
	ImageURLOld             PatchField[*string]    `json:"subject_image_url_old"`
	ImageObjectKeyOld       PatchField[*string]    `json:"subject_image_object_key_old"`
	ImageDeletePendingUntil PatchField[*time.Time] `json:"subject_image_delete_pending_until"`
}

func (p *UpdateSubjectRequest) Normalize() {
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

	if p.Code.Present && p.Code.Value != nil {
		v := strings.TrimSpace(*p.Code.Value)
		p.Code.Value = &v
	}
	if p.Name.Present && p.Name.Value != nil {
		v := strings.TrimSpace(*p.Name.Value)
		p.Name.Value = &v
	}
	// slug → sanitasi pakai Slugify(…, 160)
	if p.Slug.Present && p.Slug.Value != nil {
		v := helper.Slugify(strings.TrimSpace(*p.Slug.Value), 160)
		p.Slug.Value = &v
	}

	if p.Desc.Present {
		trim(p.Desc.Value, false)
	}
	if p.ImageURL.Present {
		trim(p.ImageURL.Value, false)
	}
	if p.ImageObjectKey.Present {
		trim(p.ImageObjectKey.Value, false)
	}
	if p.ImageURLOld.Present {
		trim(p.ImageURLOld.Value, false)
	}
	if p.ImageObjectKeyOld.Present {
		trim(p.ImageObjectKeyOld.Value, false)
	}
}

func (p UpdateSubjectRequest) Apply(mo *m.SubjectModel) {
	if p.MasjidID != nil {
		mo.SubjectMasjidID = *p.MasjidID
	}

	// scalar
	if p.Code.Present && p.Code.Value != nil {
		mo.SubjectCode = *p.Code.Value
	}
	if p.Name.Present && p.Name.Value != nil {
		mo.SubjectName = *p.Name.Value
	}
	if p.Slug.Present && p.Slug.Value != nil {
		mo.SubjectSlug = *p.Slug.Value
	}
	if p.IsActive.Present && p.IsActive.Value != nil {
		mo.SubjectIsActive = *p.IsActive.Value
	}

	// nullable string
	if p.Desc.Present {
		if p.Desc.Value == nil {
			mo.SubjectDesc = nil
		} else {
			mo.SubjectDesc = *p.Desc.Value
		}
	}
	if p.ImageURL.Present {
		if p.ImageURL.Value == nil {
			mo.SubjectImageURL = nil
		} else {
			mo.SubjectImageURL = *p.ImageURL.Value
		}
	}
	if p.ImageObjectKey.Present {
		if p.ImageObjectKey.Value == nil {
			mo.SubjectImageObjectKey = nil
		} else {
			mo.SubjectImageObjectKey = *p.ImageObjectKey.Value
		}
	}
	if p.ImageURLOld.Present {
		if p.ImageURLOld.Value == nil {
			mo.SubjectImageURLOld = nil
		} else {
			mo.SubjectImageURLOld = *p.ImageURLOld.Value
		}
	}
	if p.ImageObjectKeyOld.Present {
		if p.ImageObjectKeyOld.Value == nil {
			mo.SubjectImageObjectKeyOld = nil
		} else {
			mo.SubjectImageObjectKeyOld = *p.ImageObjectKeyOld.Value
		}
	}

	// nullable time
	if p.ImageDeletePendingUntil.Present {
		if p.ImageDeletePendingUntil.Value == nil {
			mo.SubjectImageDeletePendingUntil = nil
		} else {
			mo.SubjectImageDeletePendingUntil = *p.ImageDeletePendingUntil.Value
		}
	}

	mo.SubjectUpdatedAt = time.Now()
}

// BindMultipartPatch: baca multipart form → set tri-state
func BindMultipartPatch(c *fiber.Ctx) (UpdateSubjectRequest, *multipart.FileHeader, error) {
	var req UpdateSubjectRequest

	form, err := c.MultipartForm()
	if err != nil {
		return req, nil, fiber.NewError(fiber.StatusBadRequest, "Multipart tidak valid")
	}
	vals := form.Value

	// --- helpers ---
	has := func(k string) bool {
		_, ok := vals[k]
		if ok {
			return true
		}
		return strings.TrimSpace(c.FormValue(k)) != ""
	}
	get := func(k string) string { return strings.TrimSpace(c.FormValue(k)) }
	isNull := func(k string) bool {
		v := strings.ToLower(strings.TrimSpace(c.FormValue(k + "__null")))
		return v == "1" || v == "true" || v == "yes" || v == "on"
	}
	setPtrStr := func(dst **string, s string) {
		v := s
		*dst = &v
	}
	setPtrTime := func(dst **time.Time, t time.Time) {
		v := t
		*dst = &v
	}

	// masjid_id (opsional; biasanya di-force di controller)
	if has("subject_masjid_id") {
		if id, err := uuid.Parse(get("subject_masjid_id")); err == nil {
			req.MasjidID = &id
		}
	}

	// scalar strings
	if has("subject_code") {
		req.Code.Present = true
		v := get("subject_code")
		req.Code.Value = &v
	}
	if has("subject_name") {
		req.Name.Present = true
		v := get("subject_name")
		req.Name.Value = &v
	}
	if has("subject_slug") {
		req.Slug.Present = true
		v := helper.Slugify(get("subject_slug"), 160)
		req.Slug.Value = &v
	}

	// desc (nullable via __null)
	if has("subject_desc") || isNull("subject_desc") {
		req.Desc.Present = true
		if isNull("subject_desc") {
			req.Desc.Value = nil
		} else {
			req.Desc.Value = new(*string)
			setPtrStr(req.Desc.Value, get("subject_desc"))
		}
	}

	// is_active
	if has("subject_is_active") {
		req.IsActive.Present = true
		if b, err := parseBoolLoose(get("subject_is_active")); err == nil {
			req.IsActive.Value = &b
		}
	}

	// image columns (nullable)
	if has("subject_image_url") || isNull("subject_image_url") {
		req.ImageURL.Present = true
		if isNull("subject_image_url") {
			req.ImageURL.Value = nil
		} else {
			req.ImageURL.Value = new(*string)
			setPtrStr(req.ImageURL.Value, get("subject_image_url"))
		}
	}
	if has("subject_image_object_key") || isNull("subject_image_object_key") {
		req.ImageObjectKey.Present = true
		if isNull("subject_image_object_key") {
			req.ImageObjectKey.Value = nil
		} else {
			req.ImageObjectKey.Value = new(*string)
			setPtrStr(req.ImageObjectKey.Value, get("subject_image_object_key"))
		}
	}
	if has("subject_image_url_old") || isNull("subject_image_url_old") {
		req.ImageURLOld.Present = true
		if isNull("subject_image_url_old") {
			req.ImageURLOld.Value = nil
		} else {
			req.ImageURLOld.Value = new(*string)
			setPtrStr(req.ImageURLOld.Value, get("subject_image_url_old"))
		}
	}
	if has("subject_image_object_key_old") || isNull("subject_image_object_key_old") {
		req.ImageObjectKeyOld.Present = true
		if isNull("subject_image_object_key_old") {
			req.ImageObjectKeyOld.Value = nil
		} else {
			req.ImageObjectKeyOld.Value = new(*string)
			setPtrStr(req.ImageObjectKeyOld.Value, get("subject_image_object_key_old"))
		}
	}
	if has("subject_image_delete_pending_until") || isNull("subject_image_delete_pending_until") {
		req.ImageDeletePendingUntil.Present = true
		if isNull("subject_image_delete_pending_until") {
			req.ImageDeletePendingUntil.Value = nil
		} else {
			raw := get("subject_image_delete_pending_until")
			if t, err := time.Parse(time.RFC3339, raw); err == nil {
				req.ImageDeletePendingUntil.Value = new(*time.Time)
				setPtrTime(req.ImageDeletePendingUntil.Value, t)
			} else {
				return req, nil, fiber.NewError(fiber.StatusBadRequest,
					"Waktu (RFC3339) tidak valid pada subject_image_delete_pending_until")
			}
		}
	}

	// file (opsional)
	var fh *multipart.FileHeader
	if f, err := c.FormFile("image"); err == nil && f != nil {
		fh = f
	} else if f2, err2 := c.FormFile("file"); err2 == nil && f2 != nil {
		fh = f2
	}

	return req, fh, nil
}

/* =========================================================
   RESPONSE
   ========================================================= */

// helpers…

func parseBoolLoose(s string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "t", "yes", "y", "on":
		return true, nil
	case "0", "false", "f", "no", "n", "off":
		return false, nil
	default:
		_, err := strconv.ParseBool(s)
		if err != nil {
			return false, errors.New("bool tidak valid")
		}
		return false, nil
	}
}

func jsonUnmarshal(b []byte, v any) error { return json.Unmarshal(b, v) }

/* ================= Queries & Responses ================= */

type ListSubjectQuery struct {
	Limit       *int    `query:"limit"`
	Offset      *int    `query:"offset"`
	Q           *string `query:"q"`
	IsActive    *bool   `query:"is_active"`
	WithDeleted *bool   `query:"with_deleted"`
	OrderBy     *string `query:"order_by"` // code|name|created_at|updated_at
	Sort        *string `query:"sort"`     // asc|desc
}

type SubjectResponse struct {
	SubjectID                      uuid.UUID  `json:"subject_id"`
	SubjectMasjidID                uuid.UUID  `json:"subject_masjid_id"`
	SubjectCode                    string     `json:"subject_code"`
	SubjectName                    string     `json:"subject_name"`
	SubjectDesc                    *string    `json:"subject_desc,omitempty"`
	SubjectSlug                    string     `json:"subject_slug"`
	SubjectImageURL                *string    `json:"subject_image_url,omitempty"`
	SubjectImageObjectKey          *string    `json:"subject_image_object_key,omitempty"`
	SubjectImageURLOld             *string    `json:"subject_image_url_old,omitempty"`
	SubjectImageObjectKeyOld       *string    `json:"subject_image_object_key_old,omitempty"`
	SubjectImageDeletePendingUntil *time.Time `json:"subject_image_delete_pending_until,omitempty"`
	SubjectIsActive                bool       `json:"subject_is_active"`
	SubjectCreatedAt               time.Time  `json:"subject_created_at"`
	SubjectUpdatedAt               time.Time  `json:"subject_updated_at"`
	SubjectDeletedAt               *time.Time `json:"subject_deleted_at,omitempty"`
}

func FromSubjectModel(mo m.SubjectModel) SubjectResponse {
	var deletedAt *time.Time
	if mo.SubjectDeletedAt.Valid {
		t := mo.SubjectDeletedAt.Time
		deletedAt = &t
	}
	return SubjectResponse{
		SubjectID:                      mo.SubjectID,
		SubjectMasjidID:                mo.SubjectMasjidID,
		SubjectCode:                    mo.SubjectCode,
		SubjectName:                    mo.SubjectName,
		SubjectDesc:                    mo.SubjectDesc,
		SubjectSlug:                    mo.SubjectSlug,
		SubjectImageURL:                mo.SubjectImageURL,
		SubjectImageObjectKey:          mo.SubjectImageObjectKey,
		SubjectImageURLOld:             mo.SubjectImageURLOld,
		SubjectImageObjectKeyOld:       mo.SubjectImageObjectKeyOld,
		SubjectImageDeletePendingUntil: mo.SubjectImageDeletePendingUntil,
		SubjectIsActive:                mo.SubjectIsActive,
		SubjectCreatedAt:               mo.SubjectCreatedAt,
		SubjectUpdatedAt:               mo.SubjectUpdatedAt,
		SubjectDeletedAt:               deletedAt,
	}
}

func FromSubjectModels(rows []m.SubjectModel) []SubjectResponse {
	out := make([]SubjectResponse, 0, len(rows))
	for i := range rows {
		out = append(out, FromSubjectModel(rows[i]))
	}
	return out
}
