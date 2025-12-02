package dto

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	model "madinahsalam_backend/internals/features/school/classes/class_materials/model"
)

/* =======================================================
   QUERY DTO (untuk list progress)
   - Dipakai di controller: c.QueryParser(&q)
======================================================= */

type StudentClassMaterialProgressListQuery struct {
	// filter by SCSST (kelas-mapel guru x murid)
	StudentClassMaterialProgressSCSSTID *uuid.UUID `query:"scsst_id"`

	// filter by class_material (lihat siapa saja yang sudah / belum selesai materi ini)
	StudentClassMaterialProgressClassMaterialID *uuid.UUID `query:"class_material_id"`

	// filter by status (not_started / in_progress / completed / skipped)
	StudentClassMaterialProgressStatus *string `query:"status"`
}

/* =======================================================
   RESPONSE DTO
   Dipakai untuk list & detail progress
   - Nama json 1:1 dengan nama kolom
======================================================= */

type StudentClassMaterialProgressResponse struct {
	StudentClassMaterialProgressID uuid.UUID `json:"student_class_material_progress_id"`

	StudentClassMaterialProgressSchoolID        uuid.UUID `json:"student_class_material_progress_school_id"`
	StudentClassMaterialProgressStudentID       uuid.UUID `json:"student_class_material_progress_student_id"`
	StudentClassMaterialProgressSCSSTID         uuid.UUID `json:"student_class_material_progress_scsst_id"`
	StudentClassMaterialProgressClassMaterialID uuid.UUID `json:"student_class_material_progress_class_material_id"`

	StudentClassMaterialProgressStatus          string         `json:"student_class_material_progress_status"`
	StudentClassMaterialProgressLastPercent     int32          `json:"student_class_material_progress_last_percent"`
	StudentClassMaterialProgressIsCompleted     bool           `json:"student_class_material_progress_is_completed"`
	StudentClassMaterialProgressViewDurationSec *int64         `json:"student_class_material_progress_view_duration_sec"`
	StudentClassMaterialProgressOpenCount       int32          `json:"student_class_material_progress_open_count"`
	StudentClassMaterialProgressFirstStartedAt  *time.Time     `json:"student_class_material_progress_first_started_at"`
	StudentClassMaterialProgressLastActivityAt  *time.Time     `json:"student_class_material_progress_last_activity_at"`
	StudentClassMaterialProgressCompletedAt     *time.Time     `json:"student_class_material_progress_completed_at"`
	StudentClassMaterialProgressExtra           datatypes.JSON `json:"student_class_material_progress_extra"`
	StudentClassMaterialProgressCreatedAt       time.Time      `json:"student_class_material_progress_created_at"`
	StudentClassMaterialProgressUpdatedAt       time.Time      `json:"student_class_material_progress_updated_at"`
}

func NewStudentClassMaterialProgressResponse(m *model.StudentClassMaterialProgressModel) *StudentClassMaterialProgressResponse {
	if m == nil {
		return nil
	}

	return &StudentClassMaterialProgressResponse{
		StudentClassMaterialProgressID:              m.StudentClassMaterialProgressID,
		StudentClassMaterialProgressSchoolID:        m.StudentClassMaterialProgressSchoolID,
		StudentClassMaterialProgressStudentID:       m.StudentClassMaterialProgressStudentID,
		StudentClassMaterialProgressSCSSTID:         m.StudentClassMaterialProgressSCSSTID,
		StudentClassMaterialProgressClassMaterialID: m.StudentClassMaterialProgressClassMaterialID,
		StudentClassMaterialProgressStatus:          string(m.StudentClassMaterialProgressStatus),
		StudentClassMaterialProgressLastPercent:     m.StudentClassMaterialProgressLastPercent,
		StudentClassMaterialProgressIsCompleted:     m.StudentClassMaterialProgressIsCompleted,
		StudentClassMaterialProgressViewDurationSec: m.StudentClassMaterialProgressViewDuration,
		StudentClassMaterialProgressOpenCount:       m.StudentClassMaterialProgressOpenCount,
		StudentClassMaterialProgressFirstStartedAt:  m.StudentClassMaterialProgressFirstStartedAt,
		StudentClassMaterialProgressLastActivityAt:  m.StudentClassMaterialProgressLastActivityAt,
		StudentClassMaterialProgressCompletedAt:     m.StudentClassMaterialProgressCompletedAt,
		StudentClassMaterialProgressExtra:           m.StudentClassMaterialProgressExtra,
		StudentClassMaterialProgressCreatedAt:       m.StudentClassMaterialProgressCreatedAt,
		StudentClassMaterialProgressUpdatedAt:       m.StudentClassMaterialProgressUpdatedAt,
	}
}

func NewStudentClassMaterialProgressResponseList(list []*model.StudentClassMaterialProgressModel) []*StudentClassMaterialProgressResponse {
	out := make([]*StudentClassMaterialProgressResponse, 0, len(list))
	for _, m := range list {
		out = append(out, NewStudentClassMaterialProgressResponse(m))
	}
	return out
}

/* =======================================================
   REQUEST DTO: Ping / Upsert Progress
   - Dipakai frontend untuk update progress article/video/pdf/dll
   - Nama json tetap verbose mengikuti kolom, kecuali _delta
======================================================= */

