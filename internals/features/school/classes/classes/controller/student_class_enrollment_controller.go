// file: internals/features/school/classes/class_enrollments/controller/student_class_enrollments_controller.go
package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	dto "schoolku_backend/internals/features/school/classes/classes/dto"
	emodel "schoolku_backend/internals/features/school/classes/classes/model"
	helper "schoolku_backend/internals/helpers"
	helperAuth "schoolku_backend/internals/helpers/auth"
)

/* =======================================================
   Controller bootstrap
======================================================= */

type StudentClassEnrollmentController struct {
	DB *gorm.DB
}

func NewStudentClassEnrollmentController(db *gorm.DB) *StudentClassEnrollmentController {
	return &StudentClassEnrollmentController{DB: db}
}

/* =======================================================
   Helpers (local)
======================================================= */

func parseUUIDParam(c *fiber.Ctx, name string) (uuid.UUID, error) {
	raw := strings.TrimSpace(c.Params(name))
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid %s", name)
	}
	return id, nil
}

func parseStatusInParam(raw string) ([]emodel.ClassEnrollmentStatus, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	out := make([]emodel.ClassEnrollmentStatus, 0, len(parts))
	seen := map[emodel.ClassEnrollmentStatus]struct{}{}
	for _, p := range parts {
		v := emodel.ClassEnrollmentStatus(strings.ToLower(strings.TrimSpace(p)))
		switch v {
		case emodel.EnrollmentInitiated,
			emodel.EnrollmentPendingReview,
			emodel.EnrollmentAwaitingPay,
			emodel.EnrollmentAccepted,
			emodel.EnrollmentWaitlisted,
			emodel.EnrollmentRejected,
			emodel.EnrollmentCanceled:
			if _, ok := seen[v]; !ok {
				seen[v] = struct{}{}
				out = append(out, v)
			}
		default:
			return nil, fmt.Errorf("invalid status_in value: %q", p)
		}
	}
	return out, nil
}

func orderClause(orderBy, sort string) string {
	col := "student_class_enrollments_created_at"
	switch strings.ToLower(strings.TrimSpace(orderBy)) {
	case "applied_at":
		col = "student_class_enrollments_applied_at"
	case "updated_at":
		col = "student_class_enrollments_updated_at"
	case "created_at", "":
		col = "student_class_enrollments_created_at"
	}
	dir := "DESC"
	if strings.EqualFold(strings.TrimSpace(sort), "asc") {
		dir = "ASC"
	}
	return col + " " + dir
}

func nowPtr() *time.Time {
	t := time.Now()
	return &t
}

