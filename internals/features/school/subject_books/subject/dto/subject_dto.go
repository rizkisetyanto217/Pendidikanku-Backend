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

	m "masjidku_backend/internals/features/school/subject_books/subject/model"
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
	MasjidID uuid.UUID `json:"subjects_masjid_id" form:"subjects_masjid_id" validate:"required"`

	Code string `json:"subjects_code" form:"subjects_code" validate:"required,min=1,max=40"`
	Name string `json:"subjects_name" form:"subjects_name" validate:"required,min=1,max=120"`

	Desc *string `json:"subjects_desc" form:"subjects_desc"`

	// Slug: NOT NULL di DB — controller biasanya auto-generate
	Slug *string `json:"subjects_slug" form:"subjects_slug" validate:"omitempty,min=1,max=160"`

	IsActive *bool `json:"subjects_is_active" form:"subjects_is_active"`

	ImageURL                *string    `json:"subjects_image_url"                 form:"subjects_image_url"`
	ImageObjectKey          *string    `json:"subjects_image_object_key"          form:"subjects_image_object_key"`
	ImageURLOld             *string    `json:"subjects_image_url_old"             form:"subjects_image_url_old"`
	ImageObjectKeyOld       *string    `json:"subjects_image_object_key_old"      form:"subjects_image_object_key_old"`
	ImageDeletePendingUntil *time.Time `json:"subjects_image_delete_pending_until" form:"subjects_image_delete_pending_until"`
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

func (r CreateSubjectRequest) ToModel() m.SubjectsModel {
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

	mm := m.SubjectsModel{
		SubjectsMasjidID:                r.MasjidID,
		SubjectsCode:                    r.Code,
		SubjectsName:                    r.Name,
		SubjectsDesc:                    r.Desc,
		SubjectsSlug:                    slug,
		SubjectsImageURL:                r.ImageURL,
		SubjectsImageObjectKey:          r.ImageObjectKey,
		SubjectsImageURLOld:             r.ImageURLOld,
		SubjectsImageObjectKeyOld:       r.ImageObjectKeyOld,
		SubjectsImageDeletePendingUntil: r.ImageDeletePendingUntil,
		SubjectsIsActive:                true,
		SubjectsCreatedAt:               now,
		SubjectsUpdatedAt:               now,
	}
	if r.IsActive != nil {
		mm.SubjectsIsActive = *r.IsActive
	}
	return mm
}

