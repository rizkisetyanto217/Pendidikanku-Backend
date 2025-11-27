package dto

import (
	model "madinahsalam_backend/internals/features/school/classes/class_sections/model"
	"time"

	"github.com/google/uuid"
)

// ===========================================
// RESPONSE DTO — COMPACT / LITE
// ===========================================

type StudentClassSectionCompactResp struct {
	StudentClassSectionID uuid.UUID `json:"student_class_section_id"`

	StudentClassSectionSchoolStudentID uuid.UUID `json:"student_class_section_school_student_id"`
	StudentClassSectionSectionID       uuid.UUID `json:"student_class_section_section_id"`
	StudentClassSectionSchoolID        uuid.UUID `json:"student_class_section_school_id"`

	// Snapshot minimal untuk list
	StudentClassSectionSectionSlugSnapshot string  `json:"student_class_section_section_slug_snapshot"`
	StudentClassSectionStudentCodeSnapshot *string `json:"student_class_section_student_code_snapshot,omitempty"`

	StudentClassSectionStatus string  `json:"student_class_section_status"`
	StudentClassSectionResult *string `json:"student_class_section_result,omitempty"`

	// Snapshot users_profile (ringkas)
	StudentClassSectionUserProfileNameSnapshot        *string `json:"student_class_section_user_profile_name_snapshot,omitempty"`
	StudentClassSectionUserProfileAvatarURLSnapshot   *string `json:"student_class_section_user_profile_avatar_url_snapshot,omitempty"`
	StudentClassSectionUserProfileWhatsappURLSnapshot *string `json:"student_class_section_user_profile_whatsapp_url_snapshot,omitempty"`
	StudentClassSectionUserProfileGenderSnapshot      *string `json:"student_class_section_user_profile_gender_snapshot,omitempty"`

	StudentClassSectionAssignedAt   time.Time  `json:"student_class_section_assigned_at"`
	StudentClassSectionUnassignedAt *time.Time `json:"student_class_section_unassigned_at,omitempty"`
	StudentClassSectionCompletedAt  *time.Time `json:"student_class_section_completed_at,omitempty"`

	StudentClassSectionCreatedAt time.Time  `json:"student_class_section_created_at"`
	StudentClassSectionUpdatedAt time.Time  `json:"student_class_section_updated_at"`
	StudentClassSectionDeletedAt *time.Time `json:"student_class_section_deleted_at,omitempty"`
}

func FromModelCompact(m *model.StudentClassSection) StudentClassSectionCompactResp {
	var res *string
	if m.StudentClassSectionResult != nil {
		v := string(*m.StudentClassSectionResult)
		res = &v
	}

	var delAt *time.Time
	if m.StudentClassSectionDeletedAt.Valid {
		t := m.StudentClassSectionDeletedAt.Time
		delAt = &t
	}

	return StudentClassSectionCompactResp{
		StudentClassSectionID: m.StudentClassSectionID,

		StudentClassSectionSchoolStudentID: m.StudentClassSectionSchoolStudentID,
		StudentClassSectionSectionID:       m.StudentClassSectionSectionID,
		StudentClassSectionSchoolID:        m.StudentClassSectionSchoolID,

		StudentClassSectionSectionSlugSnapshot: m.StudentClassSectionSectionSlugSnapshot,
		StudentClassSectionStudentCodeSnapshot: m.StudentClassSectionStudentCodeSnapshot,

		StudentClassSectionStatus: string(m.StudentClassSectionStatus),
		StudentClassSectionResult: res,

		StudentClassSectionUserProfileNameSnapshot:        m.StudentClassSectionUserProfileNameSnapshot,
		StudentClassSectionUserProfileAvatarURLSnapshot:   m.StudentClassSectionUserProfileAvatarURLSnapshot,
		StudentClassSectionUserProfileWhatsappURLSnapshot: m.StudentClassSectionUserProfileWhatsappURLSnapshot,
		StudentClassSectionUserProfileGenderSnapshot:      m.StudentClassSectionUserProfileGenderSnapshot,

		StudentClassSectionAssignedAt:   m.StudentClassSectionAssignedAt,
		StudentClassSectionUnassignedAt: m.StudentClassSectionUnassignedAt,
		StudentClassSectionCompletedAt:  m.StudentClassSectionCompletedAt,

		StudentClassSectionCreatedAt: m.StudentClassSectionCreatedAt,
		StudentClassSectionUpdatedAt: m.StudentClassSectionUpdatedAt,
		StudentClassSectionDeletedAt: delAt,
	}
}

func FromModelsStudentClassSectionCompact(list []model.StudentClassSection) []StudentClassSectionCompactResp {
	out := make([]StudentClassSectionCompactResp, 0, len(list))
	for i := range list {
		out = append(out, FromModelCompact(&list[i]))
	}
	return out
}
