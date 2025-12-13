package service

import (
	"time"

	sessionModel "madinahsalam_backend/internals/features/school/class_others/class_attendance_sessions/model"
	assessmentModel "madinahsalam_backend/internals/features/school/submissions_assesments/assesments/model"
)

/* =========================================================
   Helper: deadline dari ClassAttendanceSession
========================================================= */

func deadlineFromSession(s *sessionModel.ClassAttendanceSessionModel) *time.Time {
	if s == nil {
		return nil
	}

	// 1) Prioritas: EndsAt kalau ada
	if s.ClassAttendanceSessionEndsAt != nil {
		t := s.ClassAttendanceSessionEndsAt.UTC()
		return &t
	}

	// 2) Kalau tidak ada EndsAt, pakai StartsAt kalau ada
	if s.ClassAttendanceSessionStartsAt != nil {
		t := s.ClassAttendanceSessionStartsAt.UTC()
		return &t
	}

	// 3) Fallback: pakai tanggal, end-of-day (23:59:59)
	d := s.ClassAttendanceSessionDate
	t := time.Date(d.Year(), d.Month(), d.Day(), 23, 59, 59, 0, time.UTC)
	return &t
}

/* =========================================================
   Core logic (dipakai kedua varian)
========================================================= */

func computeIsOpenCore(a *assessmentModel.AssessmentModel, now time.Time, effectiveDue *time.Time) bool {
	// 1) Harus published
	if a.AssessmentStatus != assessmentModel.AssessmentStatusPublished {
		return false
	}

	// 4) Batas bawah: start_at (kalau ada)
	if a.AssessmentStartAt != nil && now.Before(*a.AssessmentStartAt) {
		return false
	}

	// 5) Batas atas: effectiveDue (bisa dari due_at atau session)
	if effectiveDue != nil && now.After(*effectiveDue) {
		return false
	}

	// Lolos semua rule → open
	return true
}

/* =========================================================
   Varian 1: hanya pakai field AssessmentModel
   (pakai AssessmentDueAt biasa)
========================================================= */

// ComputeIsOpen menghitung apakah assessment "open" di waktu now (UTC)
// hanya berdasarkan field di AssessmentModel (status, allow, start_at, due_at, closed_at).
func ComputeIsOpen(a *assessmentModel.AssessmentModel, now time.Time) bool {
	return computeIsOpenCore(a, now, a.AssessmentDueAt)
}

/* =========================================================
   Varian 2: pakai CollectSession kalau ada
========================================================= */

// ComputeIsOpenWithCollectSession:
// - Kalau sess != nil → deadline diambil dari session (EndsAt/StartsAt/Date).
// - Kalau sess == nil → fallback ke AssessmentDueAt seperti biasa.
func ComputeIsOpenWithCollectSession(
	a *assessmentModel.AssessmentModel,
	sess *sessionModel.ClassAttendanceSessionModel,
	now time.Time,
) bool {
	effectiveDue := a.AssessmentDueAt

	if sess != nil {
		if d := deadlineFromSession(sess); d != nil {
			effectiveDue = d
		}
	}

	return computeIsOpenCore(a, now, effectiveDue)
}
