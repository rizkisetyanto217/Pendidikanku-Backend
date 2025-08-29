// internals/features/lembaga/subjects/main/model/subject_model.go
package model

import (
	"time"

	"github.com/google/uuid"
)

// SubjectsModel merepresentasikan tabel "subjects" (lihat SQL).
// Catatan: Unik slug & code per masjid (case-insensitive, soft-delete aware)
// di-enforce di level DB lewat index partial; tidak didefinisikan via tag GORM.
type SubjectsModel struct {
	SubjectsID       uuid.UUID `gorm:"column:subjects_id;type:uuid;default:gen_random_uuid();primaryKey" json:"subjects_id"`
	SubjectsMasjidID uuid.UUID `gorm:"column:subjects_masjid_id;type:uuid;not null;index" json:"subjects_masjid_id"`

	SubjectsCode string  `gorm:"column:subjects_code;type:varchar(40);not null" json:"subjects_code"`
	SubjectsName string  `gorm:"column:subjects_name;type:varchar(120);not null" json:"subjects_name"`
	SubjectsDesc *string `gorm:"column:subjects_desc;type:text" json:"subjects_desc,omitempty"`

	// Baru: slug (nullable sesuai SQL). Uniqueness per-masjid di level DB.
	SubjectsSlug *string `gorm:"column:subjects_slug;type:varchar(160)" json:"subjects_slug,omitempty"`

	SubjectsIsActive bool `gorm:"column:subjects_is_active;not null;default:true" json:"subjects_is_active"`

	SubjectsCreatedAt time.Time  `gorm:"column:subjects_created_at;not null;default:CURRENT_TIMESTAMP" json:"subjects_created_at"`
	SubjectsUpdatedAt *time.Time `gorm:"column:subjects_updated_at" json:"subjects_updated_at,omitempty"`
	SubjectsDeletedAt *time.Time `gorm:"column:subjects_deleted_at" json:"subjects_deleted_at,omitempty"`
}

func (SubjectsModel) TableName() string { return "subjects" }
