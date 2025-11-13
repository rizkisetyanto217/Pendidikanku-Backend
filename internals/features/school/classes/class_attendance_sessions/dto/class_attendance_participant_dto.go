// file: internals/features/school/sessions/sessions/dto/class_attendance_session_participant_dto.go
package dto

import (
	"bytes"
	"encoding"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	attendanceModel "schoolku_backend/internals/features/school/classes/class_attendance_sessions/model"
)

/* ===================== PatchField (tri-state) ===================== */

type PatchFieldUserAttendance[T any] struct {
	Present bool
	Value   *T
}

/*************  ✨ Windsurf Command ⭐  *************/
// UnmarshalJSON parses a JSON-encoded value and stores it in the Value field.
//
// If the JSON value is "null", the Value field is set to nil.
// Otherwise, the JSON value is unmarshaled into the Value field.
// If unmarshaling fails, an error is returned.
// The Present field is always set to true when this function is called.
/*******  d482f074-d516-40aa-8edc-24ffe5a83b84  *******/func (p *PatchFieldUserAttendance[T]) UnmarshalJSON(b []byte) error {
	p.Present = true
	if bytes.Equal(bytes.TrimSpace(b), []byte("null")) {
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

func (p PatchFieldUserAttendance[T]) Get() (*T, bool) { return p.Value, p.Present }

/* ===================== CSV helper for QueryParser ===================== */

type CSV []string

var _ encoding.TextUnmarshaler = (*CSV)(nil)

func (c *CSV) UnmarshalText(text []byte) error {
	s := strings.TrimSpace(string(text))
	if s == "" {
		*c = nil
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	*c = out
	return nil
}

/* ===================== URL ops enums & helpers ===================== */

type URLOp string

const (
	URLOpUpsert URLOp = "upsert"
	URLOpDelete URLOp = "delete"
)

var allowedKinds = map[string]struct{}{
	"image": {}, "video": {}, "attachment": {}, "link": {}, "audio": {},
}

func normalizeKind(s string) (string, error) {
	k := strings.ToLower(strings.TrimSpace(s))
	if _, ok := allowedKinds[k]; !ok {
		return "", errors.New("invalid kind")
	}
	return k, nil
}

/* ===================== URL operation DTO ===================== */

type ClassAttendanceSessionParticipantURLOpDTO struct {
	Op URLOp `json:"op" validate:"required,oneof=upsert delete"`

	ID   *uuid.UUID `json:"id,omitempty" validate:"omitempty,uuid4"`
	Kind *string    `json:"kind,omitempty" validate:"omitempty,oneof=image video attachment link audio"`

	Label              *string    `json:"label,omitempty" validate:"omitempty,max=160"`
	Order              *int       `json:"order,omitempty" validate:"omitempty,min=0"`
	IsPrimary          *bool      `json:"is_primary,omitempty"`
	URL                *string    `json:"url,omitempty" validate:"omitempty,max=2048"`
	ObjectKey          *string    `json:"object_key,omitempty"`
	URLOld             *string    `json:"url_old,omitempty"`
	ObjectKeyOld       *string    `json:"object_key_old,omitempty"`
	DeletePendingUntil *time.Time `json:"delete_pending_until,omitempty"`

	UploaderTeacherID *uuid.UUID `json:"uploader_teacher_id,omitempty" validate:"omitempty,uuid4"`
	UploaderStudentID *uuid.UUID `json:"uploader_student_id,omitempty" validate:"omitempty,uuid4"`
}

/* ===================== Create DTO ===================== */

type ClassAttendanceSessionParticipantCreateRequest struct {
	SchoolID  uuid.UUID `json:"school_id"  validate:"required,uuid4"`
	SessionID uuid.UUID `json:"session_id" validate:"required,uuid4"`

	// participant
	Kind            *string    `json:"kind,omitempty" validate:"omitempty,oneof=student teacher assistant guest"` // default: student
	SchoolStudentID *uuid.UUID `json:"school_student_id,omitempty" validate:"omitempty,uuid4"`
	SchoolTeacherID *uuid.UUID `json:"school_teacher_id,omitempty" validate:"omitempty,uuid4"`
	TeacherRole     *string    `json:"teacher_role,omitempty" validate:"omitempty,oneof=primary co substitute observer assistant"`

	// state kehadiran (enum attendance_state_enum)
	State    *string    `json:"state,omitempty" validate:"omitempty,oneof=present absent late excused sick leave"`
	TypeID   *uuid.UUID `json:"type_id,omitempty" validate:"omitempty,uuid4"`
	Desc     *string    `json:"desc,omitempty"`
	Score    *float64   `json:"score,omitempty" validate:"omitempty,gte=0,lte=100"`
	IsPassed *bool      `json:"is_passed,omitempty"`

	CheckinAt  *time.Time `json:"checkin_at,omitempty"`
	CheckoutAt *time.Time `json:"checkout_at,omitempty"`

	MarkedAt    *time.Time `json:"marked_at,omitempty"`
	MarkedByTID *uuid.UUID `json:"marked_by_teacher_id,omitempty" validate:"omitempty,uuid4"`
	Method      *string    `json:"method,omitempty" validate:"omitempty,oneof=manual qr geo import api self"`

	Lat         *float64 `json:"lat,omitempty"`
	Lng         *float64 `json:"lng,omitempty"`
	DistanceM   *int     `json:"distance_m,omitempty" validate:"omitempty,min=0"`
	LateSeconds *int     `json:"late_seconds,omitempty" validate:"omitempty,min=0"`

	UserNote    *string    `json:"user_note,omitempty"`
	TeacherNote *string    `json:"teacher_note,omitempty"`
	LockedAt    *time.Time `json:"locked_at,omitempty"`

	URLs []ClassAttendanceSessionParticipantURLOpDTO `json:"urls,omitempty" validate:"omitempty,dive"`
}

func (r ClassAttendanceSessionParticipantCreateRequest) ToModel() attendanceModel.ClassAttendanceSessionParticipantModel {
	// kind default → student
	kindStr := "student"
	if r.Kind != nil && strings.TrimSpace(*r.Kind) != "" {
		kindStr = strings.ToLower(strings.TrimSpace(*r.Kind))
	}

	// state default → present
	stateStr := "present"
	if r.State != nil && strings.TrimSpace(*r.State) != "" {
		stateStr = strings.ToLower(strings.TrimSpace(*r.State))
	}

	m := attendanceModel.ClassAttendanceSessionParticipantModel{
		ClassAttendanceSessionParticipantSchoolID:  r.SchoolID,
		ClassAttendanceSessionParticipantSessionID: r.SessionID,

		ClassAttendanceSessionParticipantKind:  attendanceModel.ParticipantKind(kindStr),
		ClassAttendanceSessionParticipantState: attendanceModel.AttendanceState(stateStr),

		ClassAttendanceSessionParticipantDesc:     r.Desc,
		ClassAttendanceSessionParticipantScore:    r.Score,
		ClassAttendanceSessionParticipantIsPassed: r.IsPassed,

		ClassAttendanceSessionParticipantCheckinAt:  r.CheckinAt,
		ClassAttendanceSessionParticipantCheckoutAt: r.CheckoutAt,

		ClassAttendanceSessionParticipantMarkedAt: r.MarkedAt,

		ClassAttendanceSessionParticipantLat:       r.Lat,
		ClassAttendanceSessionParticipantLng:       r.Lng,
		ClassAttendanceSessionParticipantDistanceM: r.DistanceM,

		ClassAttendanceSessionParticipantLateSeconds: r.LateSeconds,

		ClassAttendanceSessionParticipantUserNote:    r.UserNote,
		ClassAttendanceSessionParticipantTeacherNote: r.TeacherNote,
		ClassAttendanceSessionParticipantLockedAt:    r.LockedAt,
	}

	// relasi student/teacher
	if r.SchoolStudentID != nil {
		m.ClassAttendanceSessionParticipantSchoolStudentID = r.SchoolStudentID
	}
	if r.SchoolTeacherID != nil {
		m.ClassAttendanceSessionParticipantSchoolTeacherID = r.SchoolTeacherID
	}
	if r.TeacherRole != nil && strings.TrimSpace(*r.TeacherRole) != "" {
		role := attendanceModel.TeacherRole(strings.ToLower(strings.TrimSpace(*r.TeacherRole)))
		m.ClassAttendanceSessionParticipantTeacherRole = &role
	}

	// type
	if r.TypeID != nil {
		m.ClassAttendanceSessionParticipantTypeID = r.TypeID
	}
	if r.MarkedByTID != nil {
		m.ClassAttendanceSessionParticipantMarkedByTeacherID = r.MarkedByTID
	}
	if r.Method != nil && strings.TrimSpace(*r.Method) != "" {
		mv := strings.ToLower(strings.TrimSpace(*r.Method))
		m.ClassAttendanceSessionParticipantMethod = &mv
	}

	return m
}

/* ===================== Patch DTO (tri-state) ===================== */

type ClassAttendanceSessionParticipantPatchRequest struct {
	ParticipantID uuid.UUID `json:"participant_id" validate:"required,uuid4"`

	// basic
	State    PatchFieldUserAttendance[string]    `json:"state,omitempty"` // present|absent|late|excused|sick|leave
	TypeID   PatchFieldUserAttendance[uuid.UUID] `json:"type_id,omitempty"`
	Desc     PatchFieldUserAttendance[string]    `json:"desc,omitempty"`
	Score    PatchFieldUserAttendance[float64]   `json:"score,omitempty"`
	IsPassed PatchFieldUserAttendance[bool]      `json:"is_passed,omitempty"`

	CheckinAt  PatchFieldUserAttendance[time.Time] `json:"checkin_at,omitempty"`
	CheckoutAt PatchFieldUserAttendance[time.Time] `json:"checkout_at,omitempty"`

	MarkedAt    PatchFieldUserAttendance[time.Time] `json:"marked_at,omitempty"`
	MarkedByTID PatchFieldUserAttendance[uuid.UUID] `json:"marked_by_teacher_id,omitempty"`

	Method PatchFieldUserAttendance[string] `json:"method,omitempty"` // manual|qr|geo|import|api|self

	Lat         PatchFieldUserAttendance[float64] `json:"lat,omitempty"`
	Lng         PatchFieldUserAttendance[float64] `json:"lng,omitempty"`
	DistanceM   PatchFieldUserAttendance[int]     `json:"distance_m,omitempty"`
	LateSeconds PatchFieldUserAttendance[int]     `json:"late_seconds,omitempty"`

	UserNote    PatchFieldUserAttendance[string]    `json:"user_note,omitempty"`
	TeacherNote PatchFieldUserAttendance[string]    `json:"teacher_note,omitempty"`
	LockedAt    PatchFieldUserAttendance[time.Time] `json:"locked_at,omitempty"`

	URLs []ClassAttendanceSessionParticipantURLOpDTO `json:"urls,omitempty" validate:"omitempty,dive"`
}

func (p ClassAttendanceSessionParticipantPatchRequest) ApplyPatch(m *attendanceModel.ClassAttendanceSessionParticipantModel) error {
	if v, ok := p.State.Get(); ok {
		if v == nil || strings.TrimSpace(*v) == "" {
			// kalau dikosongkan → default present
			m.ClassAttendanceSessionParticipantState = attendanceModel.AttendanceState("present")
		} else {
			m.ClassAttendanceSessionParticipantState = attendanceModel.AttendanceState(
				strings.ToLower(strings.TrimSpace(*v)),
			)
		}
	}
	if v, ok := p.TypeID.Get(); ok {
		m.ClassAttendanceSessionParticipantTypeID = v
	}
	if v, ok := p.Desc.Get(); ok {
		m.ClassAttendanceSessionParticipantDesc = v
	}
	if v, ok := p.Score.Get(); ok {
		m.ClassAttendanceSessionParticipantScore = v
	}
	if v, ok := p.IsPassed.Get(); ok {
		m.ClassAttendanceSessionParticipantIsPassed = v
	}
	if v, ok := p.CheckinAt.Get(); ok {
		m.ClassAttendanceSessionParticipantCheckinAt = v
	}
	if v, ok := p.CheckoutAt.Get(); ok {
		m.ClassAttendanceSessionParticipantCheckoutAt = v
	}
	if v, ok := p.MarkedAt.Get(); ok {
		m.ClassAttendanceSessionParticipantMarkedAt = v
	}
	if v, ok := p.MarkedByTID.Get(); ok {
		m.ClassAttendanceSessionParticipantMarkedByTeacherID = v
	}
	if v, ok := p.Method.Get(); ok {
		if v == nil || strings.TrimSpace(*v) == "" {
			m.ClassAttendanceSessionParticipantMethod = nil
		} else {
			mv := strings.ToLower(strings.TrimSpace(*v))
			m.ClassAttendanceSessionParticipantMethod = &mv
		}
	}
	if v, ok := p.Lat.Get(); ok {
		m.ClassAttendanceSessionParticipantLat = v
	}
	if v, ok := p.Lng.Get(); ok {
		m.ClassAttendanceSessionParticipantLng = v
	}
	if v, ok := p.DistanceM.Get(); ok {
		m.ClassAttendanceSessionParticipantDistanceM = v
	}
	if v, ok := p.LateSeconds.Get(); ok {
		m.ClassAttendanceSessionParticipantLateSeconds = v
	}
	if v, ok := p.UserNote.Get(); ok {
		m.ClassAttendanceSessionParticipantUserNote = v
	}
	if v, ok := p.TeacherNote.Get(); ok {
		m.ClassAttendanceSessionParticipantTeacherNote = v
	}
	if v, ok := p.LockedAt.Get(); ok {
		m.ClassAttendanceSessionParticipantLockedAt = v
	}
	return nil
}

/* ===================== Query DTO (for List) ===================== */

type ListClassAttendanceSessionParticipantQuery struct {
	Search CSV `query:"search"`

	StateIn  CSV `query:"state_in"`  // present|absent|late|excused|sick|leave
	MethodIn CSV `query:"method_in"` // manual|qr|geo|import|api|self
	KindIn   CSV `query:"kind_in"`   // student|teacher|assistant|guest

	SessionID       string `query:"session_id"`
	SchoolStudentID string `query:"school_student_id"`
	SchoolTeacherID string `query:"school_teacher_id"`
	TypeID          string `query:"type_id"`
	MarkedByTID     string `query:"marked_by_teacher_id"`

	CreatedGE string `query:"created_ge"`
	CreatedLE string `query:"created_le"`
	MarkedGE  string `query:"marked_ge"`
	MarkedLE  string `query:"marked_le"`
}

/* ===================== URL Mutations ===================== */

type URLMutations struct {
	ToCreate []attendanceModel.ClassAttendanceSessionParticipantURLModel
	ToUpdate []attendanceModel.ClassAttendanceSessionParticipantURLModel
	ToDelete []uuid.UUID
}

func BuildURLMutations(
	participantID uuid.UUID,
	schoolID uuid.UUID,
	ops []ClassAttendanceSessionParticipantURLOpDTO,
) (URLMutations, error) {
	var out URLMutations
	for _, op := range ops {
		switch op.Op {
		case URLOpUpsert:
			if op.ID == nil {
				// CREATE
				if op.Kind == nil {
					return out, errors.New("kind required for create")
				}
				kind, err := normalizeKind(*op.Kind)
				if err != nil {
					return out, err
				}
				row := attendanceModel.ClassAttendanceSessionParticipantURLModel{
					ClassAttendanceSessionParticipantURLSchoolID:           schoolID,
					ClassAttendanceSessionParticipantURLParticipantID:      participantID,
					ClassAttendanceSessionParticipantURLKind:               kind,
					ClassAttendanceSessionParticipantURLLabel:              op.Label,
					ClassAttendanceSessionParticipantURLOrder:              pint(op.Order),
					ClassAttendanceSessionParticipantURLIsPrimary:          pbool(op.IsPrimary),
					ClassAttendanceSessionParticipantURL:                   op.URL,
					ClassAttendanceSessionParticipantURLObjectKey:          op.ObjectKey,
					ClassAttendanceSessionParticipantURLOld:                op.URLOld,
					ClassAttendanceSessionParticipantURLObjectKeyOld:       op.ObjectKeyOld,
					ClassAttendanceSessionParticipantURLDeletePendingUntil: op.DeletePendingUntil,
					ClassAttendanceSessionParticipantURLUploaderTeacherID:  op.UploaderTeacherID,
					ClassAttendanceSessionParticipantURLUploaderStudentID:  op.UploaderStudentID,
				}
				out.ToCreate = append(out.ToCreate, row)
			} else {
				// UPDATE
				kind := ""
				if op.Kind != nil {
					var err error
					kind, err = normalizeKind(*op.Kind)
					if err != nil {
						return out, err
					}
				}
				row := attendanceModel.ClassAttendanceSessionParticipantURLModel{
					ClassAttendanceSessionParticipantURLID:                 *op.ID,
					ClassAttendanceSessionParticipantURLLabel:              op.Label,
					ClassAttendanceSessionParticipantURLOrder:              pint(op.Order),
					ClassAttendanceSessionParticipantURLIsPrimary:          pbool(op.IsPrimary),
					ClassAttendanceSessionParticipantURL:                   op.URL,
					ClassAttendanceSessionParticipantURLObjectKey:          op.ObjectKey,
					ClassAttendanceSessionParticipantURLOld:                op.URLOld,
					ClassAttendanceSessionParticipantURLObjectKeyOld:       op.ObjectKeyOld,
					ClassAttendanceSessionParticipantURLDeletePendingUntil: op.DeletePendingUntil,
					ClassAttendanceSessionParticipantURLUploaderTeacherID:  op.UploaderTeacherID,
					ClassAttendanceSessionParticipantURLUploaderStudentID:  op.UploaderStudentID,
				}
				if op.Kind != nil {
					row.ClassAttendanceSessionParticipantURLKind = kind
				}
				out.ToUpdate = append(out.ToUpdate, row)
			}
		case URLOpDelete:
			if op.ID == nil {
				return out, errors.New("id required for delete")
			}
			out.ToDelete = append(out.ToDelete, *op.ID)
		default:
			return out, errors.New("unsupported op")
		}
	}
	return out, nil
}

/* ===================== small helpers ===================== */

func pbool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

func pint(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}