type StudentClassMaterialProgressPingRequest struct {
	// Identitas materi & SCSST (WAJIB)
	StudentClassMaterialProgressSCSSTID         uuid.UUID `json:"student_class_material_progress_scsst_id"`
	StudentClassMaterialProgressClassMaterialID uuid.UUID `json:"student_class_material_progress_class_material_id"`

	// Status (opsional)
	StudentClassMaterialProgressStatus *string `json:"student_class_material_progress_status"` // not_started / in_progress / completed / skipped

	// Percent 0..100 (opsional)
	StudentClassMaterialProgressLastPercent *int32 `json:"student_class_material_progress_last_percent"`

	// Tambahan detik durasi view dari ping ini (bukan total)
	StudentClassMaterialProgressViewDeltaSec *int64 `json:"student_class_material_progress_view_delta_sec"`

	// Optional: jika true → mark completed paksa
	StudentClassMaterialProgressMarkCompleted *bool `json:"student_class_material_progress_mark_completed"`

	// Extra per-type (scroll_pos, last_second, last_page, dsb)
	StudentClassMaterialProgressExtra any `json:"student_class_material_progress_extra"`
}

/* =======================================================
   Helpers: enum parsing
======================================================= */

func parseMaterialProgressStatus(s string) model.MaterialProgressStatus {
	switch model.MaterialProgressStatus(s) {
	case model.MaterialProgressStatusNotStarted,
		model.MaterialProgressStatusInProgress,
		model.MaterialProgressStatusCompleted,
		model.MaterialProgressStatusSkipped:
		return model.MaterialProgressStatus(s)
	default:
		return model.MaterialProgressStatusInProgress
	}
}

/* =======================================================
   Builder: New Model (INSERT pertama)
   - school_id & student_id diisi dari context/token
======================================================= */

func (req *StudentClassMaterialProgressPingRequest) ToNewModel(
	schoolID uuid.UUID,
	studentID uuid.UUID,
	now time.Time,
) *model.StudentClassMaterialProgressModel {
	if req == nil {
		return nil
	}

	m := &model.StudentClassMaterialProgressModel{
		StudentClassMaterialProgressSchoolID:        schoolID,
		StudentClassMaterialProgressStudentID:       studentID,
		StudentClassMaterialProgressSCSSTID:         req.StudentClassMaterialProgressSCSSTID,
		StudentClassMaterialProgressClassMaterialID: req.StudentClassMaterialProgressClassMaterialID,

		StudentClassMaterialProgressStatus:      model.MaterialProgressStatusInProgress,
		StudentClassMaterialProgressLastPercent: 0,
		StudentClassMaterialProgressIsCompleted: false,
		StudentClassMaterialProgressOpenCount:   0,
	}

	// apply field-field dinamis dari request
	req.ApplyToModel(m, now)

	return m
}

/* =======================================================
   Apply Ping ke Model (UPDATE)
   - Dipakai di service untuk upsert
======================================================= */

func (req *StudentClassMaterialProgressPingRequest) ApplyToModel(
	m *model.StudentClassMaterialProgressModel,
	now time.Time,
) {
	if req == nil || m == nil {
		return
	}

	// Pastikan SCSST & ClassMaterial tetap sinkron
	m.StudentClassMaterialProgressSCSSTID = req.StudentClassMaterialProgressSCSSTID
	m.StudentClassMaterialProgressClassMaterialID = req.StudentClassMaterialProgressClassMaterialID

	// first_started_at
	if m.StudentClassMaterialProgressFirstStartedAt == nil {
		m.StudentClassMaterialProgressFirstStartedAt = &now
	}
	// last_activity_at selalu di-update
	m.StudentClassMaterialProgressLastActivityAt = &now

	// status (jika dikirim)
	if req.StudentClassMaterialProgressStatus != nil && *req.StudentClassMaterialProgressStatus != "" {
		m.StudentClassMaterialProgressStatus = parseMaterialProgressStatus(*req.StudentClassMaterialProgressStatus)
	}

	// last_percent
	if req.StudentClassMaterialProgressLastPercent != nil {
		p := *req.StudentClassMaterialProgressLastPercent
		if p < 0 {
			p = 0
		}
		if p > 100 {
			p = 100
		}
		m.StudentClassMaterialProgressLastPercent = p
	}

	// view_duration (delta)
	if req.StudentClassMaterialProgressViewDeltaSec != nil && *req.StudentClassMaterialProgressViewDeltaSec > 0 {
		if m.StudentClassMaterialProgressViewDuration == nil {
			m.StudentClassMaterialProgressViewDuration = new(int64)
		}
		*m.StudentClassMaterialProgressViewDuration += *req.StudentClassMaterialProgressViewDeltaSec
	}

	// open_count: simple heuristic, setiap ping kita anggap 1 aktifitas
	m.StudentClassMaterialProgressOpenCount = m.StudentClassMaterialProgressOpenCount + 1

	// extra (jsonb)
	if req.StudentClassMaterialProgressExtra != nil {
		if b, err := json.Marshal(req.StudentClassMaterialProgressExtra); err == nil {
			m.StudentClassMaterialProgressExtra = datatypes.JSON(b)
		}
	}

	// completed logic
	maybeMarkCompleted := false

	if req.StudentClassMaterialProgressMarkCompleted != nil && *req.StudentClassMaterialProgressMarkCompleted {
		maybeMarkCompleted = true
	}

	if req.StudentClassMaterialProgressLastPercent != nil && *req.StudentClassMaterialProgressLastPercent >= 100 {
		maybeMarkCompleted = true
	}

	if maybeMarkCompleted {
		m.StudentClassMaterialProgressIsCompleted = true
		m.StudentClassMaterialProgressStatus = model.MaterialProgressStatusCompleted
		if m.StudentClassMaterialProgressCompletedAt == nil {
			m.StudentClassMaterialProgressCompletedAt = &now
		}
	}
}
