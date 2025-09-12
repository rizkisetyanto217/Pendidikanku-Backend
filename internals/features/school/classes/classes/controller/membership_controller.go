// file: internals/services/membership/controller/membership_controller.go
package controller

import (
	"errors"
	"log"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	ucModel "masjidku_backend/internals/features/school/classes/classes/model"
	membership "masjidku_backend/internals/features/school/classes/classes/service"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
)

/*
Route summary
POST /api/a/membership/enrollment/activate      -> aktifkan enrolment (grant student@scope + ensure ms active)
POST /api/a/membership/enrollment/deactivate    -> nonaktifkan enrolment (ensure ms inactive)
POST /api/a/membership/roles/grant              -> grant role arbitrary (global / scoped)
POST /api/a/membership/roles/revoke             -> revoke role arbitrary (global / scoped)
POST /api/a/membership/masjid-students/ensure   -> set status ms: active|inactive|alumni
*/

type MembershipController struct {
	DB  *gorm.DB
	Svc membership.Service
	V   *validator.Validate
}

func NewMembershipController(db *gorm.DB, svc membership.Service) *MembershipController {
	if svc == nil {
		svc = membership.New()
	}
	return &MembershipController{
		DB:  db,
		Svc: svc,
		V:   validator.New(),
	}
}

/* ====================== DTOs ====================== */

type grantReq struct {
	UserID   uuid.UUID  `json:"user_id" validate:"required,uuid"`
	RoleName string     `json:"role_name" validate:"required,min=2,max=32"`
	MasjidID *uuid.UUID `json:"masjid_id" validate:"omitempty,uuid"` // NULL = global role
}

type revokeReq = grantReq

type ensureMsReq struct {
	UserID   uuid.UUID  `json:"user_id" validate:"required,uuid"`
	MasjidID *uuid.UUID `json:"masjid_id" validate:"omitempty,uuid"`
	Status   string     `json:"status"  validate:"required,oneof=active inactive alumni"`
}

type enrollReq struct {
	UserID        uuid.UUID  `json:"user_id" validate:"required,uuid"`
	MasjidID      *uuid.UUID `json:"masjid_id" validate:"omitempty,uuid"`
	UserClassesID *uuid.UUID `json:"user_classes_id" validate:"omitempty,uuid"` // jika diisi: approve enrolment sekalian
	JoinedAt      *time.Time `json:"joined_at" validate:"omitempty"`
}

/* ================== Handlers ================== */

