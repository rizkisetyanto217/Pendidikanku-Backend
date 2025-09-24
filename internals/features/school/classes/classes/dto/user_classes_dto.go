// file: internals/features/school/enrolments/user_classes/dto/user_class_dto.go
package dto

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"

	m "masjidku_backend/internals/features/school/classes/classes/model"
)

/* =========================================================
   PATCH FIELD — tri-state (absent | null | value)
   ========================================================= */

type PatchFieldUC[T any] struct {
	Present bool
	Value   *T
}

func (p *PatchFieldUC[T]) UnmarshalJSON(b []byte) error {
	p.Present = true
	if string(b) == "null" {
		p.Value = nil
		return nil
	}
	var v T
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	p.Value = &v
	return nil
}

func (p PatchFieldUC[T]) Get() (*T, bool) { return p.Value, p.Present }

/* =========================================================
   CREATE REQUEST / RESPONSE
   ========================================================= */

type UserClassCreateRequest struct {
	// Wajib
	UserClassMasjidStudentID uuid.UUID `json:"user_class_masjid_student_id" validate:"required"`
	UserClassClassID         uuid.UUID `json:"user_class_class_id"          validate:"required"`
	UserClassMasjidID        uuid.UUID `json:"user_class_masjid_id"         validate:"required"`

	// Opsional
	UserClassStatus string  `json:"user_class_status" validate:"omitempty,oneof=active inactive completed"`
	UserClassResult *string `json:"user_class_result" validate:"omitempty,oneof=passed failed"`

	// Billing ringan
	UserClassRegisterPaidAt *time.Time `json:"user_class_register_paid_at"`
	UserClassPaidUntil      *time.Time `json:"user_class_paid_until"`
	UserClassPaidGraceDays  *int16     `json:"user_class_paid_grace_days" validate:"omitempty,min=0"`

	// Lifecycle enrolment
	UserClassJoinedAt    *time.Time `json:"user_class_joined_at"`
	UserClassLeftAt      *time.Time `json:"user_class_left_at"`
	UserClassCompletedAt *time.Time `json:"user_class_completed_at"`
}

func (r *UserClassCreateRequest) Normalize() {
	r.UserClassStatus = strings.ToLower(strings.TrimSpace(r.UserClassStatus))
	if r.UserClassResult != nil {
		v := strings.ToLower(strings.TrimSpace(*r.UserClassResult))
		if v == "" {
			r.UserClassResult = nil
		} else {
			r.UserClassResult = &v
		}
	}
}

func (r UserClassCreateRequest) ToModel() *m.UserClassModel {
	status := r.UserClassStatus
	if status == "" {
		status = m.UserClassStatusActive
	}
	grace := int16(0)
	if r.UserClassPaidGraceDays != nil {
		grace = *r.UserClassPaidGraceDays
	}
	now := time.Now()

	return &m.UserClassModel{
		UserClassMasjidStudentID: r.UserClassMasjidStudentID,
		UserClassClassID:         r.UserClassClassID,
		UserClassMasjidID:        r.UserClassMasjidID,

		UserClassStatus:         status,
		UserClassResult:         r.UserClassResult,
		UserClassRegisterPaidAt: r.UserClassRegisterPaidAt,
		UserClassPaidUntil:      r.UserClassPaidUntil,
		UserClassPaidGraceDays:  grace,

		UserClassJoinedAt:    r.UserClassJoinedAt,
		UserClassLeftAt:      r.UserClassLeftAt,
		UserClassCompletedAt: r.UserClassCompletedAt,

		UserClassCreatedAt: now,
		UserClassUpdatedAt: now,
	}
}

