package dto

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"masjidku_backend/internals/features/school/class_subject_books/books/model"
)

/* =========================
 * Konstanta tipe URL
 * ========================= */
const (
	BookURLTypeCover    = "cover"
	BookURLTypeDesc     = "desc"
	BookURLTypeDownload = "download"
	BookURLTypePurchase = "purchase"
)

/* =========================================================
 * REQUESTS
 * ========================================================= */

// Create (JSON)
type CreateBookURLRequest struct {
	BookURLBookID uuid.UUID `json:"book_url_book_id" validate:"required,uuid4"`
	BookURLLabel  *string   `json:"book_url_label"    validate:"omitempty,max=120"`
	BookURLType   string    `json:"book_url_type"     validate:"required,oneof=cover desc download purchase"`
	BookURLHref   string    `json:"book_url_href"     validate:"required,url"`
	// housekeeping (opsional — biasanya server-side)
	BookURLTrashURL           *string    `json:"book_url_trash_url,omitempty"            validate:"omitempty,url"`
	BookURLDeletePendingUntil *time.Time `json:"book_url_delete_pending_until,omitempty" validate:"omitempty"`
}

// Update (partial JSON)
type UpdateBookURLRequest struct {
	BookURLLabel  *string `json:"book_url_label,omitempty"  validate:"omitempty,max=120"`
	BookURLType   *string `json:"book_url_type,omitempty"   validate:"omitempty,oneof=cover desc download purchase"`
	BookURLHref   *string `json:"book_url_href,omitempty"   validate:"omitempty,url"`
	// housekeeping (opsional)
	BookURLTrashURL           *string    `json:"book_url_trash_url,omitempty"            validate:"omitempty,url"`
	BookURLDeletePendingUntil *time.Time `json:"book_url_delete_pending_until,omitempty" validate:"omitempty"`
}

// Filter (list endpoint)
type FilterBookURLRequest struct {
	BookID      *uuid.UUID `query:"book_id"   validate:"omitempty,uuid4"`
	Type        *string    `query:"type"      validate:"omitempty,oneof=cover desc download purchase"`
	Search      *string    `query:"search"    validate:"omitempty,max=200"` // match ke label/href
	OnlyAlive   *bool      `query:"only_alive"`
	Page        *int       `query:"page"      validate:"omitempty,min=1"`
	Limit       *int       `query:"limit"     validate:"omitempty,min=1,max=200"`
	Sort        *string    `query:"sort"      validate:"omitempty,oneof=created_at_desc created_at_asc label_asc label_desc"`
}

/* =========================================================
 * RESPONSES
 * ========================================================= */

type BookURLResponse struct {
	BookURLID       uuid.UUID  `json:"book_url_id"`
	BookURLMasjidID uuid.UUID  `json:"book_url_masjid_id"`
	BookURLBookID   uuid.UUID  `json:"book_url_book_id"`

	BookURLLabel *string `json:"book_url_label,omitempty"`
	BookURLType  string  `json:"book_url_type"`
	BookURLHref  string  `json:"book_url_href"`

	BookURLTrashURL           *string    `json:"book_url_trash_url,omitempty"`
	BookURLDeletePendingUntil *time.Time `json:"book_url_delete_pending_until,omitempty"`

	BookURLCreatedAt time.Time `json:"book_url_created_at"`
	BookURLUpdatedAt time.Time `json:"book_url_updated_at"`
}

/* =========================================================
 * KONVERSI MODEL <-> DTO
 * ========================================================= */

func NormalizeBookURLType(t string) string {
	t = strings.TrimSpace(strings.ToLower(t))
	switch t {
	case BookURLTypeCover, BookURLTypeDesc, BookURLTypeDownload, BookURLTypePurchase:
		return t
	default:
		return BookURLTypeDesc // default paling aman
	}
}

func NewBookURLResponse(m model.BookURLModel) BookURLResponse {
	return BookURLResponse{
		BookURLID:                m.BookURLID,
		BookURLMasjidID:          m.BookURLMasjidID,
		BookURLBookID:            m.BookURLBookID,
		BookURLLabel:             m.BookURLLabel,
		BookURLType:              m.BookURLType,
		BookURLHref:              m.BookURLHref,
		BookURLTrashURL:          m.BookURLTrashURL,
		BookURLDeletePendingUntil: m.BookURLDeletePendingUntil,
		BookURLCreatedAt:         m.BookURLCreatedAt,
		BookURLUpdatedAt:         m.BookURLUpdatedAt,
	}
}

// ToModel: dipakai saat CREATE
func (r *CreateBookURLRequest) ToModel(masjidID uuid.UUID) model.BookURLModel {
	lbl := trimPtr(r.BookURLLabel)
	trash := trimPtr(r.BookURLTrashURL)

	return model.BookURLModel{
		BookURLMasjidID:          masjidID,
		BookURLBookID:            r.BookURLBookID,
		BookURLLabel:             lbl,
		BookURLType:              NormalizeBookURLType(r.BookURLType),
		BookURLHref:              strings.TrimSpace(r.BookURLHref),
		BookURLTrashURL:          trash,
		BookURLDeletePendingUntil: r.BookURLDeletePendingUntil,
	}
}

// ApplyToModel: dipakai saat UPDATE (partial)
func (r *UpdateBookURLRequest) ApplyToModel(m *model.BookURLModel) {
	if r.BookURLLabel != nil {
		*m.BookURLLabel = strings.TrimSpace(*r.BookURLLabel)
		if strings.TrimSpace(*r.BookURLLabel) == "" {
			m.BookURLLabel = nil
		}
	}
	if r.BookURLType != nil {
		m.BookURLType = NormalizeBookURLType(*r.BookURLType)
	}
	if r.BookURLHref != nil {
		m.BookURLHref = strings.TrimSpace(*r.BookURLHref)
	}
	if r.BookURLTrashURL != nil {
		tr := strings.TrimSpace(*r.BookURLTrashURL)
		if tr == "" {
			m.BookURLTrashURL = nil
		} else {
			m.BookURLTrashURL = &tr
		}
	}
	if r.BookURLDeletePendingUntil != nil {
		m.BookURLDeletePendingUntil = r.BookURLDeletePendingUntil
	}
}

