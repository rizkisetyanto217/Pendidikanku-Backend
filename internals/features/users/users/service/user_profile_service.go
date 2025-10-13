// service/user_profile_service.go
package service

import (
	"context"
	"log"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	profilemodel "masjidku_backend/internals/features/users/users/model"
)

// service/user_profile_service.go
func EnsureProfileRow(ctx context.Context, db *gorm.DB, userID uuid.UUID, fullName *string) error {
	log.Printf("[EnsureProfileRow] called for user_id=%s", userID)

	var namePtr *string
	if fullName != nil {
		if t := strings.TrimSpace(*fullName); t != "" {
			namePtr = &t
		}
	}

	p := profilemodel.UserProfileModel{
		UserProfileUserID:           userID,
		UserProfileFullNameSnapshot: namePtr, // nilai untuk INSERT (EXCLUDED)
	}

	err := db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_profile_user_id"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				// NB: kwalifikasikan kedua operand untuk menghindari ambiguity
				"user_profile_full_name_snapshot": gorm.Expr(
					`COALESCE("user_profiles"."user_profile_full_name_snapshot", EXCLUDED."user_profile_full_name_snapshot")`,
				),
			}),
		}).
		Create(&p).Error

	if err != nil {
		log.Printf("[EnsureProfileRow] ERROR: %v", err)
	} else {
		log.Printf("[EnsureProfileRow] DONE: inserted/updated (user_id=%s, profile_id=%s)", userID, p.UserProfileID)
	}
	return err
}
