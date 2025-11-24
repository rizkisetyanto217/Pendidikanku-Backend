package controller

import (
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	tsModel "madinahsalam_backend/internals/features/users/user_teachers/model"
	userModel "madinahsalam_backend/internals/features/users/users/model"
	helper "madinahsalam_backend/internals/helpers"
	helpersAuth "madinahsalam_backend/internals/helpers/auth"
)

type ProfileCompletionStatus struct {
	HasProfile         bool `json:"has_profile"`
	IsProfileCompleted bool `json:"is_profile_completed"`

	HasTeacher         bool `json:"has_teacher"`
	IsTeacherCompleted bool `json:"is_teacher_completed"`

	// untuk frontend:
	// - kalau user adalah guru di school ini → butuh profile + teacher completed
	// - kalau bukan guru → cukup profile completed
	IsFullyCompleted bool `json:"is_fully_completed"`
}

// GET /api/:school_slug/auth/me/profile-completion
func (ctl *AuthController) GetMyProfileCompletion(c *fiber.Ctx) error {
	// 1) Ambil user_id dari token via helperAuth
	userID, err := helpersAuth.GetUserIDFromToken(c)
	if err != nil {
		// konsisten pakai JsonError
		return helper.JsonError(c, http.StatusUnauthorized, "Unauthorized")
	}

	// 2) Ambil school_id aktif (slug/token/active-school) via helperAuth
	schoolID, err := helpersAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		// ResolveSchoolIDFromContext sudah return error siap kirim (JsonError)
		return err
	}

	// 3) Cek apakah user ini punya peran guru di school ini
	isTeacherInSchool := helpersAuth.IsTeacherInSchool(c, schoolID)

	var (
		profile userModel.UserProfileModel
		teacher tsModel.UserTeacherModel
	)

	// ==== Cek user_profiles ====
	errProf := ctl.DB.
		Where("user_profile_user_id = ?", userID).
		First(&profile).Error

	hasProfile := false
	isProfileCompleted := false

	if errProf != nil {
		if !errors.Is(errProf, gorm.ErrRecordNotFound) {
			// error DB lain → 500
			return helper.JsonError(c, fiber.StatusInternalServerError, errProf.Error())
		}
	} else {
		hasProfile = true
		isProfileCompleted = profile.UserProfileIsCompleted
	}

	// ==== Cek user_teachers (global per user, bukan per school) ====
	errTeach := ctl.DB.
		Where("user_teacher_user_id = ?", userID).
		First(&teacher).Error

	hasTeacher := false
	isTeacherCompleted := false

	if errTeach != nil {
		if !errors.Is(errTeach, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusInternalServerError, errTeach.Error())
		}
	} else {
		hasTeacher = true
		isTeacherCompleted = teacher.UserTeacherIsCompleted
	}

	// ==== Hitung flag final ====
	var isFully bool
	if isTeacherInSchool {
		// mode GURU di sekolah ini: wajib profile + teacher completed
		isFully = isProfileCompleted && isTeacherCompleted
	} else {
		// mode non-guru (murid / user biasa di sekolah ini): cukup profile completed
		isFully = isProfileCompleted
	}

	resp := ProfileCompletionStatus{
		HasProfile:         hasProfile,
		IsProfileCompleted: isProfileCompleted,
		HasTeacher:         hasTeacher,
		IsTeacherCompleted: isTeacherCompleted,
		IsFullyCompleted:   isFully,
	}

	// pakai helper.JsonOK yang sudah ada
	return helper.JsonOK(c, "profile completion status", resp)
}
