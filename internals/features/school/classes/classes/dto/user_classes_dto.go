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

type UserClassesCreateRequest struct {
	// Wajib
	UserClassesMasjidStudentID uuid.UUID `json:"user_classes_masjid_student_id" validate:"required"`
	UserClassesClassID         uuid.UUID `json:"user_classes_class_id" validate:"required"`
	UserClassesMasjidID        uuid.UUID `json:"user_classes_masjid_id" validate:"required"`

	// Opsional
	UserClassesStatus string  `json:"user_classes_status" validate:"omitempty,oneof=active inactive completed"`
	UserClassesResult *string `json:"user_classes_result" validate:"omitempty,oneof=passed failed"`

	// Billing ringan
	UserClassesRegisterPaidAt *time.Time `json:"user_classes_register_paid_at"`
	UserClassesPaidUntil      *time.Time `json:"user_classes_paid_until"`
	UserClassesPaidGraceDays  *int16     `json:"user_classes_paid_grace_days" validate:"omitempty,min=0"`

	// Lifecycle enrolment
	UserClassesJoinedAt    *time.Time `json:"user_classes_joined_at"`
	UserClassesLeftAt      *time.Time `json:"user_classes_left_at"`
	UserClassesCompletedAt *time.Time `json:"user_classes_completed_at"`
}

func (r *UserClassesCreateRequest) Normalize() {
	r.UserClassesStatus = strings.ToLower(strings.TrimSpace(r.UserClassesStatus))
	trimPtr := func(pp **string) {
		if pp == nil || *pp == nil {
			return
		}
		v := strings.TrimSpace(**pp)
		if v == "" {
			*pp = nil
		} else {
			v = strings.ToLower(v)
			*pp = &v
		}
	}
	trimPtr(&r.UserClassesResult)
}

func (r UserClassesCreateRequest) ToModel() *m.UserClassesModel {
	now := time.Now()
	status := r.UserClassesStatus
	if status == "" {
		status = "active"
	}
	grace := int16(0)
	if r.UserClassesPaidGraceDays != nil {
		grace = *r.UserClassesPaidGraceDays
	}
	return &m.UserClassesModel{
		// ID dibiarkan kosong, diisi hook BeforeCreate
		UserClassesMasjidStudentID: r.UserClassesMasjidStudentID,
		UserClassesClassID:         r.UserClassesClassID,
		UserClassesMasjidID:        r.UserClassesMasjidID,

		UserClassesStatus:         status,
		UserClassesResult:         r.UserClassesResult,
		UserClassesRegisterPaidAt: r.UserClassesRegisterPaidAt,
		UserClassesPaidUntil:      r.UserClassesPaidUntil,
		UserClassesPaidGraceDays:  grace,

		UserClassesJoinedAt:    r.UserClassesJoinedAt,
		UserClassesLeftAt:      r.UserClassesLeftAt,
		UserClassesCompletedAt: r.UserClassesCompletedAt,

		UserClassesCreatedAt: now,
		UserClassesUpdatedAt: now,
	}
}

type UserClassesResponse struct {
	UserClassesID               uuid.UUID  `json:"user_classes_id"`
	UserClassesMasjidStudentID  uuid.UUID  `json:"user_classes_masjid_student_id"`
	UserClassesClassID          uuid.UUID  `json:"user_classes_class_id"`
	UserClassesMasjidID         uuid.UUID  `json:"user_classes_masjid_id"`
	UserClassesStatus           string     `json:"user_classes_status"`
	UserClassesResult           *string    `json:"user_classes_result,omitempty"`
	UserClassesRegisterPaidAt   *time.Time `json:"user_classes_register_paid_at,omitempty"`
	UserClassesPaidUntil        *time.Time `json:"user_classes_paid_until,omitempty"`
	UserClassesPaidGraceDays    int16      `json:"user_classes_paid_grace_days"`
	UserClassesJoinedAt         *time.Time `json:"user_classes_joined_at,omitempty"`
	UserClassesLeftAt           *time.Time `json:"user_classes_left_at,omitempty"`
	UserClassesCompletedAt      *time.Time `json:"user_classes_completed_at,omitempty"`
	UserClassesCreatedAt        time.Time  `json:"user_classes_created_at"`
	UserClassesUpdatedAt        time.Time  `json:"user_classes_updated_at"`
	UserClassesDeletedAt        *time.Time `json:"user_classes_deleted_at,omitempty"`
}

func FromModelUserClasses(mdl *m.UserClassesModel) UserClassesResponse {
	var deletedAt *time.Time
	if mdl.UserClassesDeletedAt.Valid {
		t := mdl.UserClassesDeletedAt.Time
		deletedAt = &t
	}
	return UserClassesResponse{
		UserClassesID:              mdl.UserClassesID,
		UserClassesMasjidStudentID: mdl.UserClassesMasjidStudentID,
		UserClassesClassID:         mdl.UserClassesClassID,
		UserClassesMasjidID:        mdl.UserClassesMasjidID,
		UserClassesStatus:          mdl.UserClassesStatus,
		UserClassesResult:          mdl.UserClassesResult,
		UserClassesRegisterPaidAt:  mdl.UserClassesRegisterPaidAt,
		UserClassesPaidUntil:       mdl.UserClassesPaidUntil,
		UserClassesPaidGraceDays:   mdl.UserClassesPaidGraceDays,
		UserClassesJoinedAt:        mdl.UserClassesJoinedAt,
		UserClassesLeftAt:          mdl.UserClassesLeftAt,
		UserClassesCompletedAt:     mdl.UserClassesCompletedAt,
		UserClassesCreatedAt:       mdl.UserClassesCreatedAt,
		UserClassesUpdatedAt:       mdl.UserClassesUpdatedAt,
		UserClassesDeletedAt:       deletedAt,
	}
}

