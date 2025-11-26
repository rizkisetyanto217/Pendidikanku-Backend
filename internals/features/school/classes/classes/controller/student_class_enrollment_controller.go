// file: internals/features/school/classes/class_enrollments/controller/student_class_enrollment_controller.go
package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	enrollDTO "madinahsalam_backend/internals/features/school/classes/classes/dto"
	emodel "madinahsalam_backend/internals/features/school/classes/classes/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
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
		case emodel.ClassEnrollmentInitiated,
			emodel.ClassEnrollmentPendingReview,
			emodel.ClassEnrollmentAwaitingPayment,
			emodel.ClassEnrollmentAccepted,
			emodel.ClassEnrollmentWaitlisted,
			emodel.ClassEnrollmentRejected,
			emodel.ClassEnrollmentCanceled:
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

// Enrich: isi username + nama student + class_name dari tabel lain
func enrichEnrollmentExtras(
	ctx context.Context,
	db *gorm.DB,
	schoolID uuid.UUID,
	items []enrollDTO.StudentClassEnrollmentResponse,
) {
	if len(items) == 0 {
		return
	}

	// Kumpulkan ID unik
	stuIDsSet := map[uuid.UUID]struct{}{}
	classIDsSet := map[uuid.UUID]struct{}{}
	for _, it := range items {
		stuIDsSet[it.StudentClassEnrollmentSchoolStudentID] = struct{}{}

		// ⬇️ pakai ClassID (field baru di DTO)
		classIDsSet[it.ClassID] = struct{}{}
	}

	stuIDs := make([]uuid.UUID, 0, len(stuIDsSet))
	classIDs := make([]uuid.UUID, 0, len(classIDsSet))
	for id := range stuIDsSet {
		stuIDs = append(stuIDs, id)
	}
	for id := range classIDsSet {
		classIDs = append(classIDs, id)
	}

	// ===== Ambil students (pakai snapshot user_profile) =====
	type stuRow struct {
		ID            uuid.UUID `gorm:"column:school_student_id"`
		UserProfileID uuid.UUID `gorm:"column:school_student_user_profile_id"`
		NameSnapshot  *string   `gorm:"column:school_student_user_profile_name_snapshot"`
	}

	stuMap := make(map[uuid.UUID]stuRow, len(stuIDs))
	if len(stuIDs) > 0 {
		var stus []stuRow
		_ = db.WithContext(ctx).
			Table("school_students").
			Select(`
				school_student_id,
				school_student_user_profile_id,
				school_student_user_profile_name_snapshot
			`).
			Where("school_student_school_id = ? AND school_student_id IN ?", schoolID, stuIDs).
			Find(&stus).Error

		for _, s := range stus {
			stuMap[s.ID] = s
		}
	}

	// ===== Ambil classes (class_name) =====
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

		for _, cRow := range clss {
			clsMap[cRow.ID] = cRow
		}
	}

	// ===== Ambil user_id dari user_profiles =====
	profileIDsSet := map[uuid.UUID]struct{}{}
	for _, s := range stuMap {
		if s.UserProfileID != uuid.Nil {
			profileIDsSet[s.UserProfileID] = struct{}{}
		}
	}

	profileIDs := make([]uuid.UUID, 0, len(profileIDsSet))
	for id := range profileIDsSet {
		profileIDs = append(profileIDs, id)
	}

	type profileRow struct {
		ID     uuid.UUID `gorm:"column:user_profile_id"`
		UserID uuid.UUID `gorm:"column:user_profile_user_id"`
	}

	profileUserMap := make(map[uuid.UUID]uuid.UUID, len(profileIDs)) // key: user_profile_id → user_id
	if len(profileIDs) > 0 {
		var prows []profileRow
		_ = db.WithContext(ctx).
			Table("user_profiles").
			Select("user_profile_id, user_profile_user_id").
			Where("user_profile_id IN ?", profileIDs).
			Where("user_profile_deleted_at IS NULL").
			Find(&prows).Error

		for _, pr := range prows {
			profileUserMap[pr.ID] = pr.UserID
		}
	}

	// ===== Ambil usernames (batch) =====
	userIDsSet := map[uuid.UUID]struct{}{}
	for _, uid := range profileUserMap {
		if uid != uuid.Nil {
			userIDsSet[uid] = struct{}{}
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
			Where("deleted_at IS NULL").
			Find(&us).Error

		for _, u := range us {
			uMap[u.ID] = u.UserName
		}
	}

	// ===== Isi ke items (DTO) =====
	for i := range items {
		if s, ok := stuMap[items[i].StudentClassEnrollmentSchoolStudentID]; ok {
			// override convenience name dengan nama snapshot terbaru
			if s.NameSnapshot != nil && *s.NameSnapshot != "" {
				items[i].StudentClassEnrollmentStudentName = *s.NameSnapshot
			}

			// isi username kalau bisa dilacak sampai users
			if s.UserProfileID != uuid.Nil {
				if uid, ok2 := profileUserMap[s.UserProfileID]; ok2 && uid != uuid.Nil {
					if un, ok3 := uMap[uid]; ok3 {
						username := un
						items[i].StudentClassEnrollmentUsername = &username
					}
				}
			}
		}

		// pakai ClassID (bukan field lama StudentClassEnrollmentClassID)
		if cRow, ok := clsMap[items[i].ClassID]; ok {
			items[i].StudentClassEnrollmentClassName = cRow.ClassName
		}
	}
}

