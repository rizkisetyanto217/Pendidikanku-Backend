// file: internals/features/users/users/service/user_profile_student_sync.go
package service

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	studentModel "madinahsalam_backend/internals/features/lembaga/school_yayasans/teachers_students/model"
	userModel "madinahsalam_backend/internals/features/users/users/model"
)

/*
Input minimal yang kita butuh dari user_profiles
(biar nggak import model user_profile ke sini → menghindari circular import)
*/
type UserProfileSnapshotInput struct {
	UserID            uuid.UUID // user_profile_user_id → FK ke users.id
	UserProfileID     uuid.UUID // user_profile_id
	FullNameCache     *string   // user_profile_full_name_cache
	UserNameCache     *string   // user_profile_user_name_cache  🆕
	AvatarURL         *string   // user_profile_avatar_url
	WhatsappURL       *string   // user_profile_whatsapp_url
	ParentName        *string   // user_profile_parent_name
	ParentWhatsappURL *string   // user_profile_parent_whatsapp_url
	Gender            *string   // user_profile_gender (string, bukan enum)
}

// helper: trim string kosong → nil
func nz(p *string) *string {
	if p == nil {
		return nil
	}
	v := strings.TrimSpace(*p)
	if v == "" {
		return nil
	}
	return &v
}

// SyncSchoolStudentsAndEnrollmentsFromUserProfile
// - Step 0: update users.full_name + users.user_name dari snapshot profile
// - Step 1: update cache di school_students
// - Step 2: propagate ke student_class_enrollments via school_students
// - Step 3: propagate ke student_class_sections via school_students
// - Step 4: propagate ke student_class_section_subject_teachers (student_csst) via school_students
func SyncSchoolStudentsAndEnrollmentsFromUserProfile(
	ctx context.Context,
	db *gorm.DB,
	in UserProfileSnapshotInput,
	now time.Time,
) error {
	// Nama "display" yang dipakai di snapshot-snapshot lain
	name := ""
	if s := nz(in.FullNameCache); s != nil {
		name = *s
	}

	// =========================================
	// STEP 0: UPDATE users.full_name & users.user_name
	// =========================================
	fullNameVal := nz(in.FullNameCache) // boleh nil → akan di-set NULL
	userNameVal := nz(in.UserNameCache) // boleh nil → kalau nil, tidak mengubah user_name

	userUpdates := map[string]any{
		"updated_at": now,
	}

	// full_name: kalau nil → benar-benar di-SET NULL
	if fullNameVal != nil {
		userUpdates["full_name"] = *fullNameVal
	} else {
		userUpdates["full_name"] = nil
	}

	// user_name: hanya di-update kalau ada nilai baru non-empty
	if userNameVal != nil {
		userUpdates["user_name"] = *userNameVal
	}

	resUser := db.WithContext(ctx).
		Model(&userModel.UserModel{}).
		Where("id = ?", in.UserID).
		Where("deleted_at IS NULL").
		Updates(userUpdates)

	if resUser.Error != nil {
		log.Printf("[SyncSchoolStudentsAndEnrollmentsFromUserProfile] update users error: %v", resUser.Error)
		return resUser.Error
	}

	if resUser.RowsAffected == 0 {
		log.Printf("[SyncSchoolStudentsAndEnrollmentsFromUserProfile] no users row for user_id=%s (skip users sync)", in.UserID)
	}

	// =========================================
	// STEP 1: UPDATE school_students
	// =========================================
	stuUpdates := map[string]any{
		"school_student_user_profile_name_cache":                name,
		"school_student_user_profile_avatar_url_cache":          nz(in.AvatarURL),
		"school_student_user_profile_whatsapp_url_cache":        nz(in.WhatsappURL),
		"school_student_user_profile_parent_name_cache":         nz(in.ParentName),
		"school_student_user_profile_parent_whatsapp_url_cache": nz(in.ParentWhatsappURL),
		"school_student_user_profile_gender_cache":              nz(in.Gender),
		"school_student_updated_at":                             now,
	}

	res := db.WithContext(ctx).
		Model(&studentModel.SchoolStudentModel{}).
		Where("school_student_user_profile_id = ?", in.UserProfileID).
		Updates(stuUpdates)
	if res.Error != nil {
		log.Printf("[SyncSchoolStudentsAndEnrollmentsFromUserProfile] update school_students error: %v", res.Error)
		return res.Error
	}

	// Kalau tidak ada student yang terkait, ya cukup update users aja
	if res.RowsAffected == 0 {
		log.Printf("[SyncSchoolStudentsAndEnrollmentsFromUserProfile] no school_students for user_profile_id=%s", in.UserProfileID)
		return nil
	}

	// =========================================
	// STEP 2: UPDATE student_class_enrollments
	// =========================================
	if err := db.WithContext(ctx).Exec(`
		UPDATE student_class_enrollments e
		SET
			student_class_enrollments_user_profile_name_cache = s.school_student_user_profile_name_cache,
			student_class_enrollments_user_profile_avatar_url_cache = s.school_student_user_profile_avatar_url_cache,
			student_class_enrollments_user_profile_whatsapp_url_cache = s.school_student_user_profile_whatsapp_url_cache,
			student_class_enrollments_user_profile_parent_name_cache = s.school_student_user_profile_parent_name_cache,
			student_class_enrollments_user_profile_parent_whatsapp_url_cache = s.school_student_user_profile_parent_whatsapp_url_cache,
			student_class_enrollments_user_profile_gender_cache = s.school_student_user_profile_gender_cache,
			student_class_enrollments_updated_at = $2
		FROM school_students s
		WHERE s.school_student_id = e.student_class_enrollments_school_student_id
		  AND s.school_student_user_profile_id = $1
		  AND s.school_student_deleted_at IS NULL
	`, in.UserProfileID, now).Error; err != nil {
		log.Printf("[SyncSchoolStudentsAndEnrollmentsFromUserProfile] update enrollments error: %v", err)
		return err
	}

	// =========================================
	// STEP 3: UPDATE student_class_sections
	// =========================================
	if err := db.WithContext(ctx).Exec(`
		UPDATE student_class_sections sc
		SET
			student_class_section_user_profile_name_cache = s.school_student_user_profile_name_cache,
			student_class_section_user_profile_avatar_url_cache = s.school_student_user_profile_avatar_url_cache,
			student_class_section_user_profile_whatsapp_url_cache = s.school_student_user_profile_whatsapp_url_cache,
			student_class_section_user_profile_parent_name_cache = s.school_student_user_profile_parent_name_cache,
			student_class_section_user_profile_parent_whatsapp_url_cache = s.school_student_user_profile_parent_whatsapp_url_cache,
			student_class_section_user_profile_gender_cache = s.school_student_user_profile_gender_cache,
			student_class_section_updated_at = $2
		FROM school_students s
		WHERE s.school_student_id = sc.student_class_section_school_student_id
		  AND s.school_student_user_profile_id = $1
		  AND s.school_student_deleted_at IS NULL
		  AND sc.student_class_section_deleted_at IS NULL
	`, in.UserProfileID, now).Error; err != nil {
		log.Printf("[SyncSchoolStudentsAndEnrollmentsFromUserProfile] update student_class_sections error: %v", err)
		return err
	}

	// =========================================
	// STEP 4: UPDATE student_class_section_subject_teachers (student_csst)
	// =========================================
	if err := db.WithContext(ctx).Exec(`
		UPDATE student_class_section_subject_teachers link
		SET
			student_csst_user_profile_name_cache = s.school_student_user_profile_name_cache,
			student_csst_user_profile_avatar_url_cache = s.school_student_user_profile_avatar_url_cache,
			student_csst_user_profile_wa_url_cache = s.school_student_user_profile_whatsapp_url_cache,
			student_csst_user_profile_parent_name_cache = s.school_student_user_profile_parent_name_cache,
			student_csst_user_profile_parent_wa_url_cache = s.school_student_user_profile_parent_whatsapp_url_cache,
			student_csst_user_profile_gender_cache = s.school_student_user_profile_gender_cache,
			student_csst_updated_at = $2
		FROM school_students s
		WHERE s.school_student_id = link.student_csst_student_id
		  AND s.school_student_user_profile_id = $1
		  AND s.school_student_deleted_at IS NULL
		  AND link.student_csst_deleted_at IS NULL
	`, in.UserProfileID, now).Error; err != nil {
		log.Printf("[SyncSchoolStudentsAndEnrollmentsFromUserProfile] update student_csst error: %v", err)
		return err
	}

	return nil
}

// Helper supaya controller cukup kirim model utuh
func SyncFromUserProfileModel(
	ctx context.Context,
	db *gorm.DB,
	p userModel.UserProfileModel,
	now time.Time,
) error {
	var genderStr *string
	if p.UserProfileGender != nil {
		g := string(*p.UserProfileGender)
		genderStr = &g
	}

	return SyncSchoolStudentsAndEnrollmentsFromUserProfile(
		ctx,
		db,
		UserProfileSnapshotInput{
			UserID:            p.UserProfileUserID,
			UserProfileID:     p.UserProfileID,
			FullNameCache:     p.UserProfileFullNameCache,
			UserNameCache:     p.UserProfileUserNameCache, // 🆕 ikut dikirim
			AvatarURL:         p.UserProfileAvatarURL,
			WhatsappURL:       p.UserProfileWhatsappURL,
			ParentName:        p.UserProfileParentName,
			ParentWhatsappURL: p.UserProfileParentWhatsappURL,
			Gender:            genderStr,
		},
		now,
	)
}
