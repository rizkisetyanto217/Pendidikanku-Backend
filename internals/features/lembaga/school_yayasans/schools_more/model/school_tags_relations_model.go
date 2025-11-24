package model

import (
	School "madinahsalam_backend/internals/features/lembaga/school_yayasans/schools/model"
	"time"

	"github.com/google/uuid"
)

type SchoolTagRelationModel struct {
	SchoolTagRelationID        uuid.UUID `gorm:"column:school_tag_relation_id;primaryKey;type:uuid;default:gen_random_uuid()" json:"school_tag_relation_id"`
	SchoolTagRelationSchoolID  uuid.UUID `gorm:"column:school_tag_relation_school_id;type:uuid;not null" json:"school_tag_relation_school_id"`
	SchoolTagRelationTagID     uuid.UUID `gorm:"column:school_tag_relation_tag_id;type:uuid;not null" json:"school_tag_relation_tag_id"`
	SchoolTagRelationCreatedAt time.Time `gorm:"column:school_tag_relation_created_at;autoCreateTime" json:"school_tag_relation_created_at"`

	// Optional: relasi ke school dan tag jika ingin digunakan untuk Preload
	School School.SchoolModel `gorm:"foreignKey:SchoolTagRelationSchoolID;references:SchoolID" json:"school,omitempty"`

	SchoolTag *SchoolTagModel `gorm:"foreignKey:SchoolTagRelationTagID;references:SchoolTagID" json:"tag,omitempty"`
}

// TableName override
func (SchoolTagRelationModel) TableName() string {
	return "school_tag_relations"
}
