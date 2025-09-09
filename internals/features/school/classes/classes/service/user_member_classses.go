// file: internals/features/school/classes/classes/service/membership_service.go
package service

import (
	"database/sql"
	"errors"
	"log"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ————————————————————————————
// Public API
// ————————————————————————————

type Service interface {
	// Hooks enrolment
	OnEnrollmentActivated(tx *gorm.DB, userID, masjidID, assignedBy uuid.UUID) error
	OnEnrollmentDeactivated(tx *gorm.DB, userID, masjidID uuid.UUID) error

	// Utilities
	GrantRole(tx *gorm.DB, userID uuid.UUID, roleName string, masjidID *uuid.UUID, assignedBy uuid.UUID) error
	RevokeRole(tx *gorm.DB, userID uuid.UUID, roleName string, masjidID *uuid.UUID) error
	EnsureMasjidStudentStatus(tx *gorm.DB, userID, masjidID uuid.UUID, status string) error
}

type membershipSvc struct{}

func New() Service { return &membershipSvc{} }

// ————————————————————————————
// Hooks
// ————————————————————————————

func (s *membershipSvc) OnEnrollmentActivated(tx *gorm.DB, userID, masjidID, assignedBy uuid.UUID) error {
	log.Printf("[membership] OnEnrollmentActivated user=%s masjid=%s assignedBy=%s", userID, masjidID, assignedBy)

	// Grant scoped role: student@masjid
	if err := s.GrantRole(tx, userID, "student", &masjidID, assignedBy); err != nil {
		log.Printf("[membership] GrantRole ERROR user=%s masjid=%s err=%v", userID, masjidID, err)
		return err
	}

	// Ensure masjid_students → active
	if err := s.EnsureMasjidStudentStatus(tx, userID, masjidID, StatusActive); err != nil {
		log.Printf("[membership] EnsureMasjidStudentStatus ERROR user=%s masjid=%s err=%v", userID, masjidID, err)
		return err
	}

	log.Printf("[membership] OnEnrollmentActivated DONE user=%s masjid=%s", userID, masjidID)
	return nil
}

func (s *membershipSvc) OnEnrollmentDeactivated(tx *gorm.DB, userID, masjidID uuid.UUID) error {
	// Kebijakan minimal: role tidak otomatis dicabut; hanya turunkan status ms ke inactive.
	return s.EnsureMasjidStudentStatus(tx, userID, masjidID, StatusInactive)
}

// ————————————————————————————
// Utilities
// ————————————————————————————

func (s *membershipSvc) GrantRole(tx *gorm.DB, userID uuid.UUID, roleName string, masjidID *uuid.UUID, assignedBy uuid.UUID) error {
	role := sanitize(roleName)
	var idStr string

	if masjidID == nil {
		err := tx.Raw(
			`SELECT fn_grant_role(?::uuid, ?::text, NULL::uuid, ?::uuid)::text AS id`,
			userID, role, assignedBy,
		).Scan(&idStr).Error
		log.Printf("[membership] GrantRole user=%s role=%s masjid=NULL assignedBy=%s -> id=%s err=%v",
			userID, role, assignedBy, idStr, err)
		return err
	}

	err := tx.Raw(
		`SELECT fn_grant_role(?::uuid, ?::text, ?::uuid, ?::uuid)::text AS id`,
		userID, role, *masjidID, assignedBy,
	).Scan(&idStr).Error
	log.Printf("[membership] GrantRole user=%s role=%s masjid=%s assignedBy=%s -> id=%s err=%v",
		userID, role, *masjidID, assignedBy, idStr, err)
	return err
}

func (s *membershipSvc) RevokeRole(tx *gorm.DB, userID uuid.UUID, roleName string, masjidID *uuid.UUID) error {
	role := sanitize(roleName)
	var ok bool

	// fn_revoke_role(user, role, masjid|null) → boolean
	if masjidID == nil {
		return tx.Raw(
			`SELECT fn_revoke_role(?::uuid, ?::text, NULL::uuid)`,
			userID, role,
		).Scan(&ok).Error
	}

	return tx.Raw(
		`SELECT fn_revoke_role(?::uuid, ?::text, ?::uuid)`,
		userID, role, *masjidID,
	).Scan(&ok).Error
}

// EnsureMasjidStudentStatus memastikan ada baris masjid_students (alive) untuk (user, masjid)
// dan mengeset status-nya (active|inactive|alumni). Idempotent & revive bila soft-deleted.
func (s *membershipSvc) EnsureMasjidStudentStatus(tx *gorm.DB, userID, masjidID uuid.UUID, status string) error {
	status = sanitize(status)
	if !isValidStatus(status) {
		return ErrInvalidMasjidStudentStatus
	}

	// ——— BEFORE: status existing (kalau ada) ———
	var before sql.NullString
	_ = tx.Raw(`
		SELECT masjid_student_status
		FROM masjid_students
		WHERE masjid_student_user_id = @user
		  AND masjid_student_masjid_id = @masjid
		  AND masjid_student_deleted_at IS NULL
		ORDER BY masjid_student_created_at DESC
		LIMIT 1
	`, sql.Named("user", userID), sql.Named("masjid", masjidID)).Scan(&before).Error
	log.Printf("[membership] MS Ensure BEFORE user=%s masjid=%s current=%s target=%s",
		userID, masjidID, nullStr(before), status)

	// ——— CTE dengan indikator cabang ———
	const q = `
WITH revived AS (
  UPDATE masjid_students
     SET masjid_student_deleted_at = NULL,
         masjid_student_status = @status,
         masjid_student_updated_at = now()
   WHERE masjid_student_user_id = @user
     AND masjid_student_masjid_id = @masjid
     AND masjid_student_deleted_at IS NOT NULL
  RETURNING 1
),
updated AS (
  UPDATE masjid_students
     SET masjid_student_status = @status,
         masjid_student_updated_at = now()
   WHERE masjid_student_user_id = @user
     AND masjid_student_masjid_id = @masjid
     AND masjid_student_deleted_at IS NULL
     AND masjid_student_status <> @status
  RETURNING 1
),
inserted AS (
  INSERT INTO masjid_students (masjid_student_user_id, masjid_student_masjid_id, masjid_student_status)
  SELECT @user, @masjid, @status
  WHERE NOT EXISTS (
    SELECT 1 FROM masjid_students
     WHERE masjid_student_user_id = @user
       AND masjid_student_masjid_id = @masjid
       AND masjid_student_deleted_at IS NULL
  )
  RETURNING 1
)
SELECT 
  COALESCE((SELECT COUNT(1) FROM revived),  0) AS revived,
  COALESCE((SELECT COUNT(1) FROM updated),  0) AS updated,
  COALESCE((SELECT COUNT(1) FROM inserted), 0) AS inserted;
`
	var meta struct {
		Revived  int `gorm:"column:revived"`
		Updated  int `gorm:"column:updated"`
		Inserted int `gorm:"column:inserted"`
	}
	if err := tx.Raw(q,
		sql.Named("user", userID),
		sql.Named("masjid", masjidID),
		sql.Named("status", status),
	).Scan(&meta).Error; err != nil {
		log.Printf("[membership] MS Ensure ERROR user=%s masjid=%s err=%v", userID, masjidID, err)
		return err
	}

	// ——— AFTER: status existing (kalau ada) ———
	var after sql.NullString
	_ = tx.Raw(`
		SELECT masjid_student_status
		FROM masjid_students
		WHERE masjid_student_user_id = @user
		  AND masjid_student_masjid_id = @masjid
		  AND masjid_student_deleted_at IS NULL
		ORDER BY masjid_student_created_at DESC
		LIMIT 1
	`, sql.Named("user", userID), sql.Named("masjid", masjidID)).Scan(&after).Error

	log.Printf("[membership] MS Ensure AFTER  user=%s masjid=%s result[revived=%d updated=%d inserted=%d] => current=%s",
		userID, masjidID, meta.Revived, meta.Updated, meta.Inserted, nullStr(after))

	return nil
}

// ————————————————————————————
// Private helpers & constants
// ————————————————————————————

func nullStr(s sql.NullString) string {
	if !s.Valid {
		return "<nil>"
	}
	return s.String
}

const (
	StatusActive   = "active"
	StatusInactive = "inactive"
	StatusAlumni   = "alumni"
)

var ErrInvalidMasjidStudentStatus = errors.New("invalid masjid_student_status")

func isValidStatus(s string) bool {
	switch s {
	case StatusActive, StatusInactive, StatusAlumni:
		return true
	default:
		return false
	}
}

func sanitize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
