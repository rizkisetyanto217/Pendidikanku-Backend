package controller

import (
	"log"
	"masjidku_backend/internals/constants"
	certificateModel "masjidku_backend/internals/features/masjids/certificate/model"
	"masjidku_backend/internals/features/masjids/lectures/exams/dto"
	"masjidku_backend/internals/features/masjids/lectures/exams/model"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserLectureExamController struct {
	DB *gorm.DB
}

func NewUserLectureExamController(db *gorm.DB) *UserLectureExamController {
	return &UserLectureExamController{DB: db}
}


func (ctrl *UserLectureExamController) CreateUserLectureExam(c *fiber.Ctx) error {
	var body dto.CreateUserLectureExamRequest
	if err := c.BodyParser(&body); err != nil {
		log.Printf("[ERROR] Failed to parse request body: %v", err)
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}
	log.Printf("[DEBUG] Payload diterima: %+v", body)

	// Validasi minimal
	if body.UserLectureExamUserID == nil && body.UserLectureExamUserName == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Wajib isi user ID atau nama user")
	}

	// Ambil masjid_id dari slug
	var masjid struct {
		MasjidID string
	}
	if err := ctrl.DB.
		Table("masjids").
		Select("masjid_id").
		Where("masjid_slug = ?", body.UserLectureExamMasjidSlug).
		Scan(&masjid).Error; err != nil || masjid.MasjidID == "" {
		log.Printf("[ERROR] Masjid slug tidak ditemukan: %v", err)
		return fiber.NewError(fiber.StatusBadRequest, "Masjid tidak ditemukan")
	}
	log.Printf("[DEBUG] Masjid ditemukan, ID: %s", masjid.MasjidID)

	parsedMasjidID, err := uuid.Parse(masjid.MasjidID)
	if err != nil {
		log.Printf("[ERROR] Gagal parse masjid_id: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Masjid ID invalid")
	}

	// Simpan data ke user_lecture_exams
	newExam := model.UserLectureExamModel{
		UserLectureExamGrade:    body.UserLectureExamGrade,
		UserLectureExamExamID:   body.UserLectureExamExamID,
		UserLectureExamUserID:   body.UserLectureExamUserID,
		UserLectureExamMasjidID: parsedMasjidID,
		UserLectureExamUserName: body.UserLectureExamUserName,
	}
	if err := ctrl.DB.Create(&newExam).Error; err != nil {
		log.Printf("[ERROR] Gagal simpan exam: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal simpan data ujian")
	}
	log.Println("[INFO] Progress ujian berhasil disimpan")

	// Ambil lecture_id dari exam
	var lectureIDStr string
	if err := ctrl.DB.
		Table("lecture_exams").
		Select("lecture_exam_lecture_id").
		Where("lecture_exam_id = ?", body.UserLectureExamExamID).
		Scan(&lectureIDStr).Error; err != nil {
		log.Printf("[INFO] Gagal ambil lecture_id dari exam: %v", err)
		return c.Status(fiber.StatusCreated).JSON(dto.ToUserLectureExamDTO(newExam))
	}
	lectureID, err := uuid.Parse(lectureIDStr)
	if err != nil {
		log.Printf("[INFO] Gagal parse lecture_id: %v", err)
		return c.Status(fiber.StatusCreated).JSON(dto.ToUserLectureExamDTO(newExam))
	}

	// Cari sertifikat berdasarkan lecture_id
	var cert certificateModel.CertificateModel
	if err := ctrl.DB.Where("certificate_lecture_id = ?", lectureID).First(&cert).Error; err != nil {
		log.Printf("[INFO] Sertifikat tidak ditemukan untuk lecture_id: %v", lectureID)
		return c.Status(fiber.StatusCreated).JSON(dto.ToUserLectureExamDTO(newExam))
	}

	// Cek kelulusan
	if newExam.UserLectureExamGrade != nil && *newExam.UserLectureExamGrade >= 70 {
		log.Printf("[INFO] User lulus ujian dengan nilai %v", *newExam.UserLectureExamGrade)

		userCert := certificateModel.UserCertificateModel{
			UserCertCertificateID: cert.CertificateID,
			UserCertScore:         toIntPointer(newExam.UserLectureExamGrade),
			UserCertSlugURL:       uuid.New().String(),
			UserCertIsUpToDate:    true,
			UserCertIssuedAt:      time.Now(),
		}

		// Tetap isi user_id: pakai real ID jika tersedia, kalau tidak dummy
		if body.UserLectureExamUserID != nil {
			userCert.UserCertUserID = *body.UserLectureExamUserID
			log.Printf("[INFO] Sertifikat dikaitkan dengan user_id: %v", userCert.UserCertUserID)
		} else {
			userCert.UserCertUserID = constants.DummyUserID
			log.Println("[INFO] Sertifikat dikaitkan dengan user_id dummy (non-login user)")
		}


		if err := ctrl.DB.Create(&userCert).Error; err != nil {
			log.Printf("[ERROR] Gagal buat sertifikat: %v", err)
		} else {
			log.Printf("[SUCCESS] Sertifikat berhasil dibuat: %s", userCert.UserCertSlugURL)
		}
	} else {
		log.Printf("[INFO] Nilai tidak mencukupi atau kosong: %v", newExam.UserLectureExamGrade)
	}

	return c.Status(fiber.StatusCreated).JSON(dto.ToUserLectureExamDTO(newExam))
}



func toIntPointer(f *float64) *int {
	if f == nil {
		return nil
	}
	v := int(*f)
	return &v
}


// GET - Lihat semua hasil exam user
func (ctrl *UserLectureExamController) GetAllUserLectureExams(c *fiber.Ctx) error {
	var records []model.UserLectureExamModel
	if err := ctrl.DB.Find(&records).Error; err != nil {
		log.Printf("[ERROR] Failed to fetch exams: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch exam data")
	}

	var response []dto.UserLectureExamDTO
	for _, r := range records {
		response = append(response, dto.ToUserLectureExamDTO(r))
	}

	return c.JSON(response)
}

// GET - Detail hasil exam user berdasarkan ID
func (ctrl *UserLectureExamController) GetUserLectureExamByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var record model.UserLectureExamModel
	if err := ctrl.DB.First(&record, "user_lecture_exam_id = ?", id).Error; err != nil {
		log.Printf("[ERROR] Exam not found: %v", err)
		return fiber.NewError(fiber.StatusNotFound, "Exam record not found")
	}

	return c.JSON(dto.ToUserLectureExamDTO(record))
}
