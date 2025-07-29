package controller

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"masjidku_backend/internals/features/donations/donations/dto"
	"masjidku_backend/internals/features/donations/donations/model"

	helpers "masjidku_backend/internals/helpers"
)

type DonationLikeController struct {
	DB *gorm.DB
}

func NewDonationLikeController(db *gorm.DB) *DonationLikeController {
	return &DonationLikeController{DB: db}
}


func (ctl *DonationLikeController) ToggleDonationLike(c *fiber.Ctx) error {
	var input dto.CreateOrToggleDonationLikeDTO
	if err := c.BodyParser(&input); err != nil {
		return helpers.Error(c, fiber.StatusBadRequest, "Invalid request body")
	}

	// Ambil dan validasi user_id dari context
	userIDStr, ok := c.Locals("user_id").(string)
	if !ok || userIDStr == "" {
		return helpers.Error(c, fiber.StatusUnauthorized, "Unauthorized user")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return helpers.Error(c, fiber.StatusBadRequest, "Invalid user_id format")
	}

	// Ambil dan validasi slug
	slug := c.Params("slug")
	if slug == "" {
		return helpers.Error(c, fiber.StatusBadRequest, "Slug masjid wajib diisi")
	}

	// Query masjid_id dari slug
	var masjidIDStr string
	if err := ctl.DB.
		Table("masjids").
		Select("masjid_id").
		Where("masjid_slug = ?", slug).
		Scan(&masjidIDStr).Error; err != nil || masjidIDStr == "" {
		return helpers.Error(c, fiber.StatusNotFound, "Masjid tidak ditemukan")
	}

	masjidID, err := uuid.Parse(masjidIDStr)
	if err != nil {
		return helpers.Error(c, fiber.StatusInternalServerError, "Masjid ID tidak valid")
	}

	// Cek apakah like sudah ada
	var like model.DonationLikeModel
	if err := ctl.DB.
		Where("donation_like_donation_id = ? AND donation_like_user_id = ?", input.DonationLikeDonationID, userID).
		First(&like).Error; err != nil {
		
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Buat baru
			newLike := model.DonationLikeModel{
				DonationLikeDonationID: input.DonationLikeDonationID,
				DonationLikeUserID:     userID,
				DonationLikeMasjidID:   &masjidID,
				DonationLikeIsLiked:    true,
				DonationLikeUpdatedAt:  time.Now(),
			}

			if err := ctl.DB.Create(&newLike).Error; err != nil {
				return helpers.Error(c, fiber.StatusInternalServerError, "Gagal menyimpan like")
			}
			return helpers.Success(c, "Berhasil menyukai donasi", newLike)
		}

		return helpers.Error(c, fiber.StatusInternalServerError, "Gagal cek like")
	}

	// Toggle existing like
	like.DonationLikeIsLiked = !like.DonationLikeIsLiked
	like.DonationLikeUpdatedAt = time.Now()
	like.DonationLikeMasjidID = &masjidID

	if err := ctl.DB.Save(&like).Error; err != nil {
		return helpers.Error(c, fiber.StatusInternalServerError, "Gagal toggle like")
	}

	return helpers.Success(c, "Berhasil toggle like", like)
}




// GET /public/donation-likes/count/:donation_id
func (ctl *DonationLikeController) GetDonationLikeCount(c *fiber.Ctx) error {
	donationID := c.Params("donation_id")
	if donationID == "" {
		return helpers.Error(c, fiber.StatusBadRequest, "donation_id wajib diisi")
	}

	var count int64
	if err := ctl.DB.Model(&model.DonationLikeModel{}).
		Where("donation_like_donation_id = ? AND donation_like_is_liked = true", donationID).
		Count(&count).Error; err != nil {
		return helpers.Error(c, fiber.StatusInternalServerError, "Gagal menghitung like")
	}

	return helpers.Success(c, "Jumlah like berhasil diambil", fiber.Map{
		"donation_id": donationID,
		"like_count":  count,
	})
}

// GET /public/donation-likes/is-liked/:donation_id
func (ctl *DonationLikeController) IsDonationLikedByUser(c *fiber.Ctx) error {
	donationID := c.Params("donation_id")
	if donationID == "" {
		return helpers.Error(c, fiber.StatusBadRequest, "donation_id wajib diisi")
	}

	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return helpers.Error(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	var like model.DonationLikeModel
	err := ctl.DB.Where("donation_like_donation_id = ? AND donation_like_user_id = ? AND donation_like_is_liked = true", donationID, userID).First(&like).Error

	if err == gorm.ErrRecordNotFound {
		return helpers.Success(c, "Belum menyukai donasi", fiber.Map{
			"is_liked": false,
		})
	} else if err != nil {
		return helpers.Error(c, fiber.StatusInternalServerError, "Gagal cek status like")
	}

	return helpers.Success(c, "Donasi sudah disukai", fiber.Map{
		"is_liked": true,
	})
}
