// file: internals/features/users/profile/model/users_profile_document_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UsersProfileDocumentModel merepresentasikan tabel users_profile_documents
type UsersProfileDocumentModel struct {
	ID                      uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID                  uuid.UUID      `gorm:"type:uuid;not null;uniqueIndex:uq_user_doc_type" json:"user_id"`
	DocType                 string         `gorm:"type:varchar(50);not null;uniqueIndex:uq_user_doc_type" json:"doc_type"`
	FileURL                 string         `gorm:"type:text;not null" json:"file_url"`
	FileTrashURL            *string        `gorm:"type:text" json:"file_trash_url"`
	FileDeletePendingUntil  *time.Time     `gorm:"type:timestamptz" json:"file_delete_pending_until"`
	UploadedAt              time.Time      `gorm:"type:timestamptz;not null;default:now()" json:"uploaded_at"`
	UpdatedAt               *time.Time     `gorm:"type:timestamptz" json:"updated_at"` // diperlukan oleh trigger set_updated_at()
	DeletedAt               gorm.DeletedAt `gorm:"type:timestamptz;index" json:"deleted_at,omitempty"`
}

func (UsersProfileDocumentModel) TableName() string {
	return "users_profile_documents"
}