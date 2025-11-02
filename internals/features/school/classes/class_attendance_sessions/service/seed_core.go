package service

import (
	"gorm.io/gorm"
)

// EnsureSessionSeeded: untuk dipanggil di handler (JIT fallback)
func EnsureSessionSeeded(db *gorm.DB, sessionID, schoolID string, autoOpen bool) error {
	return db.Transaction(func(tx *gorm.DB) error {
		return ensureSessionSeededTx(tx, sessionID, schoolID, autoOpen)
	})
}

// ensureSessionSeededTx: dipakai oleh worker & JIT; idempotent
// ensureSessionSeededTx: dipakai oleh worker & JIT; idempotent
func ensureSessionSeededTx(tx *gorm.DB, sessionID, schoolID string, autoOpen bool) error {
	// langsung insert user_attendance, biarkan ON CONFLICT jadi idempotent
	if err := tx.Exec(`
		INSERT INTO user_attendance (
			user_attendance_school_id,
			user_attendance_session_id,
			user_attendance_school_student_id,
			user_attendance_status
		)
		SELECT
			? AS school_id,
			? AS session_id,
			ms.school_student_id,
			'unmarked'
		FROM class_attendance_sessions s
		JOIN class_sections sec
		  ON sec.class_sections_id = s.class_attendance_sessions_section_id
		JOIN class_section_students css
		  ON css.class_section_students_section_id = sec.class_sections_id
		 AND css.class_section_students_is_active = TRUE
		JOIN school_students ms
		  ON ms.school_student_id = css.class_section_students_school_student_id
		 AND ms.school_student_deleted_at IS NULL
		WHERE s.class_attendance_sessions_id = ?
		  AND s.class_attendance_sessions_school_id = ?
		ON CONFLICT ON CONSTRAINT uq_user_attendance_alive DO NOTHING
	`, schoolID, sessionID, sessionID, schoolID).Error; err != nil {
		return err
	}

	if autoOpen {
		// hanya update status jika masih scheduled
		return tx.Exec(`
			UPDATE class_attendance_sessions
			SET class_attendance_sessions_status = CASE
			      WHEN class_attendance_sessions_status = 'scheduled' THEN 'open'
			      ELSE class_attendance_sessions_status END
			WHERE class_attendance_sessions_id = ?
		`, sessionID).Error
	}
	return nil
}
