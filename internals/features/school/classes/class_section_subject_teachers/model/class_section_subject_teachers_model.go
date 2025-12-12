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

	MODEL: class_section_subject_teachers (IKUT SQL terbaru)

=========================================================
*/
type ClassSectionSubjectTeacherModel struct {
	/* ===== PK & Tenant ===== */
	CSSTID       uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:csst_id" json:"csst_id"`
	CSSTSchoolID uuid.UUID `gorm:"type:uuid;not null;column:csst_school_id" json:"csst_school_id"`

	/* ===== Identitas & Fasilitas ===== */
	CSSTSlug        *string `gorm:"type:varchar(160);column:csst_slug" json:"csst_slug,omitempty"`
	CSSTDescription *string `gorm:"type:text;column:csst_description" json:"csst_description,omitempty"`
	CSSTGroupURL    *string `gorm:"type:text;column:csst_group_url" json:"csst_group_url,omitempty"`

	/* ===== Agregat & Quota ===== */
	CSSTTotalAttendance       int  `gorm:"not null;default:0;column:csst_total_attendance" json:"csst_total_attendance"`
	CSSTTotalMeetingsTarget   *int `gorm:"column:csst_total_meetings_target" json:"csst_total_meetings_target,omitempty"`
	CSSTQuotaTotal            *int `gorm:"column:csst_quota_total" json:"csst_quota_total,omitempty"`
	CSSTQuotaTaken            int  `gorm:"not null;default:0;column:csst_quota_taken" json:"csst_quota_taken"`
	CSSTTotalAssessments      int  `gorm:"not null;default:0;column:csst_total_assessments" json:"csst_total_assessments"`
	CSSTTotalAssessmentsTrain int  `gorm:"not null;default:0;column:csst_total_assessments_training" json:"csst_total_assessments_training"`
	CSSTTotalAssessmentsDaily int  `gorm:"not null;default:0;column:csst_total_assessments_daily_exam" json:"csst_total_assessments_daily_exam"`
	CSSTTotalAssessmentsExam  int  `gorm:"not null;default:0;column:csst_total_assessments_exam" json:"csst_total_assessments_exam"`
	CSSTTotalStudentsPassed   int  `gorm:"not null;default:0;column:csst_total_students_passed" json:"csst_total_students_passed"`

	/* ===== Delivery mode ===== */
	CSSTDeliveryMode ClassDeliveryMode `gorm:"type:class_delivery_mode_enum;not null;default:'offline';column:csst_delivery_mode" json:"csst_delivery_mode"`

	/* ===== Attendance entry mode cache ===== */
	CSSTSchoolAttendanceEntryModeCache *AttendanceEntryMode `gorm:"type:attendance_entry_mode_enum;column:csst_school_attendance_entry_mode_cache" json:"csst_school_attendance_entry_mode_cache,omitempty"`

	/* =======================
	   SECTION cache
	   ======================= */
	CSSTClassSectionID        uuid.UUID `gorm:"type:uuid;not null;column:csst_class_section_id" json:"csst_class_section_id"`
	CSSTClassSectionSlugCache *string   `gorm:"type:varchar(160);column:csst_class_section_slug_cache" json:"csst_class_section_slug_cache,omitempty"`
	CSSTClassSectionNameCache *string   `gorm:"type:varchar(160);column:csst_class_section_name_cache" json:"csst_class_section_name_cache,omitempty"`
	CSSTClassSectionCodeCache *string   `gorm:"type:varchar(50);column:csst_class_section_code_cache" json:"csst_class_section_code_cache,omitempty"`
	CSSTClassSectionURLCache  *string   `gorm:"type:text;column:csst_class_section_url_cache" json:"csst_class_section_url_cache,omitempty"`

	/* =======================
	   ROOM cache
	   ======================= */
	CSSTClassRoomID        *uuid.UUID      `gorm:"type:uuid;column:csst_class_room_id" json:"csst_class_room_id,omitempty"`
	CSSTClassRoomSlugCache *string         `gorm:"type:varchar(160);column:csst_class_room_slug_cache" json:"csst_class_room_slug_cache,omitempty"`
	CSSTClassRoomCache     *datatypes.JSON `gorm:"type:jsonb;column:csst_class_room_cache" json:"csst_class_room_cache,omitempty"`

	// generated (read-only)
	CSSTClassRoomNameCache     *string `gorm:"->;column:csst_class_room_name_cache" json:"csst_class_room_name_cache,omitempty"`
	CSSTClassRoomSlugCacheGen  *string `gorm:"->;column:csst_class_room_slug_cache_gen" json:"csst_class_room_slug_cache_gen,omitempty"`
	CSSTClassRoomLocationCache *string `gorm:"->;column:csst_class_room_location_cache" json:"csst_class_room_location_cache,omitempty"`

	/* =======================
	   PEOPLE cache (teacher & assistant)
	   ======================= */
	CSSTSchoolTeacherID        *uuid.UUID      `gorm:"type:uuid;column:csst_school_teacher_id" json:"csst_school_teacher_id,omitempty"`
	CSSTSchoolTeacherSlugCache *string         `gorm:"type:varchar(160);column:csst_school_teacher_slug_cache" json:"csst_school_teacher_slug_cache,omitempty"`
	CSSTSchoolTeacherCache     *datatypes.JSON `gorm:"type:jsonb;column:csst_school_teacher_cache" json:"csst_school_teacher_cache,omitempty"`

	CSSTAssistantSchoolTeacherID        *uuid.UUID      `gorm:"type:uuid;column:csst_assistant_school_teacher_id" json:"csst_assistant_school_teacher_id,omitempty"`
	CSSTAssistantSchoolTeacherSlugCache *string         `gorm:"type:varchar(160);column:csst_assistant_school_teacher_slug_cache" json:"csst_assistant_school_teacher_slug_cache,omitempty"`
	CSSTAssistantSchoolTeacherCache     *datatypes.JSON `gorm:"type:jsonb;column:csst_assistant_school_teacher_cache" json:"csst_assistant_school_teacher_cache,omitempty"`

	// generated names (read-only)
	CSSTSchoolTeacherNameCache          *string `gorm:"->;column:csst_school_teacher_name_cache" json:"csst_school_teacher_name_cache,omitempty"`
	CSSTAssistantSchoolTeacherNameCache *string `gorm:"->;column:csst_assistant_school_teacher_name_cache" json:"csst_assistant_school_teacher_name_cache,omitempty"`

	/* =======================
	   SUBJECT cache (via CLASS_SUBJECT)
	   ======================= */
	CSSTTotalBooks       int        `gorm:"not null;default:0;column:csst_total_books" json:"csst_total_books"`
	CSSTClassSubjectID   uuid.UUID  `gorm:"type:uuid;not null;column:csst_class_subject_id" json:"csst_class_subject_id"`
	CSSTSubjectID        *uuid.UUID `gorm:"type:uuid;column:csst_subject_id" json:"csst_subject_id,omitempty"`
	CSSTSubjectNameCache *string    `gorm:"type:varchar(160);column:csst_subject_name_cache" json:"csst_subject_name_cache,omitempty"`
	CSSTSubjectCodeCache *string    `gorm:"type:varchar(80);column:csst_subject_code_cache" json:"csst_subject_code_cache,omitempty"`
	CSSTSubjectSlugCache *string    `gorm:"type:varchar(160);column:csst_subject_slug_cache" json:"csst_subject_slug_cache,omitempty"`

	/* =======================
	   ACADEMIC_TERM cache
	   ======================= */
	CSSTAcademicTermID            *uuid.UUID `gorm:"type:uuid;column:csst_academic_term_id" json:"csst_academic_term_id,omitempty"`
	CSSTAcademicTermNameCache     *string    `gorm:"type:varchar(160);column:csst_academic_term_name_cache" json:"csst_academic_term_name_cache,omitempty"`
	CSSTAcademicTermSlugCache     *string    `gorm:"type:varchar(160);column:csst_academic_term_slug_cache" json:"csst_academic_term_slug_cache,omitempty"`
	CSSTAcademicYearCache         *string    `gorm:"type:varchar(160);column:csst_academic_year_cache" json:"csst_academic_year_cache,omitempty"`
	CSSTAcademicTermAngkatanCache *int       `gorm:"column:csst_academic_term_angkatan_cache" json:"csst_academic_term_angkatan_cache,omitempty"`

	/* =======================
	   KKM cache per CSST
	   ======================= */
	CSSTMinPassingScoreClassSubjectCache *int `gorm:"column:csst_min_passing_score_class_subject_cache" json:"csst_min_passing_score_class_subject_cache,omitempty"`

	/* ===== Status & audit ===== */
	CSSTStatus      ClassStatus    `gorm:"type:class_status_enum;not null;default:'active';column:csst_status" json:"csst_status"`
	CSSTCompletedAt *time.Time     `gorm:"type:timestamptz;column:csst_completed_at" json:"csst_completed_at,omitempty"`
	CSSTCreatedAt   time.Time      `gorm:"type:timestamptz;not null;default:now();column:csst_created_at" json:"csst_created_at"`
	CSSTUpdatedAt   time.Time      `gorm:"type:timestamptz;not null;default:now();column:csst_updated_at" json:"csst_updated_at"`
	CSSTDeletedAt   gorm.DeletedAt `gorm:"type:timestamptz;column:csst_deleted_at;index" json:"csst_deleted_at,omitempty"`
}

func (ClassSectionSubjectTeacherModel) TableName() string {
	return "class_section_subject_teachers"
}
