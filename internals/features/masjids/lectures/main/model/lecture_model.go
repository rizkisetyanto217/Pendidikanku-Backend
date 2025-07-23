package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type LectureModel struct {
	LectureID                    uuid.UUID      `gorm:"column:lecture_id;primaryKey;type:uuid;default:gen_random_uuid()" json:"lecture_id"`
	LectureTitle                 string         `gorm:"column:lecture_title;type:varchar(255);not null" json:"lecture_title"`
	LectureDescription           string         `gorm:"column:lecture_description;type:text" json:"lecture_description"`
	TotalLectureSessions         *int           `gorm:"column:total_lecture_sessions" json:"total_lecture_sessions,omitempty"`
	LectureImageURL              *string        `gorm:"column:lecture_image_url;type:text" json:"lecture_image_url,omitempty"`
	LectureTeachers              datatypes.JSON `gorm:"column:lecture_teachers;type:jsonb" json:"lecture_teachers,omitempty"`
	LectureMasjidID              uuid.UUID      `gorm:"column:lecture_masjid_id;type:uuid;not null" json:"lecture_masjid_id"`

	// Pendaftaran dan pembayaran
	LectureIsRegistrationRequired bool       `gorm:"column:lecture_is_registration_required;default:false" json:"lecture_is_registration_required"`
	LectureIsPaid                 bool       `gorm:"column:lecture_is_paid;default:false" json:"lecture_is_paid"`
	LecturePrice                  *int       `gorm:"column:lecture_price" json:"lecture_price,omitempty"`
	LecturePaymentDeadline        *time.Time `gorm:"column:lecture_payment_deadline" json:"lecture_payment_deadline,omitempty"`

	// Kapasitas & visibilitas
	LectureCapacity  *int  `gorm:"column:lecture_capacity" json:"lecture_capacity,omitempty"`
	LectureIsActive  bool  `gorm:"column:lecture_is_active;default:true" json:"lecture_is_active"`
	LectureIsCerticateGenerated bool `gorm:"column:lecture_is_certificate_generated;default:false" json:"lecture_is_certificate_generated"`

	// Timestamps
	LectureCreatedAt time.Time  `gorm:"column:lecture_created_at;autoCreateTime" json:"lecture_created_at"`
	LectureUpdatedAt *time.Time `gorm:"column:lecture_updated_at;autoUpdateTime" json:"lecture_updated_at,omitempty"`
	DeletedAt gorm.DeletedAt `gorm:"column:lecture_deleted_at;softDelete" json:"lecture_deleted_at,omitempty"`

}

func (LectureModel) TableName() string {
	return "lectures"
}
