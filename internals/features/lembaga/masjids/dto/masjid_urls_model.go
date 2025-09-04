// file: internals/features/lembaga/masjids/dto/masjid_url_dto.go
package dto

import (
	"time"

	m "masjidku_backend/internals/features/lembaga/masjids/model"

	"github.com/google/uuid"
)

/* =========================
   (Opsional) String rule buat runtime validate.Var(...)
   ========================= */
const MasjidURLTypeOneOf = "oneof=logo stempel ttd_ketua banner profile_cover gallery qr other bg_behind_main main linktree_bg"

/* =========================
   REQUEST DTOs
   ========================= */

// Admin bisa kirim masjid_id; kalau endpoint non-admin, isi dari token di controller.
type CreateMasjidURLRequest struct {
	MasjidID  *uuid.UUID `json:"masjid_id" validate:"omitempty,uuid4"`
	Type      string     `json:"type"       validate:"required,oneof=logo stempel ttd_ketua banner profile_cover gallery qr other bg_behind_main main linktree_bg"`
	FileURL   string     `json:"file_url"   validate:"required,url"`
	IsPrimary *bool      `json:"is_primary" validate:"omitempty"`
	IsActive  *bool      `json:"is_active"  validate:"omitempty"`
}

// Semua opsional (partial update). Pakai pointer agar bisa deteksi field dikirim/tidak.
type UpdateMasjidURLRequest struct {
	Type      *string `json:"type"       validate:"omitempty,oneof=logo stempel ttd_ketua banner profile_cover gallery qr other bg_behind_main main linktree_bg"`
	FileURL   *string `json:"file_url"   validate:"omitempty,url"`
	IsPrimary *bool   `json:"is_primary" validate:"omitempty"`
	IsActive  *bool   `json:"is_active"  validate:"omitempty"`
}

// PATCH /masjid-urls/:id — identik dengan Update (dipisah agar jelas semantic PATCH)
type PatchMasjidURLRequest struct {
	Type      *string `json:"type"       validate:"omitempty,oneof=logo stempel ttd_ketua banner profile_cover gallery qr other bg_behind_main main linktree_bg"`
	FileURL   *string `json:"file_url"   validate:"omitempty,url"`
	IsPrimary *bool   `json:"is_primary" validate:"omitempty"`
	IsActive  *bool   `json:"is_active"  validate:"omitempty"`
}

// Query sederhana untuk list/filter
type ListMasjidURLQuery struct {
	MasjidID    uuid.UUID `json:"masjid_id"    validate:"required,uuid4"`
	Type        *string   `json:"type"         validate:"omitempty,oneof=logo stempel ttd_ketua banner profile_cover gallery qr other bg_behind_main main linktree_bg"`
	OnlyActive  *bool     `json:"only_active"  validate:"omitempty"`
	OnlyPrimary *bool     `json:"only_primary" validate:"omitempty"`
	Page        int       `json:"page"         validate:"omitempty,min=1"`
	PerPage     int       `json:"per_page"     validate:"omitempty,min=1,max=200"`
}

// Normalisasi nilai paging ketika kosong
func (q *ListMasjidURLQuery) Normalize() {
	if q.Page == 0 {
		q.Page = 1
	}
	if q.PerPage == 0 {
		q.PerPage = 20
	}
}

/* =========================
   BULK PATCH DTOs
   ========================= */

type BulkPatchMasjidURLItem struct {
	ID        uuid.UUID `json:"id"         validate:"required,uuid4"`
	Type      *string   `json:"type"       validate:"omitempty,oneof=logo stempel ttd_ketua banner profile_cover gallery qr other bg_behind_main main linktree_bg"`
	FileURL   *string   `json:"file_url"   validate:"omitempty,url"`
	IsPrimary *bool     `json:"is_primary" validate:"omitempty"`
	IsActive  *bool     `json:"is_active"  validate:"omitempty"`
}

type BulkPatchMasjidURLRequest struct {
	Items []BulkPatchMasjidURLItem `json:"items" validate:"required,min=1,dive"`
}

/* =========================
   RESPONSE DTOs
   ========================= */