/* =======================================================
   Routes (ADMIN/PMB — DKM/Admin only)
======================================================= */

// POST /:school_id/class-enrollments
func (ctl *StudentClassEnrollmentController) Create(c *fiber.Ctx) error {
	// tenant
	schoolID, err := helperAuth.ParseSchoolIDFromPath(c)
	if err != nil {
		return err
	}
	// ⬇️ DKM/Admin only
	if er := helperAuth.EnsureDKMSchool(c, schoolID); er != nil {
		return er
	}

	// body
	var body enrollDTO.CreateStudentClassEnrollmentRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid body")
	}

	// ===== Lookup student (pastikan belong to school) =====
	var stu struct {
		ID           uuid.UUID `gorm:"column:school_student_id"`
		Slug         string    `gorm:"column:school_student_slug"`
		Code         *string   `gorm:"column:school_student_code"`
		NameSnapshot *string   `gorm:"column:school_student_user_profile_name_snapshot"`
	}

	if err := ctl.DB.WithContext(c.Context()).
		Table("school_students").
		Select(`
			school_student_id,
			school_student_slug,
			school_student_code,
			school_student_user_profile_name_snapshot
		`).
		Where("school_student_id = ? AND school_student_school_id = ?", body.SchoolStudentID, schoolID).
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

	// Normalisasi nilai snapshot student
	studentName := ""
	if stu.NameSnapshot != nil {
		studentName = *stu.NameSnapshot
	}

	// ===== Build model & isi SNAPSHOT =====
	m := emodel.StudentClassEnrollmentModel{
		StudentClassEnrollmentsSchoolID:        schoolID,
		StudentClassEnrollmentsSchoolStudentID: body.SchoolStudentID,
		StudentClassEnrollmentsClassID:         body.ClassID,

		StudentClassEnrollmentsStatus:      emodel.ClassEnrollmentInitiated,
		StudentClassEnrollmentsTotalDueIDR: body.TotalDueIDR,
		StudentClassEnrollmentsAppliedAt:   time.Now(),

		// snapshots
		StudentClassEnrollmentsClassNameSnapshot:       cls.ClassName,
		StudentClassEnrollmentsClassSlugSnapshot:       &cls.Slug,
		StudentClassEnrollmentsUserProfileNameSnapshot: studentName,
		StudentClassEnrollmentsStudentCodeSnapshot:     stu.Code,
		StudentClassEnrollmentsStudentSlugSnapshot:     &stu.Slug,
	}

	if body.Preferences != nil {
		if b, er := json.Marshal(body.Preferences); er == nil {
			m.StudentClassEnrollmentsPreferences = datatypes.JSON(b)
		}
	}

	if err := ctl.DB.WithContext(c.Context()).Create(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// response
	resp := enrollDTO.FromModelStudentClassEnrollment(&m)
	list := []enrollDTO.StudentClassEnrollmentResponse{resp}
	enrichEnrollmentExtras(c.Context(), ctl.DB, schoolID, list)
	return helper.JsonCreated(c, "created", list[0])
}

// PATCH /:school_id/class-enrollments/:id
func (ctl *StudentClassEnrollmentController) Update(c *fiber.Ctx) error {
	schoolID, err := helperAuth.ParseSchoolIDFromPath(c)
	if err != nil {
		return err
	}
	// ⬇️ DKM/Admin only
	if er := helperAuth.EnsureDKMSchool(c, schoolID); er != nil {
		return er
	}
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var body enrollDTO.UpdateStudentClassEnrollmentRequest
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
		m.StudentClassEnrollmentsTotalDueIDR = *body.TotalDueIDR
	}
	if body.Preferences != nil {
		if b, er := json.Marshal(body.Preferences); er == nil {
			m.StudentClassEnrollmentsPreferences = datatypes.JSON(b)
		}
	}

	if err := ctl.DB.WithContext(c.Context()).Save(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.JsonUpdated(c, "updated", enrollDTO.FromModelStudentClassEnrollment(&m))
}

