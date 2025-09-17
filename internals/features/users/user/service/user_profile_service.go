package service

import (
	"context"
	"log"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	profilemodel "masjidku_backend/internals/features/users/user/model"
)

func EnsureProfileRow(ctx context.Context, db *gorm.DB, userID uuid.UUID) error {
    log.Printf("[EnsureProfileRow] called for user_id=%s", userID)

    p := profilemodel.UserProfileModel{UserProfileUserID: userID}
    err := db.WithContext(ctx).
        Clauses(clause.OnConflict{
            Columns:   []clause.Column{{Name: "user_profile_user_id"}},
            DoNothing: true,
        }).
        Create(&p).Error

    if err != nil {
        log.Printf("[EnsureProfileRow] ERROR: %v", err)
    } else {
        log.Printf("[EnsureProfileRow] DONE: inserted or skipped (user_id=%s, profile_id=%s)", userID, p.UserProfileID)
    }
    return err
}
