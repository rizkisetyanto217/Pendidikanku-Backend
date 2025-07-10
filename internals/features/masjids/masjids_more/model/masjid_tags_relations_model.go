package model

import (
	Masjid "masjidku_backend/internals/features/masjids/masjids/model"
	"time"

	"github.com/google/uuid"
)

type MasjidTagRelationModel struct {
	MasjidTagRelationID        uuid.UUID `gorm:"column:masjid_tag_relation_id;primaryKey;type:uuid;default:gen_random_uuid()" json:"masjid_tag_relation_id"`
	MasjidTagRelationMasjidID  uuid.UUID `gorm:"column:masjid_tag_relation_masjid_id;type:uuid;not null" json:"masjid_tag_relation_masjid_id"`
	MasjidTagRelationTagID     uuid.UUID `gorm:"column:masjid_tag_relation_tag_id;type:uuid;not null" json:"masjid_tag_relation_tag_id"`
	MasjidTagRelationCreatedAt time.Time `gorm:"column:masjid_tag_relation_created_at;autoCreateTime" json:"masjid_tag_relation_created_at"`

	// Optional: relasi ke masjid dan tag jika ingin digunakan untuk Preload
	Masjid Masjid.MasjidModel `gorm:"foreignKey:MasjidTagRelationMasjidID;references:MasjidID" json:"masjid,omitempty"`

	MasjidTag *MasjidTagModel `gorm:"foreignKey:MasjidTagRelationTagID;references:MasjidTagID" json:"tag,omitempty"`
}

// TableName override
func (MasjidTagRelationModel) TableName() string {
	return "masjid_tag_relations"
}