// BindMultipartCreate: ambil text fields + file (opsional)
func BindMultipartCreate(c *fiber.Ctx) (CreateSubjectRequest, *multipart.FileHeader, error) {
	var req CreateSubjectRequest

	// text fields
	req.Code = strings.TrimSpace(c.FormValue("subjects_code"))
	req.Name = strings.TrimSpace(c.FormValue("subjects_name"))

	if v := strings.TrimSpace(c.FormValue("subjects_desc")); v != "" {
		req.Desc = &v
	}
	if v := strings.TrimSpace(c.FormValue("subjects_slug")); v != "" {
		s := helper.Slugify(v, 160)
		if s != "" {
			req.Slug = &s
		}
	}
	if v := strings.TrimSpace(c.FormValue("subjects_is_active")); v != "" {
		if b, err := parseBoolLoose(v); err == nil {
			req.IsActive = &b
		}
	}

	// image columns (opsional)
	if v := strings.TrimSpace(c.FormValue("subjects_image_url")); v != "" {
		req.ImageURL = &v
	}
	if v := strings.TrimSpace(c.FormValue("subjects_image_object_key")); v != "" {
		req.ImageObjectKey = &v
	}
	if v := strings.TrimSpace(c.FormValue("subjects_image_url_old")); v != "" {
		req.ImageURLOld = &v
	}
	if v := strings.TrimSpace(c.FormValue("subjects_image_object_key_old")); v != "" {
		req.ImageObjectKeyOld = &v
	}
	if v := strings.TrimSpace(c.FormValue("subjects_image_delete_pending_until")); v != "" {
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
	MasjidID *uuid.UUID `json:"subjects_masjid_id" form:"subjects_masjid_id"`

	Code     PatchField[string]  `json:"subjects_code"`
	Name     PatchField[string]  `json:"subjects_name"`
	Desc     PatchField[*string] `json:"subjects_desc"`
	Slug     PatchField[string]  `json:"subjects_slug"`
	IsActive PatchField[bool]    `json:"subjects_is_active"`

	ImageURL                PatchField[*string]    `json:"subjects_image_url"`
	ImageObjectKey          PatchField[*string]    `json:"subjects_image_object_key"`
	ImageURLOld             PatchField[*string]    `json:"subjects_image_url_old"`
	ImageObjectKeyOld       PatchField[*string]    `json:"subjects_image_object_key_old"`
	ImageDeletePendingUntil PatchField[*time.Time] `json:"subjects_image_delete_pending_until"`
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

func (p UpdateSubjectRequest) Apply(mo *m.SubjectsModel) {
	if p.MasjidID != nil {
		mo.SubjectsMasjidID = *p.MasjidID
	}

	// scalar
	if p.Code.Present && p.Code.Value != nil {
		mo.SubjectsCode = *p.Code.Value
	}
	if p.Name.Present && p.Name.Value != nil {
		mo.SubjectsName = *p.Name.Value
	}
	if p.Slug.Present && p.Slug.Value != nil {
		mo.SubjectsSlug = *p.Slug.Value
	}
	if p.IsActive.Present && p.IsActive.Value != nil {
		mo.SubjectsIsActive = *p.IsActive.Value
	}

	// nullable string
	if p.Desc.Present {
		if p.Desc.Value == nil {
			mo.SubjectsDesc = nil
		} else {
			mo.SubjectsDesc = *p.Desc.Value
		}
	}
	if p.ImageURL.Present {
		if p.ImageURL.Value == nil {
			mo.SubjectsImageURL = nil
		} else {
			mo.SubjectsImageURL = *p.ImageURL.Value
		}
	}
	if p.ImageObjectKey.Present {
		if p.ImageObjectKey.Value == nil {
			mo.SubjectsImageObjectKey = nil
		} else {
			mo.SubjectsImageObjectKey = *p.ImageObjectKey.Value
		}
	}
	if p.ImageURLOld.Present {
		if p.ImageURLOld.Value == nil {
			mo.SubjectsImageURLOld = nil
		} else {
			mo.SubjectsImageURLOld = *p.ImageURLOld.Value
		}
	}
	if p.ImageObjectKeyOld.Present {
		if p.ImageObjectKeyOld.Value == nil {
			mo.SubjectsImageObjectKeyOld = nil
		} else {
			mo.SubjectsImageObjectKeyOld = *p.ImageObjectKeyOld.Value
		}
	}

	// nullable time
	if p.ImageDeletePendingUntil.Present {
		if p.ImageDeletePendingUntil.Value == nil {
			mo.SubjectsImageDeletePendingUntil = nil
		} else {
			mo.SubjectsImageDeletePendingUntil = *p.ImageDeletePendingUntil.Value
		}
	}

	mo.SubjectsUpdatedAt = time.Now()
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
	if has("subjects_masjid_id") {
		if id, err := uuid.Parse(get("subjects_masjid_id")); err == nil {
			req.MasjidID = &id
		}
	}

	// scalar strings
	if has("subjects_code") {
		req.Code.Present = true
		v := get("subjects_code")
		req.Code.Value = &v
	}
	if has("subjects_name") {
		req.Name.Present = true
		v := get("subjects_name")
		req.Name.Value = &v
	}
	if has("subjects_slug") {
		req.Slug.Present = true
		v := helper.Slugify(get("subjects_slug"), 160)
		req.Slug.Value = &v
	}

	// desc (nullable via __null)
	if has("subjects_desc") || isNull("subjects_desc") {
		req.Desc.Present = true
		if isNull("subjects_desc") {
			req.Desc.Value = nil
		} else {
			req.Desc.Value = new(*string)
			setPtrStr(req.Desc.Value, get("subjects_desc"))
		}
	}

	// is_active
	if has("subjects_is_active") {
		req.IsActive.Present = true
		if b, err := parseBoolLoose(get("subjects_is_active")); err == nil {
			req.IsActive.Value = &b
		}
	}

	// image columns (nullable)
	if has("subjects_image_url") || isNull("subjects_image_url") {
		req.ImageURL.Present = true
		if isNull("subjects_image_url") {
			req.ImageURL.Value = nil
		} else {
			req.ImageURL.Value = new(*string)
			setPtrStr(req.ImageURL.Value, get("subjects_image_url"))
		}
	}
	if has("subjects_image_object_key") || isNull("subjects_image_object_key") {
		req.ImageObjectKey.Present = true
		if isNull("subjects_image_object_key") {
			req.ImageObjectKey.Value = nil
		} else {
			req.ImageObjectKey.Value = new(*string)
			setPtrStr(req.ImageObjectKey.Value, get("subjects_image_object_key"))
		}
	}
	if has("subjects_image_url_old") || isNull("subjects_image_url_old") {
		req.ImageURLOld.Present = true
		if isNull("subjects_image_url_old") {
			req.ImageURLOld.Value = nil
		} else {
			req.ImageURLOld.Value = new(*string)
			setPtrStr(req.ImageURLOld.Value, get("subjects_image_url_old"))
		}
	}
	if has("subjects_image_object_key_old") || isNull("subjects_image_object_key_old") {
		req.ImageObjectKeyOld.Present = true
		if isNull("subjects_image_object_key_old") {
			req.ImageObjectKeyOld.Value = nil
		} else {
			req.ImageObjectKeyOld.Value = new(*string)
			setPtrStr(req.ImageObjectKeyOld.Value, get("subjects_image_object_key_old"))
		}
	}
	if has("subjects_image_delete_pending_until") || isNull("subjects_image_delete_pending_until") {
		req.ImageDeletePendingUntil.Present = true
		if isNull("subjects_image_delete_pending_until") {
			req.ImageDeletePendingUntil.Value = nil
		} else {
			raw := get("subjects_image_delete_pending_until")
			if t, err := time.Parse(time.RFC3339, raw); err == nil {
				req.ImageDeletePendingUntil.Value = new(*time.Time)
				setPtrTime(req.ImageDeletePendingUntil.Value, t)
			} else {
				return req, nil, fiber.NewError(fiber.StatusBadRequest,
					"Waktu (RFC3339) tidak valid pada subjects_image_delete_pending_until")
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

/* ================= Queries & Responses (unchanged) ================= */

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
	SubjectsID                      uuid.UUID  `json:"subjects_id"`
	SubjectsMasjidID                uuid.UUID  `json:"subjects_masjid_id"`
	SubjectsCode                    string     `json:"subjects_code"`
	SubjectsName                    string     `json:"subjects_name"`
	SubjectsDesc                    *string    `json:"subjects_desc,omitempty"`
	SubjectsSlug                    string     `json:"subjects_slug"`
	SubjectsImageURL                *string    `json:"subjects_image_url,omitempty"`
	SubjectsImageObjectKey          *string    `json:"subjects_image_object_key,omitempty"`
	SubjectsImageURLOld             *string    `json:"subjects_image_url_old,omitempty"`
	SubjectsImageObjectKeyOld       *string    `json:"subjects_image_object_key_old,omitempty"`
	SubjectsImageDeletePendingUntil *time.Time `json:"subjects_image_delete_pending_until,omitempty"`
	SubjectsIsActive                bool       `json:"subjects_is_active"`
	SubjectsCreatedAt               time.Time  `json:"subjects_created_at"`
	SubjectsUpdatedAt               time.Time  `json:"subjects_updated_at"`
	SubjectsDeletedAt               *time.Time `json:"subjects_deleted_at,omitempty"`
}

func FromSubjectModel(mo m.SubjectsModel) SubjectResponse {
	var deletedAt *time.Time
	if mo.SubjectsDeletedAt.Valid {
		t := mo.SubjectsDeletedAt.Time
		deletedAt = &t
	}
	return SubjectResponse{
		SubjectsID:                      mo.SubjectsID,
		SubjectsMasjidID:                mo.SubjectsMasjidID,
		SubjectsCode:                    mo.SubjectsCode,
		SubjectsName:                    mo.SubjectsName,
		SubjectsDesc:                    mo.SubjectsDesc,
		SubjectsSlug:                    mo.SubjectsSlug,
		SubjectsImageURL:                mo.SubjectsImageURL,
		SubjectsImageObjectKey:          mo.SubjectsImageObjectKey,
		SubjectsImageURLOld:             mo.SubjectsImageURLOld,
		SubjectsImageObjectKeyOld:       mo.SubjectsImageObjectKeyOld,
		SubjectsImageDeletePendingUntil: mo.SubjectsImageDeletePendingUntil,
		SubjectsIsActive:                mo.SubjectsIsActive,
		SubjectsCreatedAt:               mo.SubjectsCreatedAt,
		SubjectsUpdatedAt:               mo.SubjectsUpdatedAt,
		SubjectsDeletedAt:               deletedAt,
	}
}

func FromSubjectModels(rows []m.SubjectsModel) []SubjectResponse {
	out := make([]SubjectResponse, 0, len(rows))
	for i := range rows {
		out = append(out, FromSubjectModel(rows[i]))
	}
	return out
}