/* =========================================================
   PATCH REQUEST — tri-state
   ========================================================= */

type UserClassesPatchRequest struct {
	UserClassesStatus PatchFieldUC[string]  `json:"user_classes_status"` // active|inactive|completed
	UserClassesResult PatchFieldUC[*string] `json:"user_classes_result"` // null → clear

	UserClassesRegisterPaidAt PatchFieldUC[*time.Time] `json:"user_classes_register_paid_at"`
	UserClassesPaidUntil      PatchFieldUC[*time.Time] `json:"user_classes_paid_until"`
	UserClassesPaidGraceDays  PatchFieldUC[int16]      `json:"user_classes_paid_grace_days"` // null → reset 0

	UserClassesJoinedAt    PatchFieldUC[*time.Time] `json:"user_classes_joined_at"`
	UserClassesLeftAt      PatchFieldUC[*time.Time] `json:"user_classes_left_at"`
	UserClassesCompletedAt PatchFieldUC[*time.Time] `json:"user_classes_completed_at"`
}

func (p *UserClassesPatchRequest) Normalize() {
	// status → lowercase
	if p.UserClassesStatus.Present && p.UserClassesStatus.Value != nil {
		v := strings.ToLower(strings.TrimSpace(*p.UserClassesStatus.Value))
		p.UserClassesStatus.Value = &v
	}
	// result → trim+lower (ingat: Value bertipe **string)
	if p.UserClassesResult.Present && p.UserClassesResult.Value != nil && *p.UserClassesResult.Value != nil {
		v := strings.ToLower(strings.TrimSpace(**p.UserClassesResult.Value))
		*p.UserClassesResult.Value = &v
	}
	// timestamps & angka: no-op
}

func (p UserClassesPatchRequest) Apply(uc *m.UserClassesModel) {
	// status
	if p.UserClassesStatus.Present && p.UserClassesStatus.Value != nil {
		uc.UserClassesStatus = *p.UserClassesStatus.Value
	}

	// result (*string dalam Patch → **string di Value)
	if p.UserClassesResult.Present {
		if p.UserClassesResult.Value == nil {
			uc.UserClassesResult = nil
		} else {
			uc.UserClassesResult = *p.UserClassesResult.Value
		}
	}

	// ===== billing (PatchFieldUC[*time.Time] → **time.Time) =====
	if p.UserClassesRegisterPaidAt.Present {
		if p.UserClassesRegisterPaidAt.Value == nil {
			uc.UserClassesRegisterPaidAt = nil
		} else {
			uc.UserClassesRegisterPaidAt = *p.UserClassesRegisterPaidAt.Value
		}
	}
	if p.UserClassesPaidUntil.Present {
		if p.UserClassesPaidUntil.Value == nil {
			uc.UserClassesPaidUntil = nil
		} else {
			uc.UserClassesPaidUntil = *p.UserClassesPaidUntil.Value
		}
	}
	if p.UserClassesPaidGraceDays.Present {
		if p.UserClassesPaidGraceDays.Value == nil {
			uc.UserClassesPaidGraceDays = 0
		} else {
			uc.UserClassesPaidGraceDays = *p.UserClassesPaidGraceDays.Value
		}
	}

	// ===== lifecycle times (PatchFieldUC[*time.Time]) =====
	if p.UserClassesJoinedAt.Present {
		if p.UserClassesJoinedAt.Value == nil {
			uc.UserClassesJoinedAt = nil
		} else {
			uc.UserClassesJoinedAt = *p.UserClassesJoinedAt.Value
		}
	}
	if p.UserClassesLeftAt.Present {
		if p.UserClassesLeftAt.Value == nil {
			uc.UserClassesLeftAt = nil
		} else {
			uc.UserClassesLeftAt = *p.UserClassesLeftAt.Value
		}
	}
	if p.UserClassesCompletedAt.Present {
		if p.UserClassesCompletedAt.Value == nil {
			uc.UserClassesCompletedAt = nil
		} else {
			uc.UserClassesCompletedAt = *p.UserClassesCompletedAt.Value
		}
	}

	uc.UserClassesUpdatedAt = time.Now()
}


/* =========================================================
   LIST QUERY + HELPERS
   ========================================================= */

type ListUserClassesQuery struct {
	Limit     int         `query:"limit"`
	Offset    int         `query:"offset"`
	Status    *string     `query:"status"`      // active|inactive|completed
	Result    *string     `query:"result"`      // passed|failed
	ClassID   *uuid.UUID  `query:"class_id"`
	StudentID *uuid.UUID  `query:"masjid_student_id"`
	JoinedGt  *time.Time  `query:"joined_gt"`
	JoinedLt  *time.Time  `query:"joined_lt"`
	Search    string      `query:"q"`           // opsional
	PaidDueLt *time.Time  `query:"paid_due_lt"` // paid_until < t
	PaidDueGt *time.Time  `query:"paid_due_gt"` // paid_until > t
}

func ToUserClassesResponses(rows []m.UserClassesModel) []UserClassesResponse {
	out := make([]UserClassesResponse, 0, len(rows))
	for i := range rows {
		out = append(out, FromModelUserClasses(&rows[i]))
	}
	return out
}

type PaginationMetaUserClasses struct {
	Total      int64 `json:"total"`
	Limit      int   `json:"limit"`
	Offset     int   `json:"offset"`
	Count      int   `json:"count"`
	NextOffset *int  `json:"next_offset,omitempty"`
	PrevOffset *int  `json:"prev_offset,omitempty"`
	HasMore    bool  `json:"has_more"`
}

func NewPaginationMetaUserClasses(total int64, limit, offset, count int) PaginationMeta {
	if limit <= 0 {
		limit = 20
	}
	meta := PaginationMeta{
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