// POST /api/a/membership/enrollment/activate
func (h *MembershipController) ActivateEnrollment(c *fiber.Ctx) error {
	var req enrollReq
	if err := c.BodyParser(&req); err != nil {
	 return fiber.NewError(fiber.StatusBadRequest, "payload invalid")
	}
	if err := h.V.Struct(req); err != nil {
	 return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// ⬇️ Ambil masjid dari token; jika body kirim masjid_id, wajib sama
	mid, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil || mid == uuid.Nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}
	if req.MasjidID != nil && *req.MasjidID != uuid.Nil && *req.MasjidID != mid {
		return fiber.NewError(fiber.StatusForbidden, "masjid_id pada body tidak boleh berbeda dengan token")
	}
	masjidID := mid

	assignedBy, err := helperAuth.GetUserIDFromToken(c)
	if err != nil {
		return err
	}

	log.Printf("[membership] ActivateEnrollment IN user=%s masjid=%s assignedBy=%s user_classes_id=%v",
		req.UserID, masjidID, assignedBy, req.UserClassesID)

	return h.DB.Transaction(func(tx *gorm.DB) error {
		// 1) hooks membership (role + ms status)
		if err := h.Svc.OnEnrollmentActivated(tx, req.UserID, masjidID, assignedBy); err != nil {
			log.Printf("[membership] ActivateEnrollment hooks ERROR: %v", err)
		 return err
		}

		// 2) (opsional) approve enrolment
		if req.UserClassesID != nil && *req.UserClassesID != uuid.Nil {
			var row struct {
				MasjidID uuid.UUID `gorm:"column:user_classes_masjid_id"`
				MSID     uuid.UUID `gorm:"column:user_classes_masjid_student_id"`
				ClassID  uuid.UUID `gorm:"column:user_classes_class_id"`
				Owner    uuid.UUID `gorm:"column:masjid_student_user_id"`
			}
			q := tx.Table("user_classes uc").
				Select(`
					uc.user_classes_masjid_id,
					uc.user_classes_masjid_student_id,
					uc.user_classes_class_id,
					ms.masjid_student_user_id
				`).
				Joins(`JOIN masjid_students ms
					ON ms.masjid_student_id = uc.user_classes_masjid_student_id
					AND ms.masjid_student_deleted_at IS NULL`).
				Where("uc.user_classes_id = ? AND uc.user_classes_deleted_at IS NULL", *req.UserClassesID)

			if err := q.First(&row).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fiber.NewError(fiber.StatusNotFound, "Enrolment tidak ditemukan")
				}
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil enrolment")
			}
			if row.MasjidID != masjidID {
				return fiber.NewError(fiber.StatusBadRequest, "Enrolment bukan milik masjid ini")
			}
			if row.Owner != req.UserID {
				return fiber.NewError(fiber.StatusBadRequest, "Enrolment bukan milik user tersebut")
			}

			updates := map[string]any{
				"user_classes_status":     ucModel.UserClassStatusActive,
				"user_classes_updated_at": time.Now(),
			}
			if req.JoinedAt != nil {
				updates["user_classes_joined_at"] = *req.JoinedAt
			}

			if err := tx.Model(&ucModel.UserClassesModel{}).
				Where("user_classes_id = ? AND user_classes_deleted_at IS NULL", *req.UserClassesID).
				Updates(updates).Error; err != nil {

				if strings.Contains(strings.ToLower(err.Error()), "duplicate") || errors.Is(err, gorm.ErrDuplicatedKey) {
					return fiber.NewError(fiber.StatusConflict, "Sudah ada enrolment aktif untuk kelas ini")
				}
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengaktifkan enrolment")
			}

			log.Printf("[membership] Enrolment APPROVED user_classes_id=%s class=%s msid=%s",
				*req.UserClassesID, row.ClassID, row.MSID)
		}

		return helper.JsonOK(c, "enrollment activated", fiber.Map{
			"user_id":         req.UserID,
			"masjid_id":       masjidID,
			"user_classes_id": req.UserClassesID,
		})
	})
}

// POST /api/a/membership/enrollment/deactivate
func (h *MembershipController) DeactivateEnrollment(c *fiber.Ctx) error {
	var req enrollReq
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "payload invalid")
	}
	if err := h.V.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// ⬇️ Ambil masjid dari token; jika body kirim masjid_id, wajib sama
	mid, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil || mid == uuid.Nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}
	if req.MasjidID != nil && *req.MasjidID != uuid.Nil && *req.MasjidID != mid {
		return fiber.NewError(fiber.StatusForbidden, "masjid_id pada body tidak boleh berbeda dengan token")
	}
	masjidID := mid

	log.Printf("[membership] DeactivateEnrollment IN user=%s masjid=%s user_classes_id=%v",
		req.UserID, masjidID, req.UserClassesID)

	return h.DB.Transaction(func(tx *gorm.DB) error {
		// 1) Hooks membership: ensure masjid_students → inactive
		if err := h.Svc.OnEnrollmentDeactivated(tx, req.UserID, masjidID); err != nil {
			return err
		}

		// 2) (opsional) set enrolment -> inactive
		if req.UserClassesID != nil && *req.UserClassesID != uuid.Nil {
			var row struct {
				MasjidID uuid.UUID `gorm:"column:user_classes_masjid_id"`
				Owner    uuid.UUID `gorm:"column:masjid_student_user_id"`
			}
			q := tx.Table("user_classes uc").
				Select(`
					uc.user_classes_masjid_id,
					ms.masjid_student_user_id
				`).
				Joins(`JOIN masjid_students ms
					ON ms.masjid_student_id = uc.user_classes_masjid_student_id
					AND ms.masjid_student_deleted_at IS NULL`).
				Where("uc.user_classes_id = ? AND uc.user_classes_deleted_at IS NULL", *req.UserClassesID)

			if err := q.First(&row).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fiber.NewError(fiber.StatusNotFound, "Enrolment tidak ditemukan")
				}
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil enrolment")
			}
			if row.MasjidID != masjidID {
				return fiber.NewError(fiber.StatusBadRequest, "Enrolment bukan milik masjid ini")
			}
			if row.Owner != req.UserID {
				return fiber.NewError(fiber.StatusBadRequest, "Enrolment bukan milik user tersebut")
			}

			if err := tx.Model(&ucModel.UserClassesModel{}).
				Where("user_classes_id = ? AND user_classes_deleted_at IS NULL", *req.UserClassesID).
				Updates(map[string]any{
					"user_classes_status":     ucModel.UserClassStatusInactive,
					"user_classes_updated_at": time.Now(),
				}).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal menonaktifkan enrolment")
			}
		}

		return helper.JsonOK(c, "enrollment deactivated", fiber.Map{
			"user_id":         req.UserID,
			"masjid_id":       masjidID,
			"user_classes_id": req.UserClassesID,
		})
	})
}

