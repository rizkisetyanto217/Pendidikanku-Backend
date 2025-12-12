// file: internals/features/school/sessions/sessions/dto/class_attendance_session_participant_dto.go
package dto

import (
	"bytes"
	"encoding"
	"encoding/json"
	"errors"
	"strings"
	"time"

	dbtime "madinahsalam_backend/internals/helpers/dbtime"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	attendanceModel "madinahsalam_backend/internals/features/school/class_others/class_attendance_sessions/model"
)

/* ===================== PatchField (tri-state) ===================== */

type PatchFieldUserAttendance[T any] struct {
	Present bool
	Value   *T
}

func (p *PatchFieldUserAttendance[T]) UnmarshalJSON(b []byte) error {
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

// JSON TAG = persis nama kolom di model
type ClassAttendanceSessionParticipantCreateRequest struct {
	ClassAttendanceSessionParticipantSchoolID  uuid.UUID `json:"class_attendance_session_participant_school_id"  validate:"required,uuid4"`
	ClassAttendanceSessionParticipantSessionID uuid.UUID `json:"class_attendance_session_participant_session_id" validate:"required,uuid4"`

	// participant
	ClassAttendanceSessionParticipantKind *string `json:"class_attendance_session_participant_kind,omitempty" validate:"omitempty,oneof=student teacher assistant guest"`

	// relasi detail
	ClassAttendanceSessionParticipantSchoolStudentID *uuid.UUID `json:"class_attendance_session_participant_school_student_id,omitempty" validate:"omitempty,uuid4"`
	ClassAttendanceSessionParticipantSchoolTeacherID *uuid.UUID `json:"class_attendance_session_participant_school_teacher_id,omitempty" validate:"omitempty,uuid4"`
	ClassAttendanceSessionParticipantTeacherRole     *string    `json:"class_attendance_session_participant_teacher_role,omitempty" validate:"omitempty,oneof=primary co substitute observer assistant"`

	// state kehadiran
	ClassAttendanceSessionParticipantState  *string    `json:"class_attendance_session_participant_state,omitempty" validate:"omitempty,oneof=present absent late excused sick leave unmarked"`
	ClassAttendanceSessionParticipantTypeID *uuid.UUID `json:"class_attendance_session_participant_type_id,omitempty" validate:"omitempty,uuid4"`

	// penilaian / desc
	ClassAttendanceSessionParticipantDesc     *string  `json:"class_attendance_session_participant_desc,omitempty"`
	ClassAttendanceSessionParticipantScore    *float64 `json:"class_attendance_session_participant_score,omitempty" validate:"omitempty,gte=0,lte=100"`
	ClassAttendanceSessionParticipantIsPassed *bool    `json:"class_attendance_session_participant_is_passed,omitempty"`

	// waktu
	ClassAttendanceSessionParticipantCheckinAt  *time.Time `json:"class_attendance_session_participant_checkin_at,omitempty"`
	ClassAttendanceSessionParticipantCheckoutAt *time.Time `json:"class_attendance_session_participant_checkout_at,omitempty"`

	// meta penandaan
	ClassAttendanceSessionParticipantMarkedAt          *time.Time `json:"class_attendance_session_participant_marked_at,omitempty"`
	ClassAttendanceSessionParticipantMarkedByTeacherID *uuid.UUID `json:"class_attendance_session_participant_marked_by_teacher_id,omitempty" validate:"omitempty,uuid4"`

	// metode
	ClassAttendanceSessionParticipantMethod *string `json:"class_attendance_session_participant_method,omitempty" validate:"omitempty,oneof=manual qr geo import api self"`

	// geo
	ClassAttendanceSessionParticipantLat       *float64 `json:"class_attendance_session_participant_lat,omitempty"`
	ClassAttendanceSessionParticipantLng       *float64 `json:"class_attendance_session_participant_lng,omitempty"`
	ClassAttendanceSessionParticipantDistanceM *int     `json:"class_attendance_session_participant_distance_m,omitempty" validate:"omitempty,min=0"`

	// telat
	ClassAttendanceSessionParticipantLateSeconds *int `json:"class_attendance_session_participant_late_seconds,omitempty" validate:"omitempty,min=0"`

	// snapshot users_profile (opsional; biasanya diisi dari backend, bukan dari client langsung)
	ClassAttendanceSessionParticipantUserProfileNameSnapshot              *string `json:"class_attendance_session_participant_user_profile_name_snapshot,omitempty"`
	ClassAttendanceSessionParticipantUserProfileAvatarURLSnapshot         *string `json:"class_attendance_session_participant_user_profile_avatar_url_snapshot,omitempty"`
	ClassAttendanceSessionParticipantUserProfileWhatsappURLSnapshot       *string `json:"class_attendance_session_participant_user_profile_whatsapp_url_snapshot,omitempty"`
	ClassAttendanceSessionParticipantUserProfileParentNameSnapshot        *string `json:"class_attendance_session_participant_user_profile_parent_name_snapshot,omitempty"`
	ClassAttendanceSessionParticipantUserProfileParentWhatsappURLSnapshot *string `json:"class_attendance_session_participant_user_profile_parent_whatsapp_url_snapshot,omitempty"`
	ClassAttendanceSessionParticipantUserProfileGenderSnapshot            *string `json:"class_attendance_session_participant_user_profile_gender_snapshot,omitempty"`

	// notes
	ClassAttendanceSessionParticipantUserNote    *string    `json:"class_attendance_session_participant_user_note,omitempty"`
	ClassAttendanceSessionParticipantTeacherNote *string    `json:"class_attendance_session_participant_teacher_note,omitempty"`
	ClassAttendanceSessionParticipantLockedAt    *time.Time `json:"class_attendance_session_participant_locked_at,omitempty"`

	URLs []ClassAttendanceSessionParticipantURLOpDTO `json:"urls,omitempty" validate:"omitempty,dive"`
}

func (r ClassAttendanceSessionParticipantCreateRequest) ToModel() attendanceModel.ClassAttendanceSessionParticipantModel {
	// ================================
	// KIND auto-detect:
	// 1) Kalau client isi explicit → pakai itu
	// 2) Kalau ada teacher_id      → "teacher"
	// 3) Selain itu                → "student"
	// ================================
	var kindStr string
	if r.ClassAttendanceSessionParticipantKind != nil && strings.TrimSpace(*r.ClassAttendanceSessionParticipantKind) != "" {
		// explicit dari client
		kindStr = strings.ToLower(strings.TrimSpace(*r.ClassAttendanceSessionParticipantKind))
	} else if r.ClassAttendanceSessionParticipantSchoolTeacherID != nil && *r.ClassAttendanceSessionParticipantSchoolTeacherID != uuid.Nil {
		// auto: ada teacher_id
		kindStr = string(attendanceModel.ParticipantKindTeacher)
	} else {
		// default: student
		kindStr = string(attendanceModel.ParticipantKindStudent)
	}

	// ================================
	// STATE default → present
	// ================================
	stateStr := string(attendanceModel.AttendanceStatePresent)
	if r.ClassAttendanceSessionParticipantState != nil && strings.TrimSpace(*r.ClassAttendanceSessionParticipantState) != "" {
		stateStr = strings.ToLower(strings.TrimSpace(*r.ClassAttendanceSessionParticipantState))
	}

	// normalisasi waktu ke UTC (kalau ada)
	var checkinUTC, checkoutUTC, markedUTC, lockedUTC *time.Time

	if r.ClassAttendanceSessionParticipantCheckinAt != nil {
		t := r.ClassAttendanceSessionParticipantCheckinAt.UTC()
		checkinUTC = &t
	}
	if r.ClassAttendanceSessionParticipantCheckoutAt != nil {
		t := r.ClassAttendanceSessionParticipantCheckoutAt.UTC()
		checkoutUTC = &t
	}
	if r.ClassAttendanceSessionParticipantMarkedAt != nil {
		t := r.ClassAttendanceSessionParticipantMarkedAt.UTC()
		markedUTC = &t
	}
	if r.ClassAttendanceSessionParticipantLockedAt != nil {
		t := r.ClassAttendanceSessionParticipantLockedAt.UTC()
		lockedUTC = &t
	}

	m := attendanceModel.ClassAttendanceSessionParticipantModel{
		ClassAttendanceSessionParticipantSchoolID:  r.ClassAttendanceSessionParticipantSchoolID,
		ClassAttendanceSessionParticipantSessionID: r.ClassAttendanceSessionParticipantSessionID,

		ClassAttendanceSessionParticipantKind:  attendanceModel.ParticipantKind(kindStr),
		ClassAttendanceSessionParticipantState: attendanceModel.AttendanceState(stateStr),

		ClassAttendanceSessionParticipantDesc:     r.ClassAttendanceSessionParticipantDesc,
		ClassAttendanceSessionParticipantScore:    r.ClassAttendanceSessionParticipantScore,
		ClassAttendanceSessionParticipantIsPassed: r.ClassAttendanceSessionParticipantIsPassed,

		ClassAttendanceSessionParticipantCheckinAt:  checkinUTC,
		ClassAttendanceSessionParticipantCheckoutAt: checkoutUTC,

		ClassAttendanceSessionParticipantMarkedAt: markedUTC,

		ClassAttendanceSessionParticipantLat:       r.ClassAttendanceSessionParticipantLat,
		ClassAttendanceSessionParticipantLng:       r.ClassAttendanceSessionParticipantLng,
		ClassAttendanceSessionParticipantDistanceM: r.ClassAttendanceSessionParticipantDistanceM,

		ClassAttendanceSessionParticipantLateSeconds: r.ClassAttendanceSessionParticipantLateSeconds,

		// snapshot users_profile
		ClassAttendanceSessionParticipantUserProfileNameSnapshot:              r.ClassAttendanceSessionParticipantUserProfileNameSnapshot,
		ClassAttendanceSessionParticipantUserProfileAvatarURLSnapshot:         r.ClassAttendanceSessionParticipantUserProfileAvatarURLSnapshot,
		ClassAttendanceSessionParticipantUserProfileWhatsappURLSnapshot:       r.ClassAttendanceSessionParticipantUserProfileWhatsappURLSnapshot,
		ClassAttendanceSessionParticipantUserProfileParentNameSnapshot:        r.ClassAttendanceSessionParticipantUserProfileParentNameSnapshot,
		ClassAttendanceSessionParticipantUserProfileParentWhatsappURLSnapshot: r.ClassAttendanceSessionParticipantUserProfileParentWhatsappURLSnapshot,
		ClassAttendanceSessionParticipantUserProfileGenderSnapshot:            r.ClassAttendanceSessionParticipantUserProfileGenderSnapshot,

		ClassAttendanceSessionParticipantUserNote:    r.ClassAttendanceSessionParticipantUserNote,
		ClassAttendanceSessionParticipantTeacherNote: r.ClassAttendanceSessionParticipantTeacherNote,
		ClassAttendanceSessionParticipantLockedAt:    lockedUTC,
	}

	// relasi student/teacher
	if r.ClassAttendanceSessionParticipantSchoolStudentID != nil {
		m.ClassAttendanceSessionParticipantSchoolStudentID = r.ClassAttendanceSessionParticipantSchoolStudentID
	}
	if r.ClassAttendanceSessionParticipantSchoolTeacherID != nil {
		m.ClassAttendanceSessionParticipantSchoolTeacherID = r.ClassAttendanceSessionParticipantSchoolTeacherID
	}
	if r.ClassAttendanceSessionParticipantTeacherRole != nil && strings.TrimSpace(*r.ClassAttendanceSessionParticipantTeacherRole) != "" {
		role := attendanceModel.TeacherRole(strings.ToLower(strings.TrimSpace(*r.ClassAttendanceSessionParticipantTeacherRole)))
		m.ClassAttendanceSessionParticipantTeacherRole = &role
	}

	// type
	if r.ClassAttendanceSessionParticipantTypeID != nil {
		m.ClassAttendanceSessionParticipantTypeID = r.ClassAttendanceSessionParticipantTypeID
	}
	if r.ClassAttendanceSessionParticipantMarkedByTeacherID != nil {
		m.ClassAttendanceSessionParticipantMarkedByTeacherID = r.ClassAttendanceSessionParticipantMarkedByTeacherID
	}
	if r.ClassAttendanceSessionParticipantMethod != nil && strings.TrimSpace(*r.ClassAttendanceSessionParticipantMethod) != "" {
		mv := strings.ToLower(strings.TrimSpace(*r.ClassAttendanceSessionParticipantMethod))
		m.ClassAttendanceSessionParticipantMethod = &mv
	}

	return m
}

/* ===================== Patch DTO (tri-state) ===================== */

type ClassAttendanceSessionParticipantPatchRequest struct {
	ClassAttendanceSessionParticipantID uuid.UUID `json:"class_attendance_session_participant_id" validate:"required,uuid4"`

	// basic
	ClassAttendanceSessionParticipantState    PatchFieldUserAttendance[string]    `json:"class_attendance_session_participant_state,omitempty"`
	ClassAttendanceSessionParticipantTypeID   PatchFieldUserAttendance[uuid.UUID] `json:"class_attendance_session_participant_type_id,omitempty"`
	ClassAttendanceSessionParticipantDesc     PatchFieldUserAttendance[string]    `json:"class_attendance_session_participant_desc,omitempty"`
	ClassAttendanceSessionParticipantScore    PatchFieldUserAttendance[float64]   `json:"class_attendance_session_participant_score,omitempty"`
	ClassAttendanceSessionParticipantIsPassed PatchFieldUserAttendance[bool]      `json:"class_attendance_session_participant_is_passed,omitempty"`

	ClassAttendanceSessionParticipantCheckinAt  PatchFieldUserAttendance[time.Time] `json:"class_attendance_session_participant_checkin_at,omitempty"`
	ClassAttendanceSessionParticipantCheckoutAt PatchFieldUserAttendance[time.Time] `json:"class_attendance_session_participant_checkout_at,omitempty"`

	ClassAttendanceSessionParticipantMarkedAt          PatchFieldUserAttendance[time.Time] `json:"class_attendance_session_participant_marked_at,omitempty"`
	ClassAttendanceSessionParticipantMarkedByTeacherID PatchFieldUserAttendance[uuid.UUID] `json:"class_attendance_session_participant_marked_by_teacher_id,omitempty"`

	ClassAttendanceSessionParticipantMethod PatchFieldUserAttendance[string] `json:"class_attendance_session_participant_method,omitempty"`

	ClassAttendanceSessionParticipantLat         PatchFieldUserAttendance[float64] `json:"class_attendance_session_participant_lat,omitempty"`
	ClassAttendanceSessionParticipantLng         PatchFieldUserAttendance[float64] `json:"class_attendance_session_participant_lng,omitempty"`
	ClassAttendanceSessionParticipantDistanceM   PatchFieldUserAttendance[int]     `json:"class_attendance_session_participant_distance_m,omitempty"`
	ClassAttendanceSessionParticipantLateSeconds PatchFieldUserAttendance[int]     `json:"class_attendance_session_participant_late_seconds,omitempty"`

	// snapshot users_profile
	ClassAttendanceSessionParticipantUserProfileNameSnapshot              PatchFieldUserAttendance[string] `json:"class_attendance_session_participant_user_profile_name_snapshot,omitempty"`
	ClassAttendanceSessionParticipantUserProfileAvatarURLSnapshot         PatchFieldUserAttendance[string] `json:"class_attendance_session_participant_user_profile_avatar_url_snapshot,omitempty"`
	ClassAttendanceSessionParticipantUserProfileWhatsappURLSnapshot       PatchFieldUserAttendance[string] `json:"class_attendance_session_participant_user_profile_whatsapp_url_snapshot,omitempty"`
	ClassAttendanceSessionParticipantUserProfileParentNameSnapshot        PatchFieldUserAttendance[string] `json:"class_attendance_session_participant_user_profile_parent_name_snapshot,omitempty"`
	ClassAttendanceSessionParticipantUserProfileParentWhatsappURLSnapshot PatchFieldUserAttendance[string] `json:"class_attendance_session_participant_user_profile_parent_whatsapp_url_snapshot,omitempty"`
	ClassAttendanceSessionParticipantUserProfileGenderSnapshot            PatchFieldUserAttendance[string] `json:"class_attendance_session_participant_user_profile_gender_snapshot,omitempty"`

	ClassAttendanceSessionParticipantUserNote    PatchFieldUserAttendance[string]    `json:"class_attendance_session_participant_user_note,omitempty"`
	ClassAttendanceSessionParticipantTeacherNote PatchFieldUserAttendance[string]    `json:"class_attendance_session_participant_teacher_note,omitempty"`
	ClassAttendanceSessionParticipantLockedAt    PatchFieldUserAttendance[time.Time] `json:"class_attendance_session_participant_locked_at,omitempty"`

	URLs []ClassAttendanceSessionParticipantURLOpDTO `json:"urls,omitempty" validate:"omitempty,dive"`
}

func (p ClassAttendanceSessionParticipantPatchRequest) ApplyPatch(m *attendanceModel.ClassAttendanceSessionParticipantModel) error {
	if v, ok := p.ClassAttendanceSessionParticipantState.Get(); ok {
		if v == nil || strings.TrimSpace(*v) == "" {
			m.ClassAttendanceSessionParticipantState = attendanceModel.AttendanceStatePresent
		} else {
			m.ClassAttendanceSessionParticipantState = attendanceModel.AttendanceState(
				strings.ToLower(strings.TrimSpace(*v)),
			)
		}
	}
	if v, ok := p.ClassAttendanceSessionParticipantTypeID.Get(); ok {
		m.ClassAttendanceSessionParticipantTypeID = v
	}
	if v, ok := p.ClassAttendanceSessionParticipantDesc.Get(); ok {
		m.ClassAttendanceSessionParticipantDesc = v
	}
	if v, ok := p.ClassAttendanceSessionParticipantScore.Get(); ok {
		m.ClassAttendanceSessionParticipantScore = v
	}
	if v, ok := p.ClassAttendanceSessionParticipantIsPassed.Get(); ok {
		m.ClassAttendanceSessionParticipantIsPassed = v
	}

	// waktu: paksa UTC & support null
	if v, ok := p.ClassAttendanceSessionParticipantCheckinAt.Get(); ok {
		if v == nil {
			m.ClassAttendanceSessionParticipantCheckinAt = nil
		} else {
			t := v.UTC()
			m.ClassAttendanceSessionParticipantCheckinAt = &t
		}
	}

	if v, ok := p.ClassAttendanceSessionParticipantCheckoutAt.Get(); ok {
		if v == nil {
			m.ClassAttendanceSessionParticipantCheckoutAt = nil
		} else {
			t := v.UTC()
			m.ClassAttendanceSessionParticipantCheckoutAt = &t
		}
	}

	if v, ok := p.ClassAttendanceSessionParticipantMarkedAt.Get(); ok {
		if v == nil {
			m.ClassAttendanceSessionParticipantMarkedAt = nil
		} else {
			t := v.UTC()
			m.ClassAttendanceSessionParticipantMarkedAt = &t
		}
	}

	if v, ok := p.ClassAttendanceSessionParticipantMarkedByTeacherID.Get(); ok {
		m.ClassAttendanceSessionParticipantMarkedByTeacherID = v
	}

	if v, ok := p.ClassAttendanceSessionParticipantMethod.Get(); ok {
		if v == nil || strings.TrimSpace(*v) == "" {
			m.ClassAttendanceSessionParticipantMethod = nil
		} else {
			mv := strings.ToLower(strings.TrimSpace(*v))
			m.ClassAttendanceSessionParticipantMethod = &mv
		}
	}

	if v, ok := p.ClassAttendanceSessionParticipantLat.Get(); ok {
		m.ClassAttendanceSessionParticipantLat = v
	}
	if v, ok := p.ClassAttendanceSessionParticipantLng.Get(); ok {
		m.ClassAttendanceSessionParticipantLng = v
	}
	if v, ok := p.ClassAttendanceSessionParticipantDistanceM.Get(); ok {
		m.ClassAttendanceSessionParticipantDistanceM = v
	}
	if v, ok := p.ClassAttendanceSessionParticipantLateSeconds.Get(); ok {
		m.ClassAttendanceSessionParticipantLateSeconds = v
	}

	// snapshot users_profile
	if v, ok := p.ClassAttendanceSessionParticipantUserProfileNameSnapshot.Get(); ok {
		m.ClassAttendanceSessionParticipantUserProfileNameSnapshot = v
	}
	if v, ok := p.ClassAttendanceSessionParticipantUserProfileAvatarURLSnapshot.Get(); ok {
		m.ClassAttendanceSessionParticipantUserProfileAvatarURLSnapshot = v
	}
	if v, ok := p.ClassAttendanceSessionParticipantUserProfileWhatsappURLSnapshot.Get(); ok {
		m.ClassAttendanceSessionParticipantUserProfileWhatsappURLSnapshot = v
	}
	if v, ok := p.ClassAttendanceSessionParticipantUserProfileParentNameSnapshot.Get(); ok {
		m.ClassAttendanceSessionParticipantUserProfileParentNameSnapshot = v
	}
	if v, ok := p.ClassAttendanceSessionParticipantUserProfileParentWhatsappURLSnapshot.Get(); ok {
		m.ClassAttendanceSessionParticipantUserProfileParentWhatsappURLSnapshot = v
	}
	if v, ok := p.ClassAttendanceSessionParticipantUserProfileGenderSnapshot.Get(); ok {
		m.ClassAttendanceSessionParticipantUserProfileGenderSnapshot = v
	}

	if v, ok := p.ClassAttendanceSessionParticipantUserNote.Get(); ok {
		m.ClassAttendanceSessionParticipantUserNote = v
	}
	if v, ok := p.ClassAttendanceSessionParticipantTeacherNote.Get(); ok {
		m.ClassAttendanceSessionParticipantTeacherNote = v
	}
	if v, ok := p.ClassAttendanceSessionParticipantLockedAt.Get(); ok {
		if v == nil {
			m.ClassAttendanceSessionParticipantLockedAt = nil
		} else {
			t := v.UTC()
			m.ClassAttendanceSessionParticipantLockedAt = &t
		}
	}
	return nil
}

/* ===================== Query DTO (for List) ===================== */

type ListClassAttendanceSessionParticipantQuery struct {
	Search CSV `query:"search"`

	StateIn  CSV `query:"state_in"`  // present|absent|late|excused|sick|leave|unmarked
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

/* ===================== TIME HELPERS (School Time) ===================== */
/*
   Dipanggil dari controller:
   - DB tetap simpan UTC
   - Response ke client pakai timezone sekolah (berdasar school context di fiber.Ctx)
*/

func NormalizeParticipantTimesToSchoolTime(
	c *fiber.Ctx,
	m *attendanceModel.ClassAttendanceSessionParticipantModel,
) {
	if m == nil {
		return
	}

	if m.ClassAttendanceSessionParticipantCheckinAt != nil {
		m.ClassAttendanceSessionParticipantCheckinAt = dbtime.ToSchoolTimePtr(c, m.ClassAttendanceSessionParticipantCheckinAt)
	}
	if m.ClassAttendanceSessionParticipantCheckoutAt != nil {
		m.ClassAttendanceSessionParticipantCheckoutAt = dbtime.ToSchoolTimePtr(c, m.ClassAttendanceSessionParticipantCheckoutAt)
	}
	if m.ClassAttendanceSessionParticipantMarkedAt != nil {
		m.ClassAttendanceSessionParticipantMarkedAt = dbtime.ToSchoolTimePtr(c, m.ClassAttendanceSessionParticipantMarkedAt)
	}
	if m.ClassAttendanceSessionParticipantLockedAt != nil {
		m.ClassAttendanceSessionParticipantLockedAt = dbtime.ToSchoolTimePtr(c, m.ClassAttendanceSessionParticipantLockedAt)
	}
}

// Versi slice: enak buat List
func NormalizeParticipantsSliceToSchoolTime(
	c *fiber.Ctx,
	list []attendanceModel.ClassAttendanceSessionParticipantModel,
) []attendanceModel.ClassAttendanceSessionParticipantModel {
	for i := range list {
		NormalizeParticipantTimesToSchoolTime(c, &list[i])
	}
	return list
}

// ===================== COMPACT RESPONSE =====================

type ClassAttendanceSessionParticipantCompactResponse struct {
	ClassAttendanceSessionParticipantID        uuid.UUID `json:"class_attendance_session_participant_id"`
	ClassAttendanceSessionParticipantSessionID uuid.UUID `json:"class_attendance_session_participant_session_id"`

	// relasi
	ClassAttendanceSessionParticipantSchoolStudentID *uuid.UUID `json:"class_attendance_session_participant_school_student_id,omitempty"`
	ClassAttendanceSessionParticipantSchoolTeacherID *uuid.UUID `json:"class_attendance_session_participant_school_teacher_id,omitempty"`
	ClassAttendanceSessionParticipantKind            string     `json:"class_attendance_session_participant_kind"`
	ClassAttendanceSessionParticipantTeacherRole     *string    `json:"class_attendance_session_participant_teacher_role,omitempty"`

	// state & type
	ClassAttendanceSessionParticipantState  string     `json:"class_attendance_session_participant_state"`
	ClassAttendanceSessionParticipantTypeID *uuid.UUID `json:"class_attendance_session_participant_type_id,omitempty"`

	// nilai & deskripsi
	ClassAttendanceSessionParticipantDesc     *string  `json:"class_attendance_session_participant_desc,omitempty"`
	ClassAttendanceSessionParticipantScore    *float64 `json:"class_attendance_session_participant_score,omitempty"`
	ClassAttendanceSessionParticipantIsPassed *bool    `json:"class_attendance_session_participant_is_passed,omitempty"`

	// waktu (sudah di-normalize ke school time via WithSchoolTime)
	ClassAttendanceSessionParticipantCheckinAt  *time.Time `json:"class_attendance_session_participant_checkin_at,omitempty"`
	ClassAttendanceSessionParticipantCheckoutAt *time.Time `json:"class_attendance_session_participant_checkout_at,omitempty"`
	ClassAttendanceSessionParticipantMarkedAt   *time.Time `json:"class_attendance_session_participant_marked_at,omitempty"`
	ClassAttendanceSessionParticipantLockedAt   *time.Time `json:"class_attendance_session_participant_locked_at,omitempty"`

	// metode & geo
	ClassAttendanceSessionParticipantMethod      *string  `json:"class_attendance_session_participant_method,omitempty"`

	// snapshot user profile (tetap ikut, karena ringan & kepake di UI)
	ClassAttendanceSessionParticipantUserProfileNameSnapshot              *string `json:"class_attendance_session_participant_user_profile_name_snapshot,omitempty"`
	ClassAttendanceSessionParticipantUserProfileAvatarURLSnapshot         *string `json:"class_attendance_session_participant_user_profile_avatar_url_snapshot,omitempty"`
	ClassAttendanceSessionParticipantUserProfileWhatsappURLSnapshot       *string `json:"class_attendance_session_participant_user_profile_whatsapp_url_snapshot,omitempty"`
	ClassAttendanceSessionParticipantUserProfileParentNameSnapshot        *string `json:"class_attendance_session_participant_user_profile_parent_name_snapshot,omitempty"`
	ClassAttendanceSessionParticipantUserProfileParentWhatsappURLSnapshot *string `json:"class_attendance_session_participant_user_profile_parent_whatsapp_url_snapshot,omitempty"`
	ClassAttendanceSessionParticipantUserProfileGenderSnapshot            *string `json:"class_attendance_session_participant_user_profile_gender_snapshot,omitempty"`

	// audit
	ClassAttendanceSessionParticipantCreatedAt time.Time `json:"class_attendance_session_participant_created_at"`
	ClassAttendanceSessionParticipantUpdatedAt time.Time `json:"class_attendance_session_participant_updated_at"`
}

// Converter dari Model → Compact DTO
func NewClassAttendanceSessionParticipantCompactResponse(
	m attendanceModel.ClassAttendanceSessionParticipantModel,
) ClassAttendanceSessionParticipantCompactResponse {
	return ClassAttendanceSessionParticipantCompactResponse{
		ClassAttendanceSessionParticipantID:              m.ClassAttendanceSessionParticipantID,
		ClassAttendanceSessionParticipantSessionID:       m.ClassAttendanceSessionParticipantSessionID,
		ClassAttendanceSessionParticipantSchoolStudentID: m.ClassAttendanceSessionParticipantSchoolStudentID,
		ClassAttendanceSessionParticipantSchoolTeacherID: m.ClassAttendanceSessionParticipantSchoolTeacherID,
		ClassAttendanceSessionParticipantKind:            string(m.ClassAttendanceSessionParticipantKind),
		ClassAttendanceSessionParticipantTeacherRole:     toStrPtrFromTeacherRole(m.ClassAttendanceSessionParticipantTeacherRole),

		ClassAttendanceSessionParticipantState:  string(m.ClassAttendanceSessionParticipantState),
		ClassAttendanceSessionParticipantTypeID: m.ClassAttendanceSessionParticipantTypeID,

		ClassAttendanceSessionParticipantDesc:     m.ClassAttendanceSessionParticipantDesc,
		ClassAttendanceSessionParticipantScore:    m.ClassAttendanceSessionParticipantScore,
		ClassAttendanceSessionParticipantIsPassed: m.ClassAttendanceSessionParticipantIsPassed,

		ClassAttendanceSessionParticipantCheckinAt:  m.ClassAttendanceSessionParticipantCheckinAt,
		ClassAttendanceSessionParticipantCheckoutAt: m.ClassAttendanceSessionParticipantCheckoutAt,
		ClassAttendanceSessionParticipantMarkedAt:   m.ClassAttendanceSessionParticipantMarkedAt,
		ClassAttendanceSessionParticipantLockedAt:   m.ClassAttendanceSessionParticipantLockedAt,

		ClassAttendanceSessionParticipantMethod:      m.ClassAttendanceSessionParticipantMethod,


		ClassAttendanceSessionParticipantUserProfileNameSnapshot:              m.ClassAttendanceSessionParticipantUserProfileNameSnapshot,
		ClassAttendanceSessionParticipantUserProfileAvatarURLSnapshot:         m.ClassAttendanceSessionParticipantUserProfileAvatarURLSnapshot,
		ClassAttendanceSessionParticipantUserProfileWhatsappURLSnapshot:       m.ClassAttendanceSessionParticipantUserProfileWhatsappURLSnapshot,
		ClassAttendanceSessionParticipantUserProfileParentNameSnapshot:        m.ClassAttendanceSessionParticipantUserProfileParentNameSnapshot,
		ClassAttendanceSessionParticipantUserProfileParentWhatsappURLSnapshot: m.ClassAttendanceSessionParticipantUserProfileParentWhatsappURLSnapshot,
		ClassAttendanceSessionParticipantUserProfileGenderSnapshot:            m.ClassAttendanceSessionParticipantUserProfileGenderSnapshot,

		ClassAttendanceSessionParticipantCreatedAt: m.ClassAttendanceSessionParticipantCreatedAt,
		ClassAttendanceSessionParticipantUpdatedAt: m.ClassAttendanceSessionParticipantUpdatedAt,
	}
}

// helper kecil buat convert *TeacherRole → *string
func toStrPtrFromTeacherRole(r *attendanceModel.TeacherRole) *string {
	if r == nil {
		return nil
	}
	s := string(*r)
	return &s
}

// WithSchoolTime: mirip pattern di session compact
func (r ClassAttendanceSessionParticipantCompactResponse) WithSchoolTime(
	c *fiber.Ctx,
) ClassAttendanceSessionParticipantCompactResponse {
	if r.ClassAttendanceSessionParticipantCheckinAt != nil {
		r.ClassAttendanceSessionParticipantCheckinAt =
			dbtime.ToSchoolTimePtr(c, r.ClassAttendanceSessionParticipantCheckinAt)
	}
	if r.ClassAttendanceSessionParticipantCheckoutAt != nil {
		r.ClassAttendanceSessionParticipantCheckoutAt =
			dbtime.ToSchoolTimePtr(c, r.ClassAttendanceSessionParticipantCheckoutAt)
	}
	if r.ClassAttendanceSessionParticipantMarkedAt != nil {
		r.ClassAttendanceSessionParticipantMarkedAt =
			dbtime.ToSchoolTimePtr(c, r.ClassAttendanceSessionParticipantMarkedAt)
	}
	if r.ClassAttendanceSessionParticipantLockedAt != nil {
		r.ClassAttendanceSessionParticipantLockedAt =
			dbtime.ToSchoolTimePtr(c, r.ClassAttendanceSessionParticipantLockedAt)
	}
	return r
}

// Versi slice, enak buat List di controller
func MapParticipantsToCompact(
	c *fiber.Ctx,
	list []attendanceModel.ClassAttendanceSessionParticipantModel,
) []ClassAttendanceSessionParticipantCompactResponse {
	out := make([]ClassAttendanceSessionParticipantCompactResponse, 0, len(list))
	for _, m := range list {
		item := NewClassAttendanceSessionParticipantCompactResponse(m).WithSchoolTime(c)
		out = append(out, item)
	}
	return out
}
