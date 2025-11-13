// // file: internals/features/school/classes/class_attendance_sessions/service/seed_core.go
package service

// import (
// 	"gorm.io/gorm"
// )

// // EnsureSessionSeeded: untuk dipanggil di handler (JIT fallback)
// func EnsureSessionSeeded(db *gorm.DB, sessionID, schoolID string, autoOpen bool) error {
// 	return db.Transaction(func(tx *gorm.DB) error {
// 		return ensureSessionSeededTx(tx, sessionID, schoolID, autoOpen)
// 	})
// }

// // ensureSessionSeededTx: dipakai oleh worker & JIT; idempotent
// func ensureSessionSeededTx(tx *gorm.DB, sessionID, schoolID string, autoOpen bool) error {
// 	// INSERT participants (student) untuk 1 session
// 	if err := tx.Exec(`
// 		INSERT INTO class_attendance_session_participants (
// 			class_attendance_session_participant_school_id,
// 			class_attendance_session_participant_session_id,
// 			class_attendance_session_participant_kind,
// 			class_attendance_session_participant_school_student_id,
// 			class_attendance_session_participant_state
// 		)
// 		SELECT
// 			s.class_attendance_session_school_id,
// 			s.class_attendance_session_id,
// 			'student'::participant_kind_enum,
// 			ms.school_student_id,
// 			'unmarked'::attendance_state_enum
// 		FROM class_attendance_sessions s
// 		JOIN class_sections sec
// 		  ON sec.class_section_id = s.class_attendance_session_section_id_snapshot
// 		JOIN student_class_sections scs
// 		  ON scs.student_class_section_section_id = sec.class_section_id
// 		 AND scs.student_class_section_is_active = TRUE
// 		JOIN school_students ms
// 		  ON ms.school_student_id = scs.student_class_section_school_student_id
// 		 AND ms.school_student_deleted_at IS NULL
// 		WHERE s.class_attendance_session_id = ?
// 		  AND s.class_attendance_session_school_id = ?
// 		ON CONFLICT ON CONSTRAINT uq_casp_student_alive DO NOTHING
// 	`, sessionID, schoolID).Error; err != nil {
// 		return err
// 	}

// 	if autoOpen {
// 		// buka status attendance & jadikan sesi ongoing kalau masih scheduled
// 		if err := tx.Exec(`
// 			UPDATE class_attendance_sessions
// 			SET
// 				class_attendance_session_status = CASE
// 					WHEN class_attendance_session_status = 'scheduled'
// 					THEN 'ongoing'
// 					ELSE class_attendance_session_status
// 				END,
// 				class_attendance_session_attendance_status = 'open'
// 			WHERE class_attendance_session_id = ?
// 			  AND class_attendance_session_school_id = ?
// 		`, sessionID, schoolID).Error; err != nil {
// 			return err
// 		}
// 	}

// 	return nil
// }