// PATCH /:school_id/class-enrollments/:id/status
func (ctl *StudentClassEnrollmentController) UpdateStatus(c *fiber.Ctx) error {
	schoolID, err := helperAuth.ParseSchoolIDFromPath(c)
	if err != nil {
		return err
	}
	// ⬇️ DKM/Admin only
	if er := helperAuth.EnsureDKMSchool(c, schoolID); er != nil {
		return er
	}
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var body enrollDTO.UpdateStudentClassEnrollmentStatusRequest
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
	case emodel.ClassEnrollmentInitiated:
		m.StudentClassEnrollmentsStatus = emodel.ClassEnrollmentInitiated
		if m.StudentClassEnrollmentsAppliedAt.IsZero() {
			m.StudentClassEnrollmentsAppliedAt = time.Now()
		}
	case emodel.ClassEnrollmentPendingReview:
		m.StudentClassEnrollmentsStatus = emodel.ClassEnrollmentPendingReview
		if body.ReviewedAt != nil {
			m.StudentClassEnrollmentsReviewedAt = body.ReviewedAt
		} else if m.StudentClassEnrollmentsReviewedAt == nil {
			m.StudentClassEnrollmentsReviewedAt = nowPtr()
		}
	case emodel.ClassEnrollmentAwaitingPayment:
		m.StudentClassEnrollmentsStatus = emodel.ClassEnrollmentAwaitingPayment
		if m.StudentClassEnrollmentsReviewedAt == nil {
			m.StudentClassEnrollmentsReviewedAt = nowPtr()
		}
	case emodel.ClassEnrollmentAccepted:
		m.StudentClassEnrollmentsStatus = emodel.ClassEnrollmentAccepted
		if body.AcceptedAt != nil {
			m.StudentClassEnrollmentsAcceptedAt = body.AcceptedAt
		} else {
			m.StudentClassEnrollmentsAcceptedAt = nowPtr()
		}
	case emodel.ClassEnrollmentWaitlisted:
		m.StudentClassEnrollmentsStatus = emodel.ClassEnrollmentWaitlisted
		if body.WaitlistedAt != nil {
			m.StudentClassEnrollmentsWaitlistedAt = body.WaitlistedAt
		} else {
			m.StudentClassEnrollmentsWaitlistedAt = nowPtr()
		}
	case emodel.ClassEnrollmentRejected:
		m.StudentClassEnrollmentsStatus = emodel.ClassEnrollmentRejected
		if body.RejectedAt != nil {
			m.StudentClassEnrollmentsRejectedAt = body.RejectedAt
		} else {
			m.StudentClassEnrollmentsRejectedAt = nowPtr()
		}
	case emodel.ClassEnrollmentCanceled:
		m.StudentClassEnrollmentsStatus = emodel.ClassEnrollmentCanceled
		if body.CanceledAt != nil {
			m.StudentClassEnrollmentsCanceledAt = body.CanceledAt
		} else {
			m.StudentClassEnrollmentsCanceledAt = nowPtr()
		}
	default:
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid status")
	}

	if err := ctl.DB.WithContext(c.Context()).Save(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.JsonUpdated(c, "status updated", enrollDTO.FromModelStudentClassEnrollment(&m))
}

// PATCH /:school_id/class-enrollments/:id/payment
func (ctl *StudentClassEnrollmentController) AssignPayment(c *fiber.Ctx) error {
	schoolID, err := helperAuth.ParseSchoolIDFromPath(c)
	if err != nil {
		return err
	}
	// ⬇️ DKM/Admin only
	if er := helperAuth.EnsureDKMSchool(c, schoolID); er != nil {
		return er
	}
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var body enrollDTO.AssignEnrollmentPaymentRequest
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
	m.StudentClassEnrollmentsPaymentID = &body.StudentClassEnrollmentPaymentID
	if body.StudentClassEnrollmentPaymentSnapshot != nil {
		if b, er := json.Marshal(body.StudentClassEnrollmentPaymentSnapshot); er == nil {
			m.StudentClassEnrollmentsPaymentSnapshot = datatypes.JSON(b)
		}
	}

	if err := ctl.DB.WithContext(c.Context()).Save(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.JsonUpdated(c, "payment assigned", enrollDTO.FromModelStudentClassEnrollment(&m))
}

// DELETE /:school_id/class-enrollments/:id
func (ctl *StudentClassEnrollmentController) Delete(c *fiber.Ctx) error {
	schoolID, err := helperAuth.ParseSchoolIDFromPath(c)
	if err != nil {
		return err
	}
	// ⬇️ DKM/Admin only
	if er := helperAuth.EnsureDKMSchool(c, schoolID); er != nil {
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