func enrichEnrollmentExtras(ctx context.Context, db *gorm.DB, schoolID uuid.UUID, items []dto.StudentClassEnrollmentResponse) {
	if len(items) == 0 {
		return
	}

	// Kumpulkan ID unik
	stuIDsSet := map[uuid.UUID]struct{}{}
	classIDsSet := map[uuid.UUID]struct{}{}
	for _, it := range items {
		stuIDsSet[it.StudentClassEnrollmentSchoolStudentID] = struct{}{}
		classIDsSet[it.StudentClassEnrollmentClassID] = struct{}{}
	}
	stuIDs := make([]uuid.UUID, 0, len(stuIDsSet))
	classIDs := make([]uuid.UUID, 0, len(classIDsSet))
	for id := range stuIDsSet {
		stuIDs = append(stuIDs, id)
	}
	for id := range classIDsSet {
		classIDs = append(classIDs, id)
	}

	// ===== Ambil students (name, user_id)
	type stuRow struct {
		ID     uuid.UUID  `gorm:"column:school_student_id"`
		Name   string     `gorm:"column:name"`
		UserID *uuid.UUID `gorm:"column:user_id"`
	}
	stuMap := make(map[uuid.UUID]stuRow, len(stuIDs))
	if len(stuIDs) > 0 {
		var stus []stuRow
		_ = db.WithContext(ctx).
			Table("school_students").
			Select("school_student_id, name, user_id").
			Where("school_id = ? AND school_student_id IN ?", schoolID, stuIDs).
			Find(&stus).Error
		for _, s := range stus {
			stuMap[s.ID] = s
		}
	}

	// ===== Ambil classes (class_name)
	type clsRow struct {
		ID        uuid.UUID `gorm:"column:class_id"`
		ClassName string    `gorm:"column:class_name"`
	}
	clsMap := make(map[uuid.UUID]clsRow, len(classIDs))
	if len(classIDs) > 0 {
		var clss []clsRow
		_ = db.WithContext(ctx).
			Table("classes").
			Select("class_id, class_name").
			Where("class_school_id = ? AND class_id IN ?", schoolID, classIDs).
			Find(&clss).Error
		for _, c := range clss {
			clsMap[c.ID] = c
		}
	}

	// ===== Ambil usernames (batch) opsional
	userIDsSet := map[uuid.UUID]struct{}{}
	for _, s := range stuMap {
		if s.UserID != nil {
			userIDsSet[*s.UserID] = struct{}{}
		}
	}
	userIDs := make([]uuid.UUID, 0, len(userIDsSet))
	for id := range userIDsSet {
		userIDs = append(userIDs, id)
	}
	type userRow struct {
		ID       uuid.UUID `gorm:"column:id"`
		UserName string    `gorm:"column:user_name"`
	}
	uMap := make(map[uuid.UUID]string, len(userIDs))
	if len(userIDs) > 0 {
		var us []userRow
		_ = db.WithContext(ctx).
			Table("users").
			Select("id, user_name").
			Where("id IN ?", userIDs).
			Find(&us).Error
		for _, u := range us {
			uMap[u.ID] = u.UserName
		}
	}

	// ===== Isi ke items
	for i := range items {
		if s, ok := stuMap[items[i].StudentClassEnrollmentSchoolStudentID]; ok {
			items[i].StudentName = s.Name
			if s.UserID != nil {
				if un, ok2 := uMap[*s.UserID]; ok2 {
					username := un
					items[i].Username = &username
				}
			}
		}
		if c, ok := clsMap[items[i].StudentClassEnrollmentClassID]; ok {
			items[i].ClassName = c.ClassName
		}
	}
}

/* =======================================================
   Routes
   - POST   /:school_id/class-enrollments
   - GET    /:school_id/class-enrollments/:id
   - PATCH  /:school_id/class-enrollments/:id
   - PATCH  /:school_id/class-enrollments/:id/status
   - PATCH  /:school_id/class-enrollments/:id/payment
   - DELETE /:school_id/class-enrollments/:id
======================================================= */

// POST /:school_id/class-enrollments
func (ctl *StudentClassEnrollmentController) Create(c *fiber.Ctx) error {
	// tenant
	schoolID, err := helperAuth.ParseSchoolIDFromPath(c)
	if err != nil {
		return err
	}
	if er := helperAuth.EnsureMemberSchool(c, schoolID); er != nil {
		return er
	}

	// body
	var body dto.CreateStudentClassEnrollmentRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid body")
	}

	// ===== Lookup student (pastikan belong to school) =====
	var stu struct {
		ID     uuid.UUID  `gorm:"column:school_student_id"`
		Name   string     `gorm:"column:name"`
		Code   string     `gorm:"column:code"`
		Slug   string     `gorm:"column:slug"`
		UserID *uuid.UUID `gorm:"column:user_id"`
	}
	if err := ctl.DB.WithContext(c.Context()).
		Table("school_students").
		Select("school_student_id, name, code, slug, user_id").
		Where("school_student_id = ? AND school_id = ?", body.SchoolStudentID, schoolID).
		Take(&stu).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusBadRequest, "invalid student for this school")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// ===== Lookup class (pastikan belong to school) =====
	var cls struct {
		ID        uuid.UUID `gorm:"column:class_id"`
		ClassName string    `gorm:"column:class_name"`
		Slug      string    `gorm:"column:class_slug"`
	}
	if err := ctl.DB.WithContext(c.Context()).
		Table("classes").
		Select("class_id, class_name, class_slug").
		Where("class_id = ? AND class_school_id = ?", body.ClassID, schoolID).
		Take(&cls).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusBadRequest, "invalid class for this school")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// ===== Build model & isi SNAPSHOT =====
	m := emodel.StudentClassEnrollmentModel{
		StudentClassEnrollmentSchoolID:        schoolID,
		StudentClassEnrollmentSchoolStudentID: body.SchoolStudentID,
		StudentClassEnrollmentClassID:         body.ClassID,

		StudentClassEnrollmentStatus:      emodel.EnrollmentInitiated,
		StudentClassEnrollmentTotalDueIDR: body.TotalDueIDR,
		StudentClassEnrollmentAppliedAt:   time.Now(),

		// snapshots
		StudentClassEnrollmentClassNameSnapshot:   cls.ClassName,
		StudentClassEnrollmentClassSlugSnapshot:   cls.Slug,
		StudentClassEnrollmentStudentNameSnapshot: stu.Name,
		StudentClassEnrollmentStudentCodeSnapshot: stu.Code,
		StudentClassEnrollmentStudentSlugSnapshot: stu.Slug,
	}

	if body.Preferences != nil {
		if b, er := json.Marshal(body.Preferences); er == nil {
			m.StudentClassEnrollmentPreferences = b
		}
	}

	if err := ctl.DB.WithContext(c.Context()).Create(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// response
	resp := dto.FromModelStudentClassEnrollment(&m)
	list := []dto.StudentClassEnrollmentResponse{resp}
	enrichEnrollmentExtras(c.Context(), ctl.DB, schoolID, list)
	return helper.JsonCreated(c, "created", list[0])
}