type UserClassResponse struct {
	UserClassID              uuid.UUID  `json:"user_class_id"`
	UserClassMasjidStudentID uuid.UUID  `json:"user_class_masjid_student_id"`
	UserClassClassID         uuid.UUID  `json:"user_class_class_id"`
	UserClassMasjidID        uuid.UUID  `json:"user_class_masjid_id"`
	UserClassStatus          string     `json:"user_class_status"`
	UserClassResult          *string    `json:"user_class_result,omitempty"`
	UserClassRegisterPaidAt  *time.Time `json:"user_class_register_paid_at,omitempty"`
	UserClassPaidUntil       *time.Time `json:"user_class_paid_until,omitempty"`
	UserClassPaidGraceDays   int16      `json:"user_class_paid_grace_days"`
	UserClassJoinedAt        *time.Time `json:"user_class_joined_at,omitempty"`
	UserClassLeftAt          *time.Time `json:"user_class_left_at,omitempty"`
	UserClassCompletedAt     *time.Time `json:"user_class_completed_at,omitempty"`
	UserClassCreatedAt       time.Time  `json:"user_class_created_at"`
	UserClassUpdatedAt       time.Time  `json:"user_class_updated_at"`
	UserClassDeletedAt       *time.Time `json:"user_class_deleted_at,omitempty"`
}

func FromModelUserClass(mdl *m.UserClassModel) UserClassResponse {
	var deletedAt *time.Time
	if mdl.UserClassDeletedAt.Valid {
		t := mdl.UserClassDeletedAt.Time
		deletedAt = &t
	}
	return UserClassResponse{
		UserClassID:              mdl.UserClassID,
		UserClassMasjidStudentID: mdl.UserClassMasjidStudentID,
		UserClassClassID:         mdl.UserClassClassID,
		UserClassMasjidID:        mdl.UserClassMasjidID,
		UserClassStatus:          mdl.UserClassStatus,
		UserClassResult:          mdl.UserClassResult,
		UserClassRegisterPaidAt:  mdl.UserClassRegisterPaidAt,
		UserClassPaidUntil:       mdl.UserClassPaidUntil,
		UserClassPaidGraceDays:   mdl.UserClassPaidGraceDays,
		UserClassJoinedAt:        mdl.UserClassJoinedAt,
		UserClassLeftAt:          mdl.UserClassLeftAt,
		UserClassCompletedAt:     mdl.UserClassCompletedAt,
		UserClassCreatedAt:       mdl.UserClassCreatedAt,
		UserClassUpdatedAt:       mdl.UserClassUpdatedAt,
		UserClassDeletedAt:       deletedAt,
	}
}

/* =========================================================
   PATCH REQUEST — tri-state
   ========================================================= */

type UserClassPatchRequest struct {
	UserClassStatus PatchFieldUC[string]  `json:"user_class_status"` // active|inactive|completed
	UserClassResult PatchFieldUC[*string] `json:"user_class_result"` // null → clear

	UserClassRegisterPaidAt PatchFieldUC[*time.Time] `json:"user_class_register_paid_at"`
	UserClassPaidUntil      PatchFieldUC[*time.Time] `json:"user_class_paid_until"`
	UserClassPaidGraceDays  PatchFieldUC[int16]      `json:"user_class_paid_grace_days"` // null → reset 0

	UserClassJoinedAt    PatchFieldUC[*time.Time] `json:"user_class_joined_at"`
	UserClassLeftAt      PatchFieldUC[*time.Time] `json:"user_class_left_at"`
	UserClassCompletedAt PatchFieldUC[*time.Time] `json:"user_class_completed_at"`
}

func (p *UserClassPatchRequest) Normalize() {
	if p.UserClassStatus.Present && p.UserClassStatus.Value != nil {
		v := strings.ToLower(strings.TrimSpace(*p.UserClassStatus.Value))
		p.UserClassStatus.Value = &v
	}
	if p.UserClassResult.Present && p.UserClassResult.Value != nil && *p.UserClassResult.Value != nil {
		v := strings.ToLower(strings.TrimSpace(**p.UserClassResult.Value))
		*p.UserClassResult.Value = &v
	}
}

