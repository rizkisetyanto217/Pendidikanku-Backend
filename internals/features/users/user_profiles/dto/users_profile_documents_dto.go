// file: internals/features/users/profile/dto/users_profile_document_dto.go
package dto

import (
	"time"

	"masjidku_backend/internals/features/users/user_profiles/model"

	"github.com/google/uuid"
)

/* =========================
   CREATE (JSON)
   ========================= */
// Dipakai jika file_url sudah tersedia (mis. hasil upload ke Supabase di sisi server lain)
type CreateUserProfileDocumentJSON struct {
	DocType string `json:"doc_type" validate:"required,max=50"`
	FileURL string `json:"file_url" validate:"required,url"`
}

/* =========================
   CREATE (MULTIPART)
   ========================= */
// Dipakai jika kamu menerima file langsung via multipart dan nanti menghasilkan file_url
// Catatan: field "File" tidak divalidasi url, karena akan diterima sebagai form file (c.FormFile).
type CreateUserProfileDocumentMultipart struct {
	DocType string `form:"doc_type" validate:"required,max=50"`
	// Ambil file dengan c.FormFile("file")
	// Simpan ke storage → dapatkan URL → set ke model.FileURL saat create
}

/* =========================
   UPDATE (JSON / partial)
   ========================= */
// Partial update; doc_type jarang diubah karena constraint unik (user_id, doc_type)
type UpdateUserProfileDocumentJSON struct {
	FileURL                *string    `json:"file_url" validate:"omitempty,url"`
	FileTrashURL           *string    `json:"file_trash_url" validate:"omitempty,url"`
	FileDeletePendingUntil *time.Time `json:"file_delete_pending_until" validate:"omitempty"`
}

/* =========================
   UPDATE (MULTIPART / partial)
   ========================= */
// Jika update via multipart (mis. ganti file → hasilkan file_url baru)
type UpdateUserProfileDocumentMultipart struct {
	// File baru: ambil via c.FormFile("file") lalu generate URL → set ke model
	// Opsional: bisa juga terima file_trash_url dan file_delete_pending_until
	FileTrashURL           *string `form:"file_trash_url" validate:"omitempty,url"`
	FileDeletePendingUntil *string `form:"file_delete_pending_until" validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
	// Jika ingin izinkan override manual:
	FileURL *string `form:"file_url" validate:"omitempty,url"`
}

/* =========================
   QUERY PARAMS (List & Filter)
   ========================= */
// Untuk endpoint list/filter (optional pagination)
type ListUserProfileDocumentQuery struct {
	DocType   *string `query:"doc_type" validate:"omitempty,max=50"`
	OnlyAlive *bool   `query:"only_alive"` // default true → WHERE deleted_at IS NULL
	Page      int     `query:"page" validate:"omitempty,min=1"`
	Limit     int     `query:"limit" validate:"omitempty,min=1,max=200"`
}

/* =========================
   RESPONSE DTO
   ========================= */

type UserProfileDocumentResponse struct {
	ID                     uuid.UUID  `json:"id"`
	UserID                 uuid.UUID  `json:"user_id"`
	DocType                string     `json:"doc_type"`
	FileURL                string     `json:"file_url"`
	FileTrashURL           *string    `json:"file_trash_url,omitempty"`
	FileDeletePendingUntil *time.Time `json:"file_delete_pending_until,omitempty"`
	UploadedAt             time.Time  `json:"uploaded_at"`
	UpdatedAt              *time.Time `json:"updated_at,omitempty"`
	DeletedAt              *time.Time `json:"deleted_at,omitempty"`
}

type UserProfileDocumentListResponse struct {
	Data       []UserProfileDocumentResponse `json:"data"`
	Pagination PaginationMeta                `json:"pagination"`
}

type PaginationMeta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}

/* =========================
   MAPPERS
   ========================= */

func ToResponse(m model.UsersProfileDocumentModel) UserProfileDocumentResponse {
	var deletedAtPtr *time.Time
	if m.DeletedAt.Valid {
		dt := m.DeletedAt.Time
		deletedAtPtr = &dt
	}
	return UserProfileDocumentResponse{
		ID:                     m.ID,
		UserID:                 m.UserID,
		DocType:                m.DocType,
		FileURL:                m.FileURL,
		FileTrashURL:           m.FileTrashURL,
		FileDeletePendingUntil: m.FileDeletePendingUntil,
		UploadedAt:             m.UploadedAt,
		UpdatedAt:              m.UpdatedAt,
		DeletedAt:              deletedAtPtr,
	}
}

func ToModelCreate(userID uuid.UUID, in CreateUserProfileDocumentJSON) model.UsersProfileDocumentModel {
	return model.UsersProfileDocumentModel{
		UserID:  userID,
		DocType: in.DocType,
		FileURL: in.FileURL,
	}
}

// Untuk multipart create: kamu akan set FileURL setelah upload file selesai.
func ToModelCreateMultipart(userID uuid.UUID, docType string, fileURL string) model.UsersProfileDocumentModel {
	return model.UsersProfileDocumentModel{
		UserID:  userID,
		DocType: docType,
		FileURL: fileURL,
	}
}

// Apply partial update ke model existing
func ApplyModelUpdate(m *model.UsersProfileDocumentModel, in UpdateUserProfileDocumentJSON) {
	if in.FileURL != nil {
		m.FileURL = *in.FileURL
	}
	if in.FileTrashURL != nil {
		m.FileTrashURL = in.FileTrashURL
	}
	if in.FileDeletePendingUntil != nil {
		m.FileDeletePendingUntil = in.FileDeletePendingUntil
	}
}

// Versi multipart: convert string time → time.Time di controller sebelum panggil ini
func ApplyModelUpdateMultipart(m *model.UsersProfileDocumentModel, fileURL *string, fileTrashURL *string, fileDeletePendingAt *time.Time) {
	if fileURL != nil {
		m.FileURL = *fileURL
	}
	if fileTrashURL != nil {
		m.FileTrashURL = fileTrashURL
	}
	if fileDeletePendingAt != nil {
		m.FileDeletePendingUntil = fileDeletePendingAt
	}
}
