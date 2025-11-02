package model

import (
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LectureSessionModel struct {
	LectureSessionID          uuid.UUID `gorm:"column:lecture_session_id;primaryKey;type:uuid;default:gen_random_uuid()" json:"lecture_session_id"`
	LectureSessionTitle       string    `gorm:"column:lecture_session_title;type:varchar(255);not null" json:"lecture_session_title"`
	LectureSessionSlug        string    `gorm:"column:lecture_session_slug;type:varchar(255);not null;unique" json:"lecture_session_slug"` // ✅ Slug URL
	LectureSessionDescription string    `gorm:"column:lecture_session_description;type:text" json:"lecture_session_description"`

	// 👤 Pengajar
	LectureSessionTeacherID   uuid.UUID `gorm:"column:lecture_session_teacher_id;type:uuid;not null" json:"lecture_session_teacher_id"`
	LectureSessionTeacherName string    `gorm:"column:lecture_session_teacher_name;type:varchar(255)" json:"lecture_session_teacher_name"`

	// ⏰ Jadwal
	LectureSessionStartTime time.Time `gorm:"column:lecture_session_start_time;not null" json:"lecture_session_start_time"`
	LectureSessionEndTime   time.Time `gorm:"column:lecture_session_end_time;not null" json:"lecture_session_end_time"`

	// 📍 Lokasi & Gambar
	LectureSessionPlace    *string `gorm:"column:lecture_session_place;type:text" json:"lecture_session_place"`
	LectureSessionImageURL *string `gorm:"column:lecture_session_image_url;type:text" json:"lecture_session_image_url"`

	// 🔗 Relasi ke lecture utama
	LectureSessionLectureID *uuid.UUID `gorm:"column:lecture_session_lecture_id;type:uuid" json:"lecture_session_lecture_id"`

	// ✅ Tambahan School ID
	LectureSessionSchoolID uuid.UUID `gorm:"column:lecture_session_school_id;type:uuid;not null" json:"lecture_session_school_id"`

	// ✅ Validasi Admin
	LectureSessionApprovedByAdminID *uuid.UUID `gorm:"column:lecture_session_approved_by_admin_id;type:uuid" json:"lecture_session_approved_by_admin_id"`
	LectureSessionApprovedByAdminAt *time.Time `gorm:"column:lecture_session_approved_by_admin_at" json:"lecture_session_approved_by_admin_at"`

	// ✅ Validasi Author
	LectureSessionApprovedByAuthorID *uuid.UUID `gorm:"column:lecture_session_approved_by_author_id;type:uuid" json:"lecture_session_approved_by_author_id"`
	LectureSessionApprovedByAuthorAt *time.Time `gorm:"column:lecture_session_approved_by_author_at" json:"lecture_session_approved_by_author_at"`

	// ✅ Validasi Teacher
	LectureSessionApprovedByTeacherID *uuid.UUID `gorm:"column:lecture_session_approved_by_teacher_id;type:uuid" json:"lecture_session_approved_by_teacher_id"`
	LectureSessionApprovedByTeacherAt *time.Time `gorm:"column:lecture_session_approved_by_teacher_at" json:"lecture_session_approved_by_teacher_at"`

	// ✅ Validasi Admin DKM
	LectureSessionApprovedByDkmAt *time.Time `gorm:"column:lecture_session_approved_by_dkm_at" json:"lecture_session_approved_by_dkm_at"`

	// 📌 Status publikasi
	LectureSessionIsActive bool `gorm:"column:lecture_session_is_active;default:false" json:"lecture_session_is_active"`

	// 🕒 Metadata
	LectureSessionCreatedAt time.Time      `gorm:"column:lecture_session_created_at;autoCreateTime" json:"lecture_session_created_at"`
	LectureSessionUpdatedAt *time.Time     `gorm:"column:lecture_session_updated_at;autoUpdateTime" json:"lecture_session_updated_at"`
	LectureSessionDeletedAt gorm.DeletedAt `gorm:"column:lecture_session_deleted_at" json:"lecture_session_deleted_at"`
}

func (LectureSessionModel) TableName() string {
	return "lecture_sessions"
}

// BeforeCreate generates slug automatically if not set
func (l *LectureSessionModel) BeforeCreate(tx *gorm.DB) (err error) {
	if l.LectureSessionSlug == "" && l.LectureSessionTitle != "" {
		baseSlug := generateSlug(l.LectureSessionTitle)
		slug := baseSlug
		counter := 1

		// Cek keberadaan slug di database
		for {
			var count int64
			tx.Model(&LectureSessionModel{}).Where("lecture_session_slug = ?", slug).Count(&count)
			if count == 0 {
				break
			}
			// Tambahkan suffix jika sudah ada
			slug = baseSlug + "-" + uuid.New().String()[:8]
			counter++
		}
		l.LectureSessionSlug = slug
	}
	return nil
}

// generateSlug creates a URL-friendly slug from the title
func generateSlug(title string) string {
	title = strings.ToLower(title)

	var b strings.Builder
	lastDash := false
	for _, r := range title {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastDash = false
		} else if !lastDash {
			b.WriteRune('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