type MasjidURLResponse struct {
	MasjidURLID        uuid.UUID  `json:"masjid_url_id"`
	MasjidURLMasjidID  uuid.UUID  `json:"masjid_url_masjid_id"`
	MasjidURLType      string     `json:"masjid_url_type"`
	MasjidURLFileURL   string     `json:"masjid_url_file_url"`
	MasjidURLIsPrimary bool       `json:"masjid_url_is_primary"`
	MasjidURLIsActive  bool       `json:"masjid_url_is_active"`
	MasjidURLCreatedAt time.Time  `json:"masjid_url_created_at"`
	MasjidURLUpdatedAt time.Time  `json:"masjid_url_updated_at"`
	MasjidURLDeletedAt *time.Time `json:"masjid_url_deleted_at,omitempty"`
}

// Optional paging envelope
type PageMeta struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Total   int `json:"total"`
}

type PagedMasjidURLResponse struct {
	Items []MasjidURLResponse `json:"items"`
	Meta  PageMeta            `json:"meta"`
}

/* =========================
   MAPPERS
   ========================= */

func ToMasjidURLResponse(in *m.MasjidURL) MasjidURLResponse {
	var deletedAt *time.Time
	if in.MasjidURLDeletedAt.Valid {
		deletedAt = &in.MasjidURLDeletedAt.Time
	}
	return MasjidURLResponse{
		MasjidURLID:        in.MasjidURLID,
		MasjidURLMasjidID:  in.MasjidURLMasjidID,
		MasjidURLType:      string(in.MasjidURLType),
		MasjidURLFileURL:   in.MasjidURLFileURL,
        MasjidURLIsPrimary: in.MasjidURLIsPrimary,
		MasjidURLIsActive:  in.MasjidURLIsActive,
		MasjidURLCreatedAt: in.MasjidURLCreatedAt,
		MasjidURLUpdatedAt: in.MasjidURLUpdatedAt,
		MasjidURLDeletedAt: deletedAt,
	}
}

func ToMasjidURLResponseSlice(rows []m.MasjidURL) []MasjidURLResponse {
	out := make([]MasjidURLResponse, 0, len(rows))
	for i := range rows {
		out = append(out, ToMasjidURLResponse(&rows[i]))
	}
	return out
}

/* =========================
   BUILDERS / APPLY
   ========================= */

// Controller bisa panggil ini setelah ambil masjid_id dari token bila req.MasjidID == nil
func (req *CreateMasjidURLRequest) ToModel(finalMasjidID uuid.UUID) *m.MasjidURL {
	isPrimary := false
	if req.IsPrimary != nil {
		isPrimary = *req.IsPrimary
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	return &m.MasjidURL{
		MasjidURLMasjidID:  finalMasjidID,
		MasjidURLType:      m.MasjidURLType(req.Type),
		MasjidURLFileURL:   req.FileURL,
		MasjidURLIsPrimary: isPrimary,
		MasjidURLIsActive:  isActive,
	}
}

// Terapkan partial update ke model yang sudah di-load dari DB.
func (req *UpdateMasjidURLRequest) ApplyToModel(row *m.MasjidURL) {
	if req.Type != nil {
		row.MasjidURLType = m.MasjidURLType(*req.Type)
	}
	if req.FileURL != nil {
		row.MasjidURLFileURL = *req.FileURL
	}
	if req.IsPrimary != nil {
		row.MasjidURLIsPrimary = *req.IsPrimary
	}
	if req.IsActive != nil {
		row.MasjidURLIsActive = *req.IsActive
	}
}

// PATCH (single)
func (p *PatchMasjidURLRequest) Apply(row *m.MasjidURL) {
	if p.Type != nil {
		row.MasjidURLType = m.MasjidURLType(*p.Type)
	}
	if p.FileURL != nil {
		row.MasjidURLFileURL = *p.FileURL
	}
	if p.IsPrimary != nil {
		row.MasjidURLIsPrimary = *p.IsPrimary
	}
	if p.IsActive != nil {
		row.MasjidURLIsActive = *p.IsActive
	}
}

// BULK PATCH
func (it *BulkPatchMasjidURLItem) Apply(row *m.MasjidURL) {
	if it.Type != nil {
		row.MasjidURLType = m.MasjidURLType(*it.Type)
	}
	if it.FileURL != nil {
		row.MasjidURLFileURL = *it.FileURL
	}
	if it.IsPrimary != nil {
		row.MasjidURLIsPrimary = *it.IsPrimary
	}
	if it.IsActive != nil {
		row.MasjidURLIsActive = *it.IsActive
	}
}