// POST /api/a/membership/roles/grant
func (h *MembershipController) GrantRole(c *fiber.Ctx) error {
	var req grantReq
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "payload invalid")
	}
	req.RoleName = strings.ToLower(strings.TrimSpace(req.RoleName))
	if err := h.V.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	masjidID := req.MasjidID
	if masjidID == nil && roleNeedsScope(req.RoleName) {
		mID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
		if err != nil || mID == uuid.Nil {
			return fiber.NewError(fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
		}
		masjidID = &mID
	}
	assignedBy, err := helperAuth.GetUserIDFromToken(c)
	if err != nil {
		return err
	}

	return h.DB.Transaction(func(tx *gorm.DB) error {
		if err := h.Svc.GrantRole(tx, req.UserID, req.RoleName, masjidID, assignedBy); err != nil {
			return err
		}
		return helper.JsonOK(c, "role granted", fiber.Map{
			"user_id":   req.UserID,
			"role_name": req.RoleName,
			"masjid_id": masjidID,
		})
	})
}

// POST /api/a/membership/roles/revoke
func (h *MembershipController) RevokeRole(c *fiber.Ctx) error {
	var req revokeReq
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "payload invalid")
	}
	req.RoleName = strings.ToLower(strings.TrimSpace(req.RoleName))
	if err := h.V.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	masjidID := req.MasjidID
	if masjidID == nil && roleNeedsScope(req.RoleName) {
		mID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
		if err != nil || mID == uuid.Nil {
			return fiber.NewError(fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
		}
		masjidID = &mID
	}

	return h.DB.Transaction(func(tx *gorm.DB) error {
		if err := h.Svc.RevokeRole(tx, req.UserID, req.RoleName, masjidID); err != nil {
			return err
		}
		return helper.JsonOK(c, "role revoked", fiber.Map{
			"user_id":   req.UserID,
			"role_name": req.RoleName,
			"masjid_id": masjidID,
		})
	})
}

// POST /api/a/membership/masjid-students/ensure
func (h *MembershipController) EnsureMasjidStudent(c *fiber.Ctx) error {
	var req ensureMsReq
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "payload invalid")
	}
	req.Status = strings.ToLower(strings.TrimSpace(req.Status))
	if err := h.V.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// ⬇️ Ambil masjid dari token; jika body kirim masjid_id, wajib sama
	mid, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil || mid == uuid.Nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}
	if req.MasjidID != nil && *req.MasjidID != uuid.Nil && *req.MasjidID != mid {
		return fiber.NewError(fiber.StatusForbidden, "masjid_id pada body tidak boleh berbeda dengan token")
	}
	masjidID := mid

	return h.DB.Transaction(func(tx *gorm.DB) error {
		if err := h.Svc.EnsureMasjidStudentStatus(tx, req.UserID, masjidID, req.Status); err != nil {
			return err
		}
		return helper.JsonOK(c, "masjid_student ensured", fiber.Map{
			"user_id":   req.UserID,
			"masjid_id": masjidID,
			"status":    req.Status,
		})
	})
}

/* ===== Kebijakan peran: mana yang tipikal perlu scope masjid? ===== */
func roleNeedsScope(role string) bool {
	switch role {
	case "owner", "admin", "treasurer", "dkm", "teacher", "student":
		return true
	default:
		return false
	}
}
