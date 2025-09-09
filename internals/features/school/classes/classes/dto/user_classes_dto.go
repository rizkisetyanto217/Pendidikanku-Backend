// file: internals/features/lembaga/classes/user_classes/main/dto/user_class_dto.go
package dto

import (
	"time"

	ucmodel "masjidku_backend/internals/features/school/classes/classes/model"

	"github.com/google/uuid"
)

/* ===================== REQUESTS ===================== */

type CreateUserClassRequest struct {
	// FK -> classes(class_id)
	UserClassesClassID uuid.UUID `json:"user_classes_class_id" validate:"required"`

	// Tenant. Di handler bisa diisi dari token; tetap optional di payload.
	UserClassesMasjidID *uuid.UUID `json:"user_classes_masjid_id" validate:"omitempty"`

	// Wajib: relasi ke masjid_students (DDL: NOT NULL)
	UserClassesMasjidStudentID uuid.UUID `json:"user_classes_masjid_student_id" validate:"required"`

	// Status dibatasi CHECK ('active','inactive','completed'); default 'active'
	UserClassesStatus *string `json:"user_classes_status" validate:"omitempty,oneof=active inactive completed"`

	// Outcome (hasil akhir) — hanya valid saat status=completed
	UserClassesResult *string `json:"user_classes_result" validate:"omitempty,oneof=passed failed"`

	// Jejak waktu enrolment
	UserClassesJoinedAt    *time.Time `json:"user_classes_joined_at" validate:"omitempty"`
	UserClassesLeftAt      *time.Time `json:"user_classes_left_at" validate:"omitempty"`
	UserClassesCompletedAt *time.Time `json:"user_classes_completed_at" validate:"omitempty"`
}

func (r *CreateUserClassRequest) ToModel(masjidIDFromCtx *uuid.UUID) *ucmodel.UserClassesModel {
	// Tentukan masjid_id final (payload > context)
	var masjidID uuid.UUID
	if r.UserClassesMasjidID != nil {
		masjidID = *r.UserClassesMasjidID
	} else if masjidIDFromCtx != nil {
		masjidID = *masjidIDFromCtx
	}

	status := ucmodel.UserClassStatusActive
	if r.UserClassesStatus != nil && *r.UserClassesStatus != "" {
		status = *r.UserClassesStatus
	}

	m := &ucmodel.UserClassesModel{
		UserClassesMasjidStudentID: r.UserClassesMasjidStudentID,
		UserClassesClassID:         r.UserClassesClassID,
		UserClassesMasjidID:        masjidID,
		UserClassesStatus:          status,
		UserClassesResult:          r.UserClassesResult,     // boleh nil
		UserClassesJoinedAt:        r.UserClassesJoinedAt,   // boleh nil
		UserClassesLeftAt:          r.UserClassesLeftAt,     // boleh nil
		UserClassesCompletedAt:     r.UserClassesCompletedAt, // boleh nil
	}

	// Guard ringan: jika status=completed tapi completed_at kosong → isi now
	if m.UserClassesStatus == ucmodel.UserClassStatusCompleted && m.UserClassesCompletedAt == nil {
		now := time.Now()
		m.UserClassesCompletedAt = &now
	}

	// Guard ringan: jika status!=completed → kosongkan result & completed_at (biarkan layer controller/DB enforce)
	if m.UserClassesStatus != ucmodel.UserClassStatusCompleted {
		m.UserClassesResult = nil
		m.UserClassesCompletedAt = nil
	}

	return m
}

type UpdateUserClassRequest struct {
	UserClassesClassID         *uuid.UUID `json:"user_classes_class_id" validate:"omitempty"`
	// Boleh diubah jika skenario pindah tenant dibuka (hati-hati FK komposit)
	UserClassesMasjidID        *uuid.UUID `json:"user_classes_masjid_id" validate:"omitempty"`
	// Boleh update relasi masjid_students (DDL: NOT NULL di model, jadi pointer hanya untuk signal update)
	UserClassesMasjidStudentID *uuid.UUID `json:"user_classes_masjid_student_id" validate:"omitempty"`

	// Status: active|inactive|completed
	UserClassesStatus *string `json:"user_classes_status" validate:"omitempty,oneof=active inactive completed"`

	// Outcome: passed|failed (hanya berlaku saat completed)
	UserClassesResult *string `json:"user_classes_result" validate:"omitempty,oneof=passed failed"`

	// Jejak waktu
	UserClassesJoinedAt    *time.Time `json:"user_classes_joined_at" validate:"omitempty"`
	UserClassesLeftAt      *time.Time `json:"user_classes_left_at" validate:"omitempty"`
	UserClassesCompletedAt *time.Time `json:"user_classes_completed_at" validate:"omitempty"`
}

