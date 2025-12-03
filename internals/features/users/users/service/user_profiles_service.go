package service

import (
	"context"
	"errors"
	"strings"
	"time"

	userModel "madinahsalam_backend/internals/features/users/users/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// EnsureProfileRow memastikan 1 user punya 1 baris user_profiles (idempotent).
// Kalau sudah ada, bisa update snapshot nama kalau dikirim.
func EnsureProfileRow(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
	fullName *string,
) error {
	if tx == nil {
		return errors.New("db not ready")
	}
	if userID == uuid.Nil {
		return errors.New("user_id is required")
	}

	ctxQ, cancel := context.WithTimeout(ctx, 800*time.Millisecond)
	defer cancel()

	var prof userModel.UserProfileModel
	err := tx.WithContext(ctxQ).
		Where("user_profile_user_id = ? AND user_profile_deleted_at IS NULL", userID).
		First(&prof).Error

	// ==== CASE 1: sudah ada profile → optional update snapshot ====
	if err == nil {
		if fullName == nil {
			return nil
		}
		name := strings.TrimSpace(*fullName)
		if name == "" {
			return nil
		}
		if len(name) > 100 {
			name = name[:100]
		}

		// update hanya kalau beda
		cur := ""
		if prof.UserProfileFullNameCache != nil {
			cur = strings.TrimSpace(*prof.UserProfileFullNameCache)
		}
		if cur == name {
			return nil
		}

		now := time.Now()
		prof.UserProfileFullNameCache = &name
		prof.UserProfileUpdatedAt = now

		return tx.WithContext(ctxQ).
			Model(&userModel.UserProfileModel{}).
			Where("user_profile_id = ?", prof.UserProfileID).
			Updates(map[string]any{
				"user_profile_full_name_cache": prof.UserProfileFullNameCache,
				"user_profile_updated_at":      prof.UserProfileUpdatedAt,
			}).Error
	}

	// ==== CASE 2: error selain not found ====
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	// ==== CASE 3: belum ada → buat baru ====
	now := time.Now()
	var nameSnap *string
	if fullName != nil {
		n := strings.TrimSpace(*fullName)
		if n != "" {
			if len(n) > 100 {
				n = n[:100]
			}
			nameSnap = &n
		}
	}

	newProf := userModel.UserProfileModel{
		// ID biarkan pakai default gen_random_uuid() dari DB,
		UserProfileUserID:        userID,
		UserProfileFullNameCache: nameSnap,

		UserProfileIsPublicProfile: true,
		UserProfileIsVerified:      false,
		UserProfileIsCompleted:     false,

		UserProfileCreatedAt: now,
		UserProfileUpdatedAt: now,
	}

	return tx.WithContext(ctxQ).Create(&newProf).Error
}
