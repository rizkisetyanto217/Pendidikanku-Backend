package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

/* ===== ENUM (ikuti DB: class_delivery_mode_enum) ===== */
type ClassDeliveryMode string

const (
	DeliveryModeOffline ClassDeliveryMode = "offline"
	DeliveryModeOnline  ClassDeliveryMode = "online"
	DeliveryModeHybrid  ClassDeliveryMode = "hybrid"
)

/* ===== ENUM: attendance_entry_mode_enum ===== */
type AttendanceEntryMode string

const (
	AttendanceEntryTeacherOnly AttendanceEntryMode = "teacher_only"
	AttendanceEntryStudentOnly AttendanceEntryMode = "student_only"
	AttendanceEntryBoth        AttendanceEntryMode = "both"
)

/* ===== ENUM: class_status_enum (IKUTI DB) ===== */
type ClassStatus string

const (
	ClassStatusActive    ClassStatus = "active"
	ClassStatusInactive  ClassStatus = "inactive"
	ClassStatusCompleted ClassStatus = "completed"
)

/*
=========================================================

	MODEL: class_section_subject_teachers (ikut SQL persis)

=========================================================
*/
type ClassSectionSubjectTeacherModel struct {
	/* ===== PK & Tenant ===== */
	ClassSectionSubjectTeacherID       uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:class_section_subject_teacher_id" json:"class_section_subject_teacher_id"`
	ClassSectionSubjectTeacherSchoolID uuid.UUID `gorm:"type:uuid;not null;column:class_section_subject_teacher_school_id" json:"class_section_subject_teacher_school_id"`

	/* ===== Identitas & Fasilitas ===== */
	ClassSectionSubjectTeacherSlug        *string `gorm:"type:varchar(160);column:class_section_subject_teacher_slug" json:"class_section_subject_teacher_slug,omitempty"`
	ClassSectionSubjectTeacherDescription *string `gorm:"type:text;column:class_section_subject_teacher_description" json:"class_section_subject_teacher_description,omitempty"`
	ClassSectionSubjectTeacherGroupURL    *string `gorm:"type:text;column:class_section_subject_teacher_group_url" json:"class_section_subject_teacher_group_url,omitempty"`

	/* ===== Agregat & Kapasitas (quota_total / quota_taken) ===== */
	ClassSectionSubjectTeacherTotalAttendance     int  `gorm:"not null;default:0;column:class_section_subject_teacher_total_attendance" json:"class_section_subject_teacher_total_attendance"`
	ClassSectionSubjectTeacherTotalMeetingsTarget *int `gorm:"column:class_section_subject_teacher_total_meetings_target" json:"class_section_subject_teacher_total_meetings_target,omitempty"`

	ClassSectionSubjectTeacherQuotaTotal *int `gorm:"column:class_section_subject_teacher_quota_total" json:"class_section_subject_teacher_quota_total,omitempty"`
	ClassSectionSubjectTeacherQuotaTaken int  `gorm:"not null;default:0;column:class_section_subject_teacher_quota_taken" json:"class_section_subject_teacher_quota_taken"`

	// total semua assessment (semua jenis)
	ClassSectionSubjectTeacherTotalAssessments int `gorm:"not null;default:0;column:class_section_subject_teacher_total_assessments" json:"class_section_subject_teacher_total_assessments"`

	// total per jenis assessment (training / daily_exam / exam)
	ClassSectionSubjectTeacherTotalAssessmentsTraining  int `gorm:"not null;default:0;column:class_section_subject_teacher_total_assessments_training" json:"class_section_subject_teacher_total_assessments_training"`
	ClassSectionSubjectTeacherTotalAssessmentsDailyExam int `gorm:"not null;default:0;column:class_section_subject_teacher_total_assessments_daily_exam" json:"class_section_subject_teacher_total_assessments_daily_exam"`
	ClassSectionSubjectTeacherTotalAssessmentsExam      int `gorm:"not null;default:0;column:class_section_subject_teacher_total_assessments_exam" json:"class_section_subject_teacher_total_assessments_exam"`
	ClassSectionSubjectTeacherTotalStudentsPassed       int `gorm:"not null;default:0;column:class_section_subject_teacher_total_students_passed" json:"class_section_subject_teacher_total_students_passed"`

	/* ===== Delivery mode (enum) ===== */
	ClassSectionSubjectTeacherDeliveryMode ClassDeliveryMode `gorm:"type:class_delivery_mode_enum;not null;default:'offline';column:class_section_subject_teacher_delivery_mode" json:"class_section_subject_teacher_delivery_mode"`

	/* ===== SNAPSHOT: attendance entry mode efektif di CSST ===== */
	ClassSectionSubjectTeacherSchoolAttendanceEntryModeCache *AttendanceEntryMode `gorm:"type:attendance_entry_mode_enum;column:class_section_subject_teacher_school_attendance_entry_mode_cache" json:"class_section_subject_teacher_school_attendance_entry_mode_cache,omitempty"`

	/* ===== SECTION caches (tanpa JSONB) ===== */
	ClassSectionSubjectTeacherClassSectionID        uuid.UUID `gorm:"type:uuid;not null;column:class_section_subject_teacher_class_section_id" json:"class_section_subject_teacher_class_section_id"`
	ClassSectionSubjectTeacherClassSectionSlugCache *string   `gorm:"type:varchar(160);column:class_section_subject_teacher_class_section_slug_cache" json:"class_section_subject_teacher_class_section_slug_cache,omitempty"`
	ClassSectionSubjectTeacherClassSectionNameCache *string   `gorm:"type:varchar(160);column:class_section_subject_teacher_class_section_name_cache" json:"class_section_subject_teacher_class_section_name_cache,omitempty"`
	ClassSectionSubjectTeacherClassSectionCodeCache *string   `gorm:"type:varchar(50);column:class_section_subject_teacher_class_section_code_cache" json:"class_section_subject_teacher_class_section_code_cache,omitempty"`
	ClassSectionSubjectTeacherClassSectionURLCache  *string   `gorm:"type:text;column:class_section_subject_teacher_class_section_url_cache" json:"class_section_subject_teacher_class_section_url_cache,omitempty"`

	/* ===== ROOM cache ===== */
	ClassSectionSubjectTeacherClassRoomID        *uuid.UUID      `gorm:"type:uuid;column:class_section_subject_teacher_class_room_id" json:"class_section_subject_teacher_class_room_id,omitempty"`
	ClassSectionSubjectTeacherClassRoomSlugCache *string         `gorm:"type:varchar(160);column:class_section_subject_teacher_class_room_slug_cache" json:"class_section_subject_teacher_class_room_slug_cache,omitempty"`
	ClassSectionSubjectTeacherClassRoomCache     *datatypes.JSON `gorm:"type:jsonb;column:class_section_subject_teacher_class_room_cache" json:"class_section_subject_teacher_class_room_cache,omitempty"`

	// generated (read-only)
	ClassSectionSubjectTeacherClassRoomNameCache     *string `gorm:"->;column:class_section_subject_teacher_class_room_name_cache" json:"class_section_subject_teacher_class_room_name_cache,omitempty"`
	ClassSectionSubjectTeacherClassRoomSlugCacheGen  *string `gorm:"->;column:class_section_subject_teacher_class_room_slug_cache_gen" json:"class_section_subject_teacher_class_room_slug_cache_gen,omitempty"`
	ClassSectionSubjectTeacherClassRoomLocationCache *string `gorm:"->;column:class_section_subject_teacher_class_room_location_cache" json:"class_section_subject_teacher_class_room_location_cache,omitempty"`

	/* ===== PEOPLE caches (teacher & assistant) ===== */
	ClassSectionSubjectTeacherSchoolTeacherID        *uuid.UUID      `gorm:"type:uuid;column:class_section_subject_teacher_school_teacher_id" json:"class_section_subject_teacher_school_teacher_id,omitempty"`
	ClassSectionSubjectTeacherSchoolTeacherSlugCache *string         `gorm:"type:varchar(160);column:class_section_subject_teacher_school_teacher_slug_cache" json:"class_section_subject_teacher_school_teacher_slug_cache,omitempty"`
	ClassSectionSubjectTeacherSchoolTeacherCache     *datatypes.JSON `gorm:"type:jsonb;column:class_section_subject_teacher_school_teacher_cache" json:"class_section_subject_teacher_school_teacher_cache,omitempty"`

	ClassSectionSubjectTeacherAssistantSchoolTeacherID        *uuid.UUID      `gorm:"type:uuid;column:class_section_subject_teacher_assistant_school_teacher_id" json:"class_section_subject_teacher_assistant_school_teacher_id,omitempty"`
	ClassSectionSubjectTeacherAssistantSchoolTeacherSlugCache *string         `gorm:"type:varchar(160);column:class_section_subject_teacher_assistant_school_teacher_slug_cache" json:"class_section_subject_teacher_assistant_school_teacher_slug_cache,omitempty"`
	ClassSectionSubjectTeacherAssistantSchoolTeacherCache     *datatypes.JSON `gorm:"type:jsonb;column:class_section_subject_teacher_assistant_school_teacher_cache" json:"class_section_subject_teacher_assistant_school_teacher_cache,omitempty"`

	// generated names (read-only)
	ClassSectionSubjectTeacherSchoolTeacherNameCache          *string `gorm:"->;column:class_section_subject_teacher_school_teacher_name_cache" json:"class_section_subject_teacher_school_teacher_name_cache,omitempty"`
	ClassSectionSubjectTeacherAssistantSchoolTeacherNameCache *string `gorm:"->;column:class_section_subject_teacher_assistant_school_teacher_name_cache" json:"class_section_subject_teacher_assistant_school_teacher_name_cache,omitempty"`

	/* ===== SUBJECT (via CLASS_SUBJECT) cache ===== */
	ClassSectionSubjectTeacherTotalBooks       int        `gorm:"not null;default:0;column:class_section_subject_teacher_total_books" json:"class_section_subject_teacher_total_books"`
	ClassSectionSubjectTeacherClassSubjectID   uuid.UUID  `gorm:"type:uuid;not null;column:class_section_subject_teacher_class_subject_id" json:"class_section_subject_teacher_class_subject_id"`
	ClassSectionSubjectTeacherSubjectID        *uuid.UUID `gorm:"type:uuid;column:class_section_subject_teacher_subject_id" json:"class_section_subject_teacher_subject_id,omitempty"`
	ClassSectionSubjectTeacherSubjectNameCache *string    `gorm:"type:varchar(160);column:class_section_subject_teacher_subject_name_cache" json:"class_section_subject_teacher_subject_name_cache,omitempty"`
	ClassSectionSubjectTeacherSubjectCodeCache *string    `gorm:"type:varchar(80);column:class_section_subject_teacher_subject_code_cache" json:"class_section_subject_teacher_subject_code_cache,omitempty"`
	ClassSectionSubjectTeacherSubjectSlugCache *string    `gorm:"type:varchar(160);column:class_section_subject_teacher_subject_slug_cache" json:"class_section_subject_teacher_subject_slug_cache,omitempty"`

	/* ===== ACADEMIC_TERM cache ===== */
	ClassSectionSubjectTeacherAcademicTermID            *uuid.UUID `gorm:"type:uuid;column:class_section_subject_teacher_academic_term_id" json:"class_section_subject_teacher_academic_term_id,omitempty"`
	ClassSectionSubjectTeacherAcademicTermNameCache     *string    `gorm:"type:varchar(160);column:class_section_subject_teacher_academic_term_name_cache" json:"class_section_subject_teacher_academic_term_name_cache,omitempty"`
	ClassSectionSubjectTeacherAcademicTermSlugCache     *string    `gorm:"type:varchar(160);column:class_section_subject_teacher_academic_term_slug_cache" json:"class_section_subject_teacher_academic_term_slug_cache,omitempty"`
	ClassSectionSubjectTeacherAcademicYearCache         *string    `gorm:"type:varchar(160);column:class_section_subject_teacher_academic_year_cache" json:"class_section_subject_teacher_academic_year_cache,omitempty"`
	ClassSectionSubjectTeacherAcademicTermAngkatanCache *int       `gorm:"column:class_section_subject_teacher_academic_term_angkatan_cache" json:"class_section_subject_teacher_academic_term_angkatan_cache,omitempty"`

	/* ===== KKM SNAPSHOT (cache + override per CSST) ===== */
	ClassSectionSubjectTeacherMinPassingScoreClassSubjectCache *int `gorm:"column:class_section_subject_teacher_min_passing_score_class_subject_cache" json:"class_section_subject_teacher_min_passing_score_class_subject_cache,omitempty"`
	ClassSectionSubjectTeacherMinPassingScore                  *int `gorm:"column:class_section_subject_teacher_min_passing_score" json:"class_section_subject_teacher_min_passing_score,omitempty"`

	/* ===== Status & audit ===== */
	ClassSectionSubjectTeacherStatus      ClassStatus    `gorm:"type:class_status_enum;not null;default:'active';column:class_section_subject_teacher_status" json:"class_section_subject_teacher_status"`
	ClassSectionSubjectTeacherCompletedAt *time.Time     `gorm:"type:timestamptz;column:class_section_subject_teacher_completed_at" json:"class_section_subject_teacher_completed_at,omitempty"`
	ClassSectionSubjectTeacherCreatedAt   time.Time      `gorm:"type:timestamptz;not null;default:now();column:class_section_subject_teacher_created_at" json:"class_section_subject_teacher_created_at"`
	ClassSectionSubjectTeacherUpdatedAt   time.Time      `gorm:"type:timestamptz;not null;default:now();column:class_section_subject_teacher_updated_at" json:"class_section_subject_teacher_updated_at"`
	ClassSectionSubjectTeacherDeletedAt   gorm.DeletedAt `gorm:"column:class_section_subject_teacher_deleted_at;index" json:"class_section_subject_teacher_deleted_at,omitempty"`
}

func (ClassSectionSubjectTeacherModel) TableName() string {
	return "class_section_subject_teachers"
}