func (r *UpdateUserClassRequest) ApplyToModel(m *ucmodel.UserClassesModel) {
	if r.UserClassesClassID != nil {
		m.UserClassesClassID = *r.UserClassesClassID
	}
	if r.UserClassesMasjidID != nil {
		m.UserClassesMasjidID = *r.UserClassesMasjidID
	}
	if r.UserClassesMasjidStudentID != nil {
		m.UserClassesMasjidStudentID = *r.UserClassesMasjidStudentID
	}

	if r.UserClassesJoinedAt != nil {
		m.UserClassesJoinedAt = r.UserClassesJoinedAt
	}
	if r.UserClassesLeftAt != nil {
		m.UserClassesLeftAt = r.UserClassesLeftAt
	}

	// Update status lebih dulu
	if r.UserClassesStatus != nil {
		m.UserClassesStatus = *r.UserClassesStatus
	}

	// Update result/completed_at
	if r.UserClassesResult != nil {
		m.UserClassesResult = r.UserClassesResult
	}
	if r.UserClassesCompletedAt != nil {
		m.UserClassesCompletedAt = r.UserClassesCompletedAt
	}

	// Konsistensi ringan:
	// - Jika status=completed & completed_at kosong → isi now
	// - Jika status!=completed → kosongkan result & completed_at
	if m.UserClassesStatus == ucmodel.UserClassStatusCompleted {
		if m.UserClassesCompletedAt == nil {
			now := time.Now()
			m.UserClassesCompletedAt = &now
		}
		// result boleh nil (belum diputuskan lulus/gagal) — biarkan
	} else {
		m.UserClassesResult = nil
		m.UserClassesCompletedAt = nil
	}

	// Model pakai non-pointer UpdatedAt; isi manual agar segera berubah.
	m.UserClassesUpdatedAt = time.Now()
}

/* ===================== QUERIES ===================== */

type ListUserClassQuery struct {
	ClassID         *uuid.UUID `query:"class_id"`          // filter by class
	MasjidID        *uuid.UUID `query:"masjid_id"`         // tenant
	MasjidStudentID *uuid.UUID `query:"masjid_student_id"` // filter by masjid_students

	// status: active|inactive|completed
	Status *string `query:"status"`

	// result: passed|failed (hanya relevan bila status=completed)
	Result *string `query:"result"`

	// ActiveNow: status='active' && left_at IS NULL
	ActiveNow *bool `query:"active_now"`

	// Filter rentang joined
	JoinedFrom *time.Time `query:"joined_from"`
	JoinedTo   *time.Time `query:"joined_to"`

	Limit  int `query:"limit" validate:"omitempty,min=1,max=200"`
	Offset int `query:"offset" validate:"omitempty,min=0"`

	// created_at_desc|created_at_asc|joined_at_desc|joined_at_asc|completed_at_desc|completed_at_asc
	Sort *string `query:"sort"`
}

/* ===================== RESPONSES ===================== */

type UserClassResponse struct {
	UserClassesID              uuid.UUID  `json:"user_classes_id"`
	UserClassesClassID         uuid.UUID  `json:"user_classes_class_id"`
	UserClassesMasjidID        uuid.UUID  `json:"user_classes_masjid_id"`
	UserClassesMasjidStudentID uuid.UUID  `json:"user_classes_masjid_student_id"`

	UserClassesStatus          string     `json:"user_classes_status"`
	UserClassesResult          *string    `json:"user_classes_result,omitempty"`

	UserClassesJoinedAt        *time.Time `json:"user_classes_joined_at,omitempty"`
	UserClassesLeftAt          *time.Time `json:"user_classes_left_at,omitempty"`
	UserClassesCompletedAt     *time.Time `json:"user_classes_completed_at,omitempty"`

	UserClassesCreatedAt       time.Time  `json:"user_classes_created_at"`
	UserClassesUpdatedAt       time.Time  `json:"user_classes_updated_at"`
}

func NewUserClassResponse(m *ucmodel.UserClassesModel) *UserClassResponse {
	if m == nil {
		return nil
	}
	return &UserClassResponse{
		UserClassesID:              m.UserClassesID,
		UserClassesClassID:         m.UserClassesClassID,
		UserClassesMasjidID:        m.UserClassesMasjidID,
		UserClassesMasjidStudentID: m.UserClassesMasjidStudentID,

		UserClassesStatus:      m.UserClassesStatus,
		UserClassesResult:      m.UserClassesResult,
		UserClassesJoinedAt:    m.UserClassesJoinedAt,
		UserClassesLeftAt:      m.UserClassesLeftAt,
		UserClassesCompletedAt: m.UserClassesCompletedAt,

		UserClassesCreatedAt: m.UserClassesCreatedAt,
		UserClassesUpdatedAt: m.UserClassesUpdatedAt,
	}
}