// GET /:school_id/class-enrollments/:id
func (ctl *StudentClassEnrollmentController) GetByID(c *fiber.Ctx) error {
	schoolID, err := helperAuth.ParseSchoolIDFromPath(c)
	if err != nil {
		return err
	}
	if er := helperAuth.EnsureMemberSchool(c, schoolID); er != nil {
		return er
	}
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var m emodel.StudentClassEnrollmentModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("student_class_enrollments_school_id = ?", schoolID).
		First(&m, "student_class_enrollments_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonOK(c, "ok", dto.FromModelStudentClassEnrollment(&m))
}

// PATCH /:school_id/class-enrollments/:id
func (ctl *StudentClassEnrollmentController) Update(c *fiber.Ctx) error {
	schoolID, err := helperAuth.ParseSchoolIDFromPath(c)
	if err != nil {
		return err
	}
	if er := helperAuth.EnsureMemberSchool(c, schoolID); er != nil {
		return er
	}
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var body dto.UpdateStudentClassEnrollmentRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid body")
	}

	var m emodel.StudentClassEnrollmentModel
	tx := ctl.DB.WithContext(c.Context()).
		Where("student_class_enrollments_school_id = ?", schoolID).
		First(&m, "student_class_enrollments_id = ?", id)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, tx.Error.Error())
	}

	if body.TotalDueIDR != nil {
		m.StudentClassEnrollmentTotalDueIDR = *body.TotalDueIDR
	}
	if body.Preferences != nil {
		if b, er := json.Marshal(body.Preferences); er == nil {
			m.StudentClassEnrollmentPreferences = b
		}
	}

	if err := ctl.DB.WithContext(c.Context()).Save(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.JsonUpdated(c, "updated", dto.FromModelStudentClassEnrollment(&m))
}

