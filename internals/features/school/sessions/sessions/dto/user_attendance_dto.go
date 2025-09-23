// file: internals/features/school/sessions/sessions/dto/user_attendance_dto.go
package dto

import (
	"bytes"
	"encoding"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	model "masjidku_backend/internals/features/school/sessions/sessions/model"
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

type UserAttendanceURLOpDTO struct {
	Op URLOp `json:"op" validate:"required,oneof=upsert delete"`

	ID   *uuid.UUID `json:"id,omitempty" validate:"omitempty,uuid4"`
	Kind *string    `json:"kind,omitempty" validate:"omitempty,oneof=image video attachment link audio"`

	Label              *string    `json:"label,omitempty" validate:"omitempty,max=160"`
	Order              *int       `json:"order,omitempty" validate:"omitempty,min=0"`
	IsPrimary          *bool      `json:"is_primary,omitempty"`
	Href               *string    `json:"href,omitempty" validate:"omitempty,max=2048"`
	ObjectKey          *string    `json:"object_key,omitempty"`
	ObjectKeyOld       *string    `json:"object_key_old,omitempty"`
	TrashURL           *string    `json:"trash_url,omitempty"`
	DeletePendingUntil *time.Time `json:"delete_pending_until,omitempty"`

	UploaderTeacherID *uuid.UUID `json:"uploader_teacher_id,omitempty" validate:"omitempty,uuid4"`
	UploaderStudentID *uuid.UUID `json:"uploader_student_id,omitempty" validate:"omitempty,uuid4"`
}

/* ===================== Create DTO ===================== */

type UserAttendanceCreateRequest struct {
	MasjidID        uuid.UUID `json:"masjid_id"         validate:"required,uuid4"`
	SessionID       uuid.UUID `json:"session_id"        validate:"required,uuid4"`
	MasjidStudentID uuid.UUID `json:"masjid_student_id" validate:"required,uuid4"`

	Status      *model.UserAttendanceStatus `json:"status,omitempty" validate:"omitempty,oneof=unmarked present absent excused late"`
	TypeID      *uuid.UUID                  `json:"type_id,omitempty" validate:"omitempty,uuid4"`
	Desc        *string                     `json:"desc,omitempty"`
	Score       *float64                    `json:"score,omitempty" validate:"omitempty,gte=0,lte=100"`
	IsPassed    *bool                       `json:"is_passed,omitempty"`
	MarkedAt    *time.Time                  `json:"marked_at,omitempty"`
	MarkedByTID *uuid.UUID                  `json:"marked_by_teacher_id,omitempty" validate:"omitempty,uuid4"`
	Method      *model.UserAttendanceMethod `json:"method,omitempty" validate:"omitempty,oneof=manual qr geo import api self"`

	Lat        *float64 `json:"lat,omitempty"`
	Lng        *float64 `json:"lng,omitempty"`
	DistanceM  *int     `json:"distance_m,omitempty" validate:"omitempty,min=0"`
	LateSecond *int     `json:"late_seconds,omitempty" validate:"omitempty,min=0"`

	UserNote    *string    `json:"user_note,omitempty"`
	TeacherNote *string    `json:"teacher_note,omitempty"`
	LockedAt    *time.Time `json:"locked_at,omitempty"`

	URLs []UserAttendanceURLOpDTO `json:"urls,omitempty" validate:"omitempty,dive"`
}

/* ===================== Patch DTO (tri-state) ===================== */

type UserAttendancePatchRequest struct {
	AttendanceID uuid.UUID `json:"attendance_id" validate:"required,uuid4"`

	Status      PatchFieldUserAttendance[model.UserAttendanceStatus] `json:"status,omitempty"`
	TypeID      PatchFieldUserAttendance[uuid.UUID]                  `json:"type_id,omitempty"`
	Desc        PatchFieldUserAttendance[string]                     `json:"desc,omitempty"`
	Score       PatchFieldUserAttendance[float64]                    `json:"score,omitempty"`
	IsPassed    PatchFieldUserAttendance[bool]                       `json:"is_passed,omitempty"`
	MarkedAt    PatchFieldUserAttendance[time.Time]                  `json:"marked_at,omitempty"`
	MarkedByTID PatchFieldUserAttendance[uuid.UUID]                  `json:"marked_by_teacher_id,omitempty"`
	Method      PatchFieldUserAttendance[model.UserAttendanceMethod] `json:"method,omitempty"`

	Lat         PatchFieldUserAttendance[float64] `json:"lat,omitempty"`
	Lng         PatchFieldUserAttendance[float64] `json:"lng,omitempty"`
	DistanceM   PatchFieldUserAttendance[int]     `json:"distance_m,omitempty"`
	LateSeconds PatchFieldUserAttendance[int]     `json:"late_seconds,omitempty"`

	UserNote    PatchFieldUserAttendance[string]    `json:"user_note,omitempty"`
	TeacherNote PatchFieldUserAttendance[string]    `json:"teacher_note,omitempty"`
	LockedAt    PatchFieldUserAttendance[time.Time] `json:"locked_at,omitempty"`

	URLs []UserAttendanceURLOpDTO `json:"urls,omitempty" validate:"omitempty,dive"`
}

/* ===================== Query DTO (for List) ===================== */

type ListUserAttendanceQuery struct {
	Search   string `query:"search"`
	StatusIn CSV    `query:"status_in"` // unmarked|present|absent|excused|late
	MethodIn CSV    `query:"method_in"` // manual|qr|geo|import|api|self

	SessionID         string `query:"session_id"`
	MasjidStudentID   string `query:"masjid_student_id"`
	TypeID            string `query:"type_id"`
	MarkedByTeacherID string `query:"marked_by_teacher_id"`

	CreatedGE string `query:"created_ge"`
	CreatedLE string `query:"created_le"`
	MarkedGE  string `query:"marked_ge"`
	MarkedLE  string `query:"marked_le"`
}

/* ===================== Mappers & URL mutation builder ===================== */

func (r UserAttendanceCreateRequest) ToModel() model.UserAttendanceModel {
	m := model.UserAttendanceModel{
		UserAttendanceMasjidID:        r.MasjidID,
		UserAttendanceSessionID:       r.SessionID,
		UserAttendanceMasjidStudentID: r.MasjidStudentID,

		UserAttendanceDesc:        r.Desc,
		UserAttendanceScore:       r.Score,
		UserAttendanceIsPassed:    r.IsPassed,
		UserAttendanceMarkedAt:    r.MarkedAt,
		UserAttendanceMethod:      r.Method,
		UserAttendanceLat:         r.Lat,
		UserAttendanceLng:         r.Lng,
		UserAttendanceDistanceM:   r.DistanceM,
		UserAttendanceLateSeconds: r.LateSecond,
		UserAttendanceUserNote:    r.UserNote,
		UserAttendanceTeacherNote: r.TeacherNote,
		UserAttendanceLockedAt:    r.LockedAt,
	}
	if r.Status != nil {
		m.UserAttendanceStatus = *r.Status
	}
	if r.TypeID != nil {
		m.UserAttendanceTypeID = r.TypeID
	}
	if r.MarkedByTID != nil {
		m.UserAttendanceMarkedByTeacherID = r.MarkedByTID
	}
	return m
}

func (p UserAttendancePatchRequest) ApplyPatch(m *model.UserAttendanceModel) error {
	if v, ok := p.Status.Get(); ok {
		if v == nil {
			def := model.UserAttendanceUnmarked
			m.UserAttendanceStatus = def
		} else {
			m.UserAttendanceStatus = *v
		}
	}
	if v, ok := p.TypeID.Get(); ok {
		m.UserAttendanceTypeID = v
	}
	if v, ok := p.Desc.Get(); ok {
		m.UserAttendanceDesc = v
	}
	if v, ok := p.Score.Get(); ok {
		m.UserAttendanceScore = v
	}
	if v, ok := p.IsPassed.Get(); ok {
		m.UserAttendanceIsPassed = v
	}
	if v, ok := p.MarkedAt.Get(); ok {
		m.UserAttendanceMarkedAt = v
	}
	if v, ok := p.MarkedByTID.Get(); ok {
		m.UserAttendanceMarkedByTeacherID = v
	}
	if v, ok := p.Method.Get(); ok {
		if v == nil {
			m.UserAttendanceMethod = nil
		} else {
			mv := model.UserAttendanceMethod(*v)
			m.UserAttendanceMethod = &mv
		}
	}
	if v, ok := p.Lat.Get(); ok {
		m.UserAttendanceLat = v
	}
	if v, ok := p.Lng.Get(); ok {
		m.UserAttendanceLng = v
	}
	if v, ok := p.DistanceM.Get(); ok {
		m.UserAttendanceDistanceM = v
	}
	if v, ok := p.LateSeconds.Get(); ok {
		m.UserAttendanceLateSeconds = v
	}
	if v, ok := p.UserNote.Get(); ok {
		m.UserAttendanceUserNote = v
	}
	if v, ok := p.TeacherNote.Get(); ok {
		m.UserAttendanceTeacherNote = v
	}
	if v, ok := p.LockedAt.Get(); ok {
		m.UserAttendanceLockedAt = v
	}
	return nil
}

type URLMutations struct {
	ToCreate []model.UserAttendanceURL
	ToUpdate []model.UserAttendanceURL
	ToDelete []uuid.UUID
}

func BuildURLMutations(attendanceID uuid.UUID, masjidID uuid.UUID, ops []UserAttendanceURLOpDTO) (URLMutations, error) {
	var out URLMutations
	for _, op := range ops {
		switch op.Op {
		case URLOpUpsert:
			if op.ID == nil {
				if op.Kind == nil {
					return out, errors.New("kind required for create")
				}
				kind, err := normalizeKind(*op.Kind)
				if err != nil {
					return out, err
				}
				row := model.UserAttendanceURL{
					UserAttendanceURLMasjidID:           masjidID,
					UserAttendanceURLAttendance:         attendanceID,
					UserAttendanceURLKind:               kind,
					UserAttendanceURLLabel:              op.Label,
					UserAttendanceURLOrder:              int32(pint(op.Order)),
					UserAttendanceURLIsPrimary:          pbool(op.IsPrimary),
					UserAttendanceURLHref:               op.Href,
					UserAttendanceURLObjectKey:          op.ObjectKey,
					UserAttendanceURLObjectKeyOld:       op.ObjectKeyOld,
					UserAttendanceURLTrashURL:           op.TrashURL,
					UserAttendanceURLDeletePendingUntil: op.DeletePendingUntil,
					UserAttendanceURLUploaderTeacherID:  op.UploaderTeacherID,
					UserAttendanceURLUploaderStudentID:  op.UploaderStudentID,
				}
				out.ToCreate = append(out.ToCreate, row)
			} else {
				kind := ""
				if op.Kind != nil {
					var err error
					kind, err = normalizeKind(*op.Kind)
					if err != nil {
						return out, err
					}
				}
				row := model.UserAttendanceURL{
					UserAttendanceURLID:                 *op.ID,
					UserAttendanceURLLabel:              op.Label,
					UserAttendanceURLOrder:              int32(pint(op.Order)),
					UserAttendanceURLIsPrimary:          pbool(op.IsPrimary),
					UserAttendanceURLHref:               op.Href,
					UserAttendanceURLObjectKey:          op.ObjectKey,
					UserAttendanceURLObjectKeyOld:       op.ObjectKeyOld,
					UserAttendanceURLTrashURL:           op.TrashURL,
					UserAttendanceURLDeletePendingUntil: op.DeletePendingUntil,
					UserAttendanceURLUploaderTeacherID:  op.UploaderTeacherID,
					UserAttendanceURLUploaderStudentID:  op.UploaderStudentID,
				}
				if op.Kind != nil {
					row.UserAttendanceURLKind = kind
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
