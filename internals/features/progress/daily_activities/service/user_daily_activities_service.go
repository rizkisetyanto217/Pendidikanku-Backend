package service

import (
	"time"

	"masjidku_backend/internals/features/progress/daily_activities/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func UpdateOrInsertDailyActivity(db *gorm.DB, userID uuid.UUID) error {
	today := time.Now().Truncate(24 * time.Hour)
	var existing model.UserDailyActivity

	// Cek apakah sudah ada aktivitas hari ini berdasarkan activity_date
	err := db.Where("user_daily_activity_user_id = ? AND user_daily_activity_activity_date = ?", userID, today).
		First(&existing).Error
	if err == nil {
		// Sudah ada: hanya update waktu update
		return db.Model(&existing).Update("updated_at", time.Now()).Error
	}

	// Ambil aktivitas terakhir user (jika ada)
	var lastActivity model.UserDailyActivity
	err = db.
		Where("user_daily_activity_user_id = ?", userID).
		Order("user_daily_activity_activity_date DESC").
		First(&lastActivity).Error

	var newAmountDay int
	if err == nil && lastActivity.UserDailyActivityActivityDate.Add(24*time.Hour).Equal(today) {
		// Lanjutan dari kemarin (streak)
		newAmountDay = lastActivity.UserDailyActivityAmountDay + 1
	} else {
		// Hari pertama atau streak putus
		newAmountDay = 1
	}

	// Buat entri baru
	newActivity := model.UserDailyActivity{
		UserDailyActivityUserID:       userID,
		UserDailyActivityActivityDate: today,
		UserDailyActivityAmountDay:    newAmountDay,
	}

	return db.Create(&newActivity).Error
}