func (p UserClassPatchRequest) Apply(uc *m.UserClassModel) {
	// status
	if p.UserClassStatus.Present && p.UserClassStatus.Value != nil {
		uc.UserClassStatus = *p.UserClassStatus.Value
	}

	// result (*string dalam Patch → **string di Value)
	if p.UserClassResult.Present {
		if p.UserClassResult.Value == nil {
			uc.UserClassResult = nil
		} else {
			uc.UserClassResult = *p.UserClassResult.Value
		}
	}

	// billing
	if p.UserClassRegisterPaidAt.Present {
		if p.UserClassRegisterPaidAt.Value == nil {
			uc.UserClassRegisterPaidAt = nil
		} else {
			uc.UserClassRegisterPaidAt = *p.UserClassRegisterPaidAt.Value
		}
	}
	if p.UserClassPaidUntil.Present {
		if p.UserClassPaidUntil.Value == nil {
			uc.UserClassPaidUntil = nil
		} else {
			uc.UserClassPaidUntil = *p.UserClassPaidUntil.Value
		}
	}
	if p.UserClassPaidGraceDays.Present {
		if p.UserClassPaidGraceDays.Value == nil {
			uc.UserClassPaidGraceDays = 0
		} else {
			uc.UserClassPaidGraceDays = *p.UserClassPaidGraceDays.Value
		}
	}

	// lifecycle times
	if p.UserClassJoinedAt.Present {
		if p.UserClassJoinedAt.Value == nil {
			uc.UserClassJoinedAt = nil
		} else {
			uc.UserClassJoinedAt = *p.UserClassJoinedAt.Value
		}
	}
	if p.UserClassLeftAt.Present {
		if p.UserClassLeftAt.Value == nil {
			uc.UserClassLeftAt = nil
		} else {
			uc.UserClassLeftAt = *p.UserClassLeftAt.Value
		}
	}
	if p.UserClassCompletedAt.Present {
		if p.UserClassCompletedAt.Value == nil {
			uc.UserClassCompletedAt = nil
		} else {
			uc.UserClassCompletedAt = *p.UserClassCompletedAt.Value
		}
	}

	uc.UserClassUpdatedAt = time.Now()
}

/* =========================================================
   LIST QUERY + HELPERS
   ========================================================= */

type ListUserClassQuery struct {
	Limit     int        `query:"limit"`
	Offset    int        `query:"offset"`
	Status    *string    `query:"status"` // active|inactive|completed
	Result    *string    `query:"result"` // passed|failed
	ClassID   *uuid.UUID `query:"class_id"`
	StudentID *uuid.UUID `query:"masjid_student_id"`
	JoinedGt  *time.Time `query:"joined_gt"`
	JoinedLt  *time.Time `query:"joined_lt"`
	Search    string     `query:"q"`           // opsional
	PaidDueLt *time.Time `query:"paid_due_lt"` // paid_until < t
	PaidDueGt *time.Time `query:"paid_due_gt"` // paid_until > t
}

type PaginationMetaUserClass struct {
	Total      int64 `json:"total"`
	Limit      int   `json:"limit"`
	Offset     int   `json:"offset"`
	Count      int   `json:"count"`
	NextOffset *int  `json:"next_offset,omitempty"`
	PrevOffset *int  `json:"prev_offset,omitempty"`
	HasMore    bool  `json:"has_more"`
}

func NewPaginationMetaUserClass(total int64, limit, offset, count int) PaginationMetaUserClass {
	if limit <= 0 {
		limit = 20
	}
	meta := PaginationMetaUserClass{
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		Count:   count,
		HasMore: int64(offset+count) < total,
	}
	if offset > 0 {
		prev := offset - limit
		if prev < 0 {
			prev = 0
		}
		meta.PrevOffset = &prev
	}
	if meta.HasMore {
		next := offset + count
		meta.NextOffset = &next
	}
	return meta
}

/* =========================================================
   BULK MAPPERS
   ========================================================= */

func ToUserClassResponses(rows []m.UserClassModel) []UserClassResponse {
	out := make([]UserClassResponse, 0, len(rows))
	for i := range rows {
		out = append(out, FromModelUserClass(&rows[i]))
	}
	return out
}