// PATCH /:school_id/class-enrollments/:id/status
func (ctl *StudentClassEnrollmentController) UpdateStatus(c *fiber.Ctx) error {
	schoolID, err := helperAuth.ParseSchoolIDFromPath(c)
	if err != nil {
		return err
	}
	if er := helperAuth.EnsureMemberSchool(c, schoolID); er != nil {
		return er
	}
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var body dto.UpdateStudentClassEnrollmentStatusRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid body")
	}

	var m emodel.StudentClassEnrollmentModel
	tx := ctl.DB.WithContext(c.Context()).
		Where("student_class_enrollments_school_id = ?", schoolID).
		First(&m, "student_class_enrollments_id = ?", id)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, tx.Error.Error())
	}

	// set status + timestamp (isi otomatis jika kosong)
	switch body.Status {
	case emodel.EnrollmentInitiated:
		m.StudentClassEnrollmentStatus = emodel.EnrollmentInitiated
		if m.StudentClassEnrollmentAppliedAt.IsZero() {
			m.StudentClassEnrollmentAppliedAt = time.Now()
		}
	case emodel.EnrollmentPendingReview:
		m.StudentClassEnrollmentStatus = emodel.EnrollmentPendingReview
		if body.ReviewedAt != nil {
			m.StudentClassEnrollmentReviewedAt = body.ReviewedAt
		} else if m.StudentClassEnrollmentReviewedAt == nil {
			m.StudentClassEnrollmentReviewedAt = nowPtr()
		}
	case emodel.EnrollmentAwaitingPay:
		m.StudentClassEnrollmentStatus = emodel.EnrollmentAwaitingPay
		if m.StudentClassEnrollmentReviewedAt == nil {
			m.StudentClassEnrollmentReviewedAt = nowPtr()
		}
	case emodel.EnrollmentAccepted:
		m.StudentClassEnrollmentStatus = emodel.EnrollmentAccepted
		if body.AcceptedAt != nil {
			m.StudentClassEnrollmentAcceptedAt = body.AcceptedAt
		} else {
			m.StudentClassEnrollmentAcceptedAt = nowPtr()
		}
	case emodel.EnrollmentWaitlisted:
		m.StudentClassEnrollmentStatus = emodel.EnrollmentWaitlisted
		if body.WaitlistedAt != nil {
			m.StudentClassEnrollmentWaitlistedAt = body.WaitlistedAt
		} else {
			m.StudentClassEnrollmentWaitlistedAt = nowPtr()
		}
	case emodel.EnrollmentRejected:
		m.StudentClassEnrollmentStatus = emodel.EnrollmentRejected
		if body.RejectedAt != nil {
			m.StudentClassEnrollmentRejectedAt = body.RejectedAt
		} else {
			m.StudentClassEnrollmentRejectedAt = nowPtr()
		}
	case emodel.EnrollmentCanceled:
		m.StudentClassEnrollmentStatus = emodel.EnrollmentCanceled
		if body.CanceledAt != nil {
			m.StudentClassEnrollmentCanceledAt = body.CanceledAt
		} else {
			m.StudentClassEnrollmentCanceledAt = nowPtr()
		}
	default:
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid status")
	}

	if err := ctl.DB.WithContext(c.Context()).Save(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.JsonUpdated(c, "status updated", dto.FromModelStudentClassEnrollment(&m))
}

// PATCH /:school_id/class-enrollments/:id/payment
func (ctl *StudentClassEnrollmentController) AssignPayment(c *fiber.Ctx) error {
	schoolID, err := helperAuth.ParseSchoolIDFromPath(c)
	if err != nil {
		return err
	}
	if er := helperAuth.EnsureMemberSchool(c, schoolID); er != nil {
		return er
	}
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var body dto.AssignEnrollmentPaymentRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid body")
	}

	var m emodel.StudentClassEnrollmentModel
	tx := ctl.DB.WithContext(c.Context()).
		Where("student_class_enrollments_school_id = ?", schoolID).
		First(&m, "student_class_enrollments_id = ?", id)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, tx.Error.Error())
	}

	// assign payment
	m.StudentClassEnrollmentPaymentID = &body.StudentClassEnrollmentPaymentID
	if body.StudentClassEnrollmentPaymentSnapshot != nil {
		if b, er := json.Marshal(body.StudentClassEnrollmentPaymentSnapshot); er == nil {
			m.StudentClassEnrollmentPaymentSnapshot = b
		}
	}

	if err := ctl.DB.WithContext(c.Context()).Save(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.JsonUpdated(c, "payment assigned", dto.FromModelStudentClassEnrollment(&m))
}

// DELETE /:school_id/class-enrollments/:id
func (ctl *StudentClassEnrollmentController) Delete(c *fiber.Ctx) error {
	schoolID, err := helperAuth.ParseSchoolIDFromPath(c)
	if err != nil {
		return err
	}
	if er := helperAuth.EnsureMemberSchool(c, schoolID); er != nil {
		return er
	}
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var m emodel.StudentClassEnrollmentModel
	tx := ctl.DB.WithContext(c.Context()).
		Where("student_class_enrollments_school_id = ?", schoolID).
		First(&m, "student_class_enrollments_id = ?", id)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, tx.Error.Error())
	}

	if err := ctl.DB.WithContext(c.Context()).Delete(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.JsonDeleted(c, "deleted", fiber.Map{"student_class_enrollments_id": id})
}
