package controller

import (
	"log"
	"strconv"
	"time"

	"masjidku_backend/internals/constants"
	certificateModel "masjidku_backend/internals/features/masjids/certificate/model" // jika dipakai di tempat lain
	"masjidku_backend/internals/features/masjids/lectures/exams/dto"
	"masjidku_backend/internals/features/masjids/lectures/exams/model"
	helper "masjidku_backend/internals/helpers"

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

// ➕ POST /api/u/user-lecture-exams
func (ctrl *UserLectureExamController) CreateUserLectureExam(c *fiber.Ctx) error {
	var body dto.CreateUserLectureExamRequest
	if err := c.BodyParser(&body); err != nil {
		log.Printf("[ERROR] Failed to parse request body: %v", err)
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid request body")
	}
	log.Printf("[DEBUG] Payload diterima: %+v", body)

	// Validasi minimal: butuh user_id atau user_name
	if body.UserLectureExamUserID == nil && body.UserLectureExamUserName == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Wajib isi user_id atau user_name")
	}

	// Resolve masjid_id dari slug
	var masjid struct {
		MasjidID string
	}
	if err := ctrl.DB.
		Table("masjids").
		Select("masjid_id").
		Where("masjid_slug = ?", body.UserLectureExamMasjidSlug).
		Scan(&masjid).Error; err != nil || masjid.MasjidID == "" {
		log.Printf("[ERROR] Masjid slug tidak ditemukan: %v", err)
		return helper.JsonError(c, fiber.StatusBadRequest, "Masjid tidak ditemukan")
	}
	parsedMasjidID, err := uuid.Parse(masjid.MasjidID)
	if err != nil {
		log.Printf("[ERROR] Gagal parse masjid_id: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Masjid ID invalid")
	}

	// Simpan hasil exam user
	newExam := model.UserLectureExamModel{
		UserLectureExamGrade:    body.UserLectureExamGrade,
		UserLectureExamExamID:   body.UserLectureExamExamID,
		UserLectureExamUserID:   body.UserLectureExamUserID,
		UserLectureExamMasjidID: parsedMasjidID,
		UserLectureExamUserName: body.UserLectureExamUserName,
	}
	if err := ctrl.DB.Create(&newExam).Error; err != nil {
		log.Printf("[ERROR] Gagal simpan exam: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal simpan data ujian")
	}
	log.Println("[INFO] Progress ujian berhasil disimpan")

	// -------- Sertifikat (best-effort) --------
	// Ambil lecture_id dari exam; jika gagal, lewati proses sertifikat
	var lectureIDStr string
	if err := ctrl.DB.
		Table("lecture_exams").
		Select("lecture_exam_lecture_id").
		Where("lecture_exam_id = ?", body.UserLectureExamExamID).
		Scan(&lectureIDStr).Error; err == nil && lectureIDStr != "" {

		if lectureID, err := uuid.Parse(lectureIDStr); err == nil {
			// Cari sertifikat berdasarkan lecture_id
			var cert certificateModel.CertificateModel
			if err := ctrl.DB.Where("certificate_lecture_id = ?", lectureID).First(&cert).Error; err == nil {
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
					// Gunakan real user_id bila ada; jika tidak, dummy
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
			} else {
				log.Printf("[INFO] Sertifikat tidak ditemukan untuk lecture_id: %v", lectureID)
			}
		} else {
			log.Printf("[INFO] Gagal parse lecture_id: %v", err)
		}
	} else {
		log.Printf("[INFO] Gagal ambil lecture_id dari exam atau kosong")
	}

	// Response
	return helper.JsonCreated(c, "User lecture exam created successfully", dto.ToUserLectureExamDTO(newExam))
}

func toIntPointer(f *float64) *int {
	if f == nil {
		return nil
	}
	v := int(*f)
	return &v
}

// 📄 GET /api/a/user-lecture-exams (support pagination)
func (ctrl *UserLectureExamController) GetAllUserLectureExams(c *fiber.Ctx) error {
	// pagination ringan
	page, _ := strconv.Atoi(c.Query("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int64
	if err := ctrl.DB.Model(&model.UserLectureExamModel{}).Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to count exams")
	}

	var records []model.UserLectureExamModel
	if err := ctrl.DB.
		Order("user_lecture_exam_created_at DESC").
		Limit(pageSize).Offset(offset).
		Find(&records).Error; err != nil {
		log.Printf("[ERROR] Failed to fetch exams: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to fetch exam data")
	}

	resp := make([]dto.UserLectureExamDTO, len(records))
	for i, r := range records {
		resp[i] = dto.ToUserLectureExamDTO(r)
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	pagination := fiber.Map{
		"page":        page,
		"page_size":   pageSize,
		"total":       total,
		"total_pages": totalPages,
		"has_next":    page < totalPages,
	}

	return helper.JsonList(c, resp, pagination)
}

// 🔍 GET /api/u/user-lecture-exams/:id
func (ctrl *UserLectureExamController) GetUserLectureExamByID(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID is required")
	}

	var record model.UserLectureExamModel
	if err := ctrl.DB.First(&record, "user_lecture_exam_id = ?", id).Error; err != nil {
		log.Printf("[ERROR] Exam not found: %v", err)
		return helper.JsonError(c, fiber.StatusNotFound, "Exam record not found")
	}

	return helper.JsonOK(c, "Exam record fetched successfully", dto.ToUserLectureExamDTO(record))
}
