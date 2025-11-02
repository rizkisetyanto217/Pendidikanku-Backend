package controller

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	dto "schoolku_backend/internals/features/school/submissions_assesments/submissions/dto"
	model "schoolku_backend/internals/features/school/submissions_assesments/submissions/model"

	helper "schoolku_backend/internals/helpers"
	helperAuth "schoolku_backend/internals/helpers/auth"
)

// GET /:id (READ — member; student hanya boleh lihat miliknya)
func (ctrl *SubmissionController) List(c *fiber.Ctx) error {
	c.Locals("DB", ctrl.DB)

	// 1) Resolve school context (slug/id)
	mc, err := helperAuth.ResolveSchoolContext(c)
	if err != nil {
		return err
	}
	var mid uuid.UUID
	if mc.ID != uuid.Nil {
		mid = mc.ID
	} else if s := strings.TrimSpace(mc.Slug); s != "" {
		id, er := helperAuth.GetSchoolIDBySlug(c, s)
		if er != nil || id == uuid.Nil {
			return fiber.NewError(fiber.StatusNotFound, "School (slug) tidak ditemukan")
		}
		mid = id
	} else {
		return helperAuth.ErrSchoolContextMissing
	}

	// 2) Authorize minimal member school
	if err := helperAuth.EnsureMemberSchool(c, mid); err != nil {
		return err
	}

	// 3) Parse param :id
	subID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil || subID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "submission id tidak valid")
	}

	// 4) Load submission milik tenant ini
	var row model.Submission
	if err := ctrl.DB.WithContext(c.Context()).
		Where(`
			submission_id = ?
			AND submission_school_id = ?
			AND submission_deleted_at IS NULL
		`, subID, mid).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Submission tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// 5) Student hanya boleh akses submission miliknya
	if helperAuth.IsStudent(c) && !helperAuth.IsDKM(c) && !helperAuth.IsTeacher(c) {
		if sid, _ := helperAuth.GetSchoolStudentIDForSchool(c, mid); sid == uuid.Nil || sid != row.SubmissionStudentID {
			return helper.JsonError(c, fiber.StatusForbidden, "Anda tidak diizinkan melihat submission ini")
		}
	}

	// 6) Response (+optional URLs jika ?with_urls=1/true/yes)
	if truthy(c.Query("with_urls")) {
		var urls []model.SubmissionURLModel
		_ = ctrl.DB.WithContext(c.Context()).
			Where(`
				submission_url_submission_id = ?
				AND submission_url_school_id = ?
				AND submission_url_deleted_at IS NULL
			`, row.SubmissionID, mid).
			Order("submission_url_is_primary DESC, submission_url_order ASC, submission_url_created_at ASC").
			Find(&urls)

		return helper.JsonOK(c, "OK", fiber.Map{
			"submission": dto.FromModel(&row),
			"urls":       urls, // kalau mau pakai DTO URL, bisa map ke dto.SubmissionURLItem di sini
		})
	}

	return helper.JsonOK(c, "OK", dto.FromModel(&row))
}

// helper kecil buat query bool
func truthy(s string) bool {
	v := strings.ToLower(strings.TrimSpace(s))
	return v == "1" || v == "true" || v == "yes"
}
