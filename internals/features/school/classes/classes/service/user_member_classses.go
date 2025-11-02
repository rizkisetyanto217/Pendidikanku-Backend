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
	OnEnrollmentActivated(tx *gorm.DB, userID, schoolID, assignedBy uuid.UUID) error
	OnEnrollmentDeactivated(tx *gorm.DB, userID, schoolID uuid.UUID) error

	// Utilities
	GrantRole(tx *gorm.DB, userID uuid.UUID, roleName string, schoolID *uuid.UUID, assignedBy uuid.UUID) error
	RevokeRole(tx *gorm.DB, userID uuid.UUID, roleName string, schoolID *uuid.UUID) error
	EnsureSchoolStudentStatus(tx *gorm.DB, userID, schoolID uuid.UUID, status string) error
}

type membershipSvc struct{}

func New() Service { return &membershipSvc{} }

// ————————————————————————————
// Hooks
// ————————————————————————————

func (s *membershipSvc) OnEnrollmentActivated(tx *gorm.DB, userID, schoolID, assignedBy uuid.UUID) error {
	log.Printf("[membership] OnEnrollmentActivated user=%s school=%s assignedBy=%s", userID, schoolID, assignedBy)

	// Grant scoped role: student@school
	if err := s.GrantRole(tx, userID, "student", &schoolID, assignedBy); err != nil {
		log.Printf("[membership] GrantRole ERROR user=%s school=%s err=%v", userID, schoolID, err)
		return err
	}

	// Ensure school_students → active
	if err := s.EnsureSchoolStudentStatus(tx, userID, schoolID, StatusActive); err != nil {
		log.Printf("[membership] EnsureSchoolStudentStatus ERROR user=%s school=%s err=%v", userID, schoolID, err)
		return err
	}

	log.Printf("[membership] OnEnrollmentActivated DONE user=%s school=%s", userID, schoolID)
	return nil
}

func (s *membershipSvc) OnEnrollmentDeactivated(tx *gorm.DB, userID, schoolID uuid.UUID) error {
	// Kebijakan minimal: role tidak otomatis dicabut; hanya turunkan status ms ke inactive.
	return s.EnsureSchoolStudentStatus(tx, userID, schoolID, StatusInactive)
}

// ————————————————————————————
// Utilities
// ————————————————————————————

func (s *membershipSvc) GrantRole(tx *gorm.DB, userID uuid.UUID, roleName string, schoolID *uuid.UUID, assignedBy uuid.UUID) error {
	role := sanitize(roleName)
	var idStr string

	if schoolID == nil {
		err := tx.Raw(
			`SELECT fn_grant_role(?::uuid, ?::text, NULL::uuid, ?::uuid)::text AS id`,
			userID, role, assignedBy,
		).Scan(&idStr).Error
		log.Printf("[membership] GrantRole user=%s role=%s school=NULL assignedBy=%s -> id=%s err=%v",
			userID, role, assignedBy, idStr, err)
		return err
	}

	err := tx.Raw(
		`SELECT fn_grant_role(?::uuid, ?::text, ?::uuid, ?::uuid)::text AS id`,
		userID, role, *schoolID, assignedBy,
	).Scan(&idStr).Error
	log.Printf("[membership] GrantRole user=%s role=%s school=%s assignedBy=%s -> id=%s err=%v",
		userID, role, *schoolID, assignedBy, idStr, err)
	return err
}

func (s *membershipSvc) RevokeRole(tx *gorm.DB, userID uuid.UUID, roleName string, schoolID *uuid.UUID) error {
	role := sanitize(roleName)
	var ok bool

	// fn_revoke_role(user, role, school|null) → boolean
	if schoolID == nil {
		return tx.Raw(
			`SELECT fn_revoke_role(?::uuid, ?::text, NULL::uuid)`,
			userID, role,
		).Scan(&ok).Error
	}

	return tx.Raw(
		`SELECT fn_revoke_role(?::uuid, ?::text, ?::uuid)`,
		userID, role, *schoolID,
	).Scan(&ok).Error
}

// EnsureSchoolStudentStatus memastikan ada baris school_students (alive) untuk (user, school)
// dan mengeset status-nya (active|inactive|alumni). Idempotent & revive bila soft-deleted.
func (s *membershipSvc) EnsureSchoolStudentStatus(tx *gorm.DB, userID, schoolID uuid.UUID, status string) error {
	status = sanitize(status)
	if !isValidStatus(status) {
		return ErrInvalidSchoolStudentStatus
	}

	// ——— BEFORE: status existing (kalau ada) ———
	var before sql.NullString
	_ = tx.Raw(`
		SELECT school_student_status
		FROM school_students
		WHERE school_student_user_id = @user
		  AND school_student_school_id = @school
		  AND school_student_deleted_at IS NULL
		ORDER BY school_student_created_at DESC
		LIMIT 1
	`, sql.Named("user", userID), sql.Named("school", schoolID)).Scan(&before).Error
	log.Printf("[membership] MS Ensure BEFORE user=%s school=%s current=%s target=%s",
		userID, schoolID, nullStr(before), status)

	// ——— CTE dengan indikator cabang ———
	const q = `
WITH revived AS (
  UPDATE school_students
     SET school_student_deleted_at = NULL,
         school_student_status = @status,
         school_student_updated_at = now()
   WHERE school_student_user_id = @user
     AND school_student_school_id = @school
     AND school_student_deleted_at IS NOT NULL
  RETURNING 1
),
updated AS (
  UPDATE school_students
     SET school_student_status = @status,
         school_student_updated_at = now()
   WHERE school_student_user_id = @user
     AND school_student_school_id = @school
     AND school_student_deleted_at IS NULL
     AND school_student_status <> @status
  RETURNING 1
),
inserted AS (
  INSERT INTO school_students (school_student_user_id, school_student_school_id, school_student_status)
  SELECT @user, @school, @status
  WHERE NOT EXISTS (
    SELECT 1 FROM school_students
     WHERE school_student_user_id = @user
       AND school_student_school_id = @school
       AND school_student_deleted_at IS NULL
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
		sql.Named("school", schoolID),
		sql.Named("status", status),
	).Scan(&meta).Error; err != nil {
		log.Printf("[membership] MS Ensure ERROR user=%s school=%s err=%v", userID, schoolID, err)
		return err
	}

	// ——— AFTER: status existing (kalau ada) ———
	var after sql.NullString
	_ = tx.Raw(`
		SELECT school_student_status
		FROM school_students
		WHERE school_student_user_id = @user
		  AND school_student_school_id = @school
		  AND school_student_deleted_at IS NULL
		ORDER BY school_student_created_at DESC
		LIMIT 1
	`, sql.Named("user", userID), sql.Named("school", schoolID)).Scan(&after).Error

	log.Printf("[membership] MS Ensure AFTER  user=%s school=%s result[revived=%d updated=%d inserted=%d] => current=%s",
		userID, schoolID, meta.Revived, meta.Updated, meta.Inserted, nullStr(after))

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

var ErrInvalidSchoolStudentStatus = errors.New("invalid school_student_status")

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
