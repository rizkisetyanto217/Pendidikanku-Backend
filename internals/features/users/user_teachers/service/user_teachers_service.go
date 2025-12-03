package service

import (
	"context"
	"errors"
	"strings"
	"time"

	userTeacherModel "madinahsalam_backend/internals/features/users/user_teachers/model"
	userModel "madinahsalam_backend/internals/features/users/users/model"
	userProfileService "madinahsalam_backend/internals/features/users/users/service"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// helper: trim + truncate
func copyPtrTrunc(src *string, max int) *string {
	if src == nil {
		return nil
	}
	s := strings.TrimSpace(*src)
	if s == "" {
		return nil
	}
	if max > 0 && len(s) > max {
		s = s[:max]
	}
	return &s
}

// helper: gender enum → *string untuk user_teacher_gender
func genderToStringPtr(g *userModel.Gender) *string {
	if g == nil {
		return nil
	}
	v := strings.TrimSpace(string(*g))
	if v == "" {
		return nil
	}
	return &v
}

// helper: urutan prioritas nama untuk snapshot pengajar
func deriveTeacherName(u *userModel.UserModel, prof *userModel.UserProfileModel) string {
	// 1) full_name_cache dari profile
	if prof != nil && prof.UserProfileFullNameCache != nil {
		if n := strings.TrimSpace(*prof.UserProfileFullNameCache); n != "" {
			return n
		}
	}
	// 2) FullName dari users
	if u.FullName != nil {
		if n := strings.TrimSpace(*u.FullName); n != "" {
			return n
		}
	}
	// 3) user_name dari users
	if n := strings.TrimSpace(u.UserName); n != "" {
		return n
	}
	// 4) fallback aman
	return "Pengajar"
}

// EnsureUserTeacherFromUser:
// - Pastikan user_profiles ada (via EnsureProfileRow)
// - Jika user_teachers belum ada → buat dari data users + user_profiles
// - Kalau sudah ada → return yang existing (idempotent)
func EnsureUserTeacherFromUser(
	ctx context.Context,
	tx *gorm.DB,
	user *userModel.UserModel,
) (*userTeacherModel.UserTeacherModel, error) {
	if tx == nil {
		return nil, errors.New("db not ready")
	}
	if user == nil || user.ID == uuid.Nil {
		return nil, errors.New("user is required")
	}

	// 1) pastikan user_profiles ada
	if err := userProfileService.EnsureProfileRow(ctx, tx, user.ID, user.FullName); err != nil {
		return nil, err
	}

	// 2) cek apakah sudah ada user_teacher untuk user ini
	var ut userTeacherModel.UserTeacherModel
	err := tx.WithContext(ctx).
		Where("user_teacher_user_id = ? AND user_teacher_deleted_at IS NULL", user.ID).
		First(&ut).Error
	if err == nil {
		// sudah ada → boleh sync nama jika mau, tapi minimal return saja
		return &ut, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// 3) load user_profile untuk prefill field
	var prof userModel.UserProfileModel
	_ = tx.WithContext(ctx).
		Where("user_profile_user_id = ? AND user_profile_deleted_at IS NULL", user.ID).
		First(&prof).Error

	nameSnap := deriveTeacherName(user, &prof)
	if len(nameSnap) > 100 {
		nameSnap = nameSnap[:100]
	}

	now := time.Now()

	newUT := userTeacherModel.UserTeacherModel{
		UserTeacherID:            uuid.New(),
		UserTeacherUserID:        user.ID,
		UserTeacherFullNameCache: nameSnap,

		// ringkas/bio dari profile
		UserTeacherShortBio: copyPtrTrunc(prof.UserProfileBio, 300),

		// demografis
		UserTeacherGender:   genderToStringPtr(prof.UserProfileGender),
		UserTeacherLocation: copyPtrTrunc(prof.UserProfileLocation, 100),
		UserTeacherCity:     copyPtrTrunc(prof.UserProfileCity, 100),

		// sosial
		UserTeacherWhatsappURL:      prof.UserProfileWhatsappURL,
		UserTeacherInstagramURL:     prof.UserProfileInstagramURL,
		UserTeacherYoutubeURL:       prof.UserProfileYoutubeURL,
		UserTeacherLinkedinURL:      prof.UserProfileLinkedinURL,
		UserTeacherGithubURL:        prof.UserProfileGithubURL,
		UserTeacherTelegramUsername: prof.UserProfileTelegramUsername,

		// avatar – copy dari profile
		UserTeacherAvatarURL:       prof.UserProfileAvatarURL,
		UserTeacherAvatarObjectKey: prof.UserProfileAvatarObjectKey,

		// status default
		UserTeacherIsVerified:  false,
		UserTeacherIsActive:    true,
		UserTeacherIsCompleted: false,

		UserTeacherCreatedAt: now,
		UserTeacherUpdatedAt: now,
	}

	if err := tx.WithContext(ctx).Create(&newUT).Error; err != nil {
		return nil, err
	}

	return &newUT, nil
}
