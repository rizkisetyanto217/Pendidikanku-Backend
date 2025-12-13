// file: internals/features/school/academics/classes/controller/class_controller.go
package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"mime/multipart"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	// Services & helpers
	"madinahsalam_backend/internals/features/lembaga/stats/lembaga_stats/service"
	academicTermsService "madinahsalam_backend/internals/features/school/academics/academic_terms/service"

	// ✅ pakai DTO & model classes yang baru (academics)
	dto "madinahsalam_backend/internals/features/school/classes/classes/dto"
	classmodel "madinahsalam_backend/internals/features/school/classes/classes/model"

	// class_sections & class_parent tetap di modul lama
	classSectionDto "madinahsalam_backend/internals/features/school/classes/class_sections/dto"
	classSectionModel "madinahsalam_backend/internals/features/school/classes/class_sections/model"
	classParentSnapshot "madinahsalam_backend/internals/features/school/classes/classes/service"

	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"
	dbtime "madinahsalam_backend/internals/helpers/dbtime"
	helperOSS "madinahsalam_backend/internals/helpers/oss"

	csModel "madinahsalam_backend/internals/features/school/academics/subjects/model"
	csstDto "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/dto"
	csstModel "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/model"
)

/* ================= Controller & Constructor ================= */

type ClassController struct {
	DB *gorm.DB
}

func NewClassController(db *gorm.DB) *ClassController {
	return &ClassController{DB: db}
}

var validate = validator.New()

/* ================= Helpers kecil ================= */

func ptrToStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func coalesceStr(a, b string) string {
	a = strings.TrimSpace(a)
	if a != "" {
		return a
	}
	return strings.TrimSpace(b)
}

func slugifySafe(s string, maxLen int) string {
	return helper.Slugify(strings.TrimSpace(s), maxLen)
}

// ClassName di model bertipe *string → bungkus string jadi *string (kosong → nil)
func strPtrOrNil(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

/* ================= Slug builder (parent + optional term) ================= */

func buildClassBaseSlug(
	ctx context.Context,
	db *gorm.DB,
	schoolID uuid.UUID,
	classParentID uuid.UUID,
	academicTermID *uuid.UUID,
	explicitBase string,
	maxLen int,
) (string, error) {
	// 0) Jika explicitBase ada → pakai langsung
	if s := strings.TrimSpace(explicitBase); s != "" {
		b := helper.Slugify(s, maxLen)
		if b == "" {
			b = "kelas"
		}
		log.Printf("[CLASSES][SLUG] explicit base='%s' → '%s'", s, b)
		return b, nil
	}

	// 1) Ambil data class_parent (pakai row kecil agar loose-coupling)
	type parentRow struct {
		Slug *string `gorm:"column:class_parent_slug"`
		Name string  `gorm:"column:class_parent_name"`
	}
	var pr parentRow
	if err := db.WithContext(ctx).
		Table("class_parents").
		Select("class_parent_slug, class_parent_name").
		Where("class_parent_id = ? AND class_parent_school_id = ? AND class_parent_deleted_at IS NULL",
			classParentID, schoolID).
		Take(&pr).Error; err != nil {
		return "", fmt.Errorf("class_parent tidak ditemukan / db error: %w", err)
	}
	rawParent := coalesceStr(ptrToStr(pr.Slug), pr.Name)
	parentPart := strings.TrimSpace(helper.Slugify(rawParent, maxLen))
	log.Printf("[CLASSES][SLUG] parent: slug_db=%v name_db='%s' → parentPart='%s'",
		pr.Slug, pr.Name, parentPart)

	// 2) Ambil data academic_term (jika ada)
	termPart := ""
	if academicTermID != nil && *academicTermID != uuid.Nil {
		type termRow struct {
			Slug  *string `gorm:"column:academic_term_slug"`
			Year  string  `gorm:"column:academic_term_academic_year"`
			TName string  `gorm:"column:academic_term_name"`
		}
		var tr termRow
		if err := db.WithContext(ctx).
			Table("academic_terms").
			Select("academic_term_slug, academic_term_academic_year, academic_term_name").
			Where("academic_term_id = ? AND academic_term_school_id = ? AND academic_term_deleted_at IS NULL",
				*academicTermID, schoolID).
			Take(&tr).Error; err == nil {
			if s := strings.TrimSpace(ptrToStr(tr.Slug)); s != "" {
				termPart = helper.Slugify(s, maxLen)
			} else {
				termPart = helper.Slugify(strings.TrimSpace(tr.Year+" "+tr.TName), maxLen)
			}
			log.Printf("[CLASSES][SLUG] term: slug_db=%v year='%s' name='%s' → termPart='%s'",
				tr.Slug, tr.Year, tr.TName, termPart)
		} else {
			log.Printf("[CLASSES][SLUG] term fetch error (ignored, lanjut tanpa term): %v", err)
		}
	} else {
		log.Printf("[CLASSES][SLUG] no academic term (nil)")
	}

	// 3) Gabungkan parentPart + termPart
	base := parentPart
	if termPart != "" {
		if base != "" {
			base += "-" + termPart
		} else {
			base = termPart
		}
	}
	if base == "" {
		base = "kelas"
	}
	base = helper.Slugify(base, maxLen)
	log.Printf("[CLASSES][SLUG] baseSlug='%s'", base)

	return base, nil
}

/* =========================== CREATE =========================== */

// POST /admin/classes
func (ctrl *ClassController) CreateClass(c *fiber.Ctx) error {
	start := time.Now()
	log.Printf("[CLASSES][CREATE] ▶️ incoming request")

	/* ---- Resolve School Context via helper ---- */
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		// ResolveSchoolIDFromContext sudah balikin JsonError yang rapi
		return err
	}
	log.Printf("[CLASSES][CREATE] 🕌 school_id=%s (from context)", schoolID)

	// 🔒 Hanya DKM/Admin yang boleh bikin kelas
	if err := helperAuth.EnsureDKMSchool(c, schoolID); err != nil {
		log.Printf("[CLASSES][CREATE] ❌ ensure DKM school failed: %v", err)
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusForbidden, "Hanya DKM/Admin yang diizinkan untuk mengelola kelas")
	}

	/* ---- Parse request & paksa tenant ---- */
	var req dto.CreateClassRequest
	if err := c.BodyParser(&req); err != nil {
		log.Printf("[CLASSES][CREATE] ❌ body parse error: %v", err)
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	req.ClassSchoolID = schoolID
	req.Normalize()
	log.Printf("[CLASSES][CREATE] 📩 req: parent_id=%s term_id=%v delivery=%v status=%v slug_in='%s'",
		req.ClassClassParentID, req.ClassAcademicTermID, req.ClassDeliveryMode, req.ClassStatus, req.ClassSlug)

	// 🔽 Tambahan: baca class_sections dari form-data (string JSON)
	if rawSections := strings.TrimSpace(c.FormValue("class_sections")); rawSections != "" {
		log.Printf("[CLASSES][CREATE] 🧾 raw class_sections=%s", rawSections)

		var sections []dto.CreateClassSectionInlineRequest
		if err := json.Unmarshal([]byte(rawSections), &sections); err != nil {
			log.Printf("[CLASSES][CREATE] ❌ parse class_sections JSON error: %v", err)
			return helper.JsonError(c, fiber.StatusBadRequest, "Format class_sections tidak valid (harus JSON array)")
		}
		req.ClassSections = sections
		log.Printf("[CLASSES][CREATE] ✅ parsed %d class_section(s) from form-data", len(req.ClassSections))
	} else {
		log.Printf("[CLASSES][CREATE] ℹ️ no class_sections in form-data")
	}

	/* ---- Validasi ---- */
	if err := req.Validate(); err != nil {
		log.Printf("[CLASSES][CREATE] ❌ req validate error: %v", err)
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	if err := validate.Struct(req); err != nil {
		log.Printf("[CLASSES][CREATE] ❌ struct validate error: %v", err)
		return helper.JsonError(c, fiber.StatusBadRequest, "Validasi data kelas gagal: "+err.Error())
	}

	/* ---- Bentuk model awal ---- */
	m := req.ToModel()
	log.Printf("[CLASSES][CREATE] 🔧 model init: parent_id=%s term_id=%v status=%s",
		m.ClassClassParentID, m.ClassAcademicTermID, m.ClassStatus)

	// 🔹 simpan section yang berhasil dibuat (untuk response)
	createdSections := make([]classSectionModel.ClassSectionModel, 0)
	// 🔹 simpan CSST yang berhasil dibuat (untuk response)
	createdCSSTs := make([]csstModel.ClassSectionSubjectTeacherModel, 0)

	/* ---- TX ---- */
	tx := ctrl.DB.WithContext(c.Context()).Begin()
	if tx.Error != nil {
		log.Printf("[CLASSES][CREATE] ❌ begin tx error: %v", tx.Error)
		return helper.JsonError(c, fiber.StatusInternalServerError, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback().Error
			log.Printf("[CLASSES][CREATE] 💥 panic recovered: %+v", r)
			panic(r)
		}
	}()

	/* ---- Slug komposit (parent + term) → CI-unique per school ---- */
	baseSlug, err := buildClassBaseSlug(
		c.Context(), tx, schoolID,
		m.ClassClassParentID,
		m.ClassAcademicTermID, // boleh nil
		"",                    // paksa komposit
		160,
	)
	if err != nil {
		_ = tx.Rollback().Error
		log.Printf("[CLASSES][CREATE] ❌ build base slug error: %v", err)
		return helper.JsonError(c, fiber.StatusBadRequest, "Gagal membentuk slug dasar: "+err.Error())
	}

	uniqueSlug, err := helper.EnsureUniqueSlugCI(
		c.Context(), tx,
		"classes", "class_slug",
		baseSlug,
		func(q *gorm.DB) *gorm.DB {
			return q.Where("class_school_id = ? AND class_deleted_at IS NULL", schoolID)
		},
		160,
	)
	if err != nil {
		_ = tx.Rollback().Error
		log.Printf("[CLASSES][CREATE] ❌ ensure unique slug error: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
	}
	m.ClassSlug = uniqueSlug
	log.Printf("[CLASSES][CREATE] ✅ unique_slug='%s'", m.ClassSlug)

	/* ---- SNAPSHOT (parent + term) ---- */
	if err := classParentSnapshot.HydrateClassParentSnapshot(c.Context(), tx, schoolID, m); err != nil {
		_ = tx.Rollback().Error
		log.Printf("[CLASSES][CREATE] ❌ parent snapshot error: %v", err)
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil snapshot parent: "+err.Error())
	}
	if err := academicTermsService.HydrateAcademicTermCache(c.Context(), tx, schoolID, m); err != nil {
		_ = tx.Rollback().Error
		log.Printf("[CLASSES][CREATE] ❌ term cache hydrate error: %v", err)

		// kalau academic_term_id tidak valid / term tidak ada
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusBadRequest, "Academic term tidak ditemukan")
		}

		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil cache term: "+err.Error())
	}

	/* ---- class_name (gabungan parent + term) ---- */
	parent := ""
	if m.ClassClassParentNameCache != nil {
		parent = *m.ClassClassParentNameCache
	}
	m.ClassName = strPtrOrNil(dto.ComposeClassNameSpace(parent, m.ClassAcademicTermNameCache))

	/* ---- Insert ---- */
	if err := tx.Create(m).Error; err != nil {
		_ = tx.Rollback().Error
		low := strings.ToLower(err.Error())
		log.Printf("[CLASSES][CREATE] ❌ insert error: %v", err)

		if strings.Contains(low, "uq_classes_slug_per_school_alive") ||
			(strings.Contains(low, "duplicate") && strings.Contains(low, "class_slug")) {
			return helper.JsonError(c, fiber.StatusConflict, "Slug sudah digunakan di school ini")
		}

		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat data kelas")
	}
	log.Printf("[CLASSES][CREATE] 💾 created class_id=%s", m.ClassID)

	/* ===================================================
	   AUTO CREATE CLASS SECTIONS / CSST (via query ?auto)
	   =================================================== */

	autoParam := strings.ToLower(strings.TrimSpace(c.Query("auto", "")))
	autoSection := autoParam == "class_section" || autoParam == "class_sections" || autoParam == "csst" || autoParam == "all"
	autoCSST := autoParam == "csst" || autoParam == "class_section_subject_teachers" || autoParam == "all"
	log.Printf("[CLASSES][CREATE] auto_param=%s | autoSection=%v autoCSST=%v | class_sections_len=%d",
		autoParam, autoSection, autoCSST, len(req.ClassSections))

	if autoSection && len(req.ClassSections) > 0 {
		log.Printf("[CLASSES][CREATE] 🧩 auto create %d class_section(s)", len(req.ClassSections))

		// parse angkatan term → int (kalau ada)
		var termAngkatanPtr *int
		if m.ClassAcademicTermAngkatanCache != nil {
			s := strings.TrimSpace(*m.ClassAcademicTermAngkatanCache)
			if s != "" {
				if v, err := strconv.Atoi(s); err == nil {
					tmp := v
					termAngkatanPtr = &tmp
				} else {
					log.Printf("[CLASSES][CREATE] ⚠️ gagal parse ClassAcademicTermAngkatanCache='%s' ke int: %v", s, err)
				}
			}
		}

		for idx, secReq := range req.ClassSections {
			name := strings.TrimSpace(secReq.Name)
			if name == "" {
				log.Printf("[CLASSES][CREATE] ⚠️ skip section idx=%d: empty name", idx)
				continue
			}

			// slug dasar: class_slug + nama section yang dislugify sederhana
			secSlugPart := strings.ToLower(name)
			secSlugPart = strings.ReplaceAll(secSlugPart, " ", "-")
			secSlugPart = strings.ReplaceAll(secSlugPart, "_", "-")

			baseSectionSlug := m.ClassSlug + "-" + secSlugPart

			uniqueSectionSlug, err := helper.EnsureUniqueSlugCI(
				c.Context(), tx,
				"class_sections", "class_section_slug",
				baseSectionSlug,
				func(q *gorm.DB) *gorm.DB {
					return q.Where("class_section_school_id = ? AND class_section_deleted_at IS NULL", schoolID)
				},
				160,
			)
			if err != nil {
				_ = tx.Rollback().Error
				log.Printf("[CLASSES][CREATE] ❌ ensure unique section slug error (idx=%d): %v", idx, err)
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghasilkan slug section unik")
			}

			codePtr := secReq.Code
			quotaPtr := secReq.QuotaTotal

			// 🔹 walikelas & asisten dari request (opsional)
			teacherID := secReq.SchoolTeacherID
			assistantID := secReq.AssistantSchoolTeacherID

			// ========= HANDLE IMAGE (URL atau multipart) =========
			var imageURLPtr *string
			var imageKeyPtr *string

			// 1) kalau sudah ada ImageURL di JSON, pakai itu
			if secReq.ImageURL != nil && strings.TrimSpace(*secReq.ImageURL) != "" {
				url := strings.TrimSpace(*secReq.ImageURL)
				imageURLPtr = &url

				if k, e := helperOSS.ExtractKeyFromPublicURL(url); e == nil {
					imageKeyPtr = &k
				} else if k2, e2 := helperOSS.KeyFromPublicURL(url); e2 == nil {
					imageKeyPtr = &k2
				}
			}

			// 2) kalau ada image_field, coba baca file dari multipart dan upload
			if secReq.ImageField != nil && strings.TrimSpace(*secReq.ImageField) != "" {
				field := strings.TrimSpace(*secReq.ImageField)
				if fh, err := c.FormFile(field); err == nil && fh != nil {
					log.Printf("[CLASSES][CREATE] 📤 uploading section image field=%s filename=%s size=%d (idx=%d)",
						field, fh.Filename, fh.Size, idx)

					svc, er := helperOSS.NewOSSServiceFromEnv("")
					if er == nil {
						ctx, cancel := context.WithTimeout(c.Context(), 45*time.Second)
						defer cancel()

						keyPrefix := fmt.Sprintf("schools/%s/class_sections", schoolID.String())
						if url, upErr := svc.UploadAsWebP(ctx, fh, keyPrefix); upErr == nil {
							imageURLPtr = &url

							if k, e := helperOSS.ExtractKeyFromPublicURL(url); e == nil {
								imageKeyPtr = &k
							} else if k2, e2 := helperOSS.KeyFromPublicURL(url); e2 == nil {
								imageKeyPtr = &k2
							}

							log.Printf("[CLASSES][CREATE] ✅ section image uploaded url=%s (idx=%d)", url, idx)
						} else {
							log.Printf("[CLASSES][CREATE] ❌ upload section image error (idx=%d): %v", idx, upErr)
						}
					} else {
						log.Printf("[CLASSES][CREATE] ❌ init OSS svc error for section image (idx=%d): %v", idx, er)
					}
				} else if err != nil {
					log.Printf("[CLASSES][CREATE] ⚠️ read section image field=%s error (idx=%d): %v", field, idx, err)
				}
			}

			sec := &classSectionModel.ClassSectionModel{
				ClassSectionSchoolID: schoolID,
				ClassSectionSlug:     uniqueSectionSlug,

				ClassSectionName:           name,
				ClassSectionCode:           codePtr,
				ClassSectionQuotaTotal:     quotaPtr,
				ClassSectionImageURL:       imageURLPtr,
				ClassSectionImageObjectKey: imageKeyPtr,

				// Link ke class
				ClassSectionClassID:        &m.ClassID,
				ClassSectionClassNameCache: m.ClassName,
				ClassSectionClassSlugCache: &m.ClassSlug,

				// Snapshot parent
				ClassSectionClassParentID:         &m.ClassClassParentID,
				ClassSectionClassParentNameCache:  m.ClassClassParentNameCache,
				ClassSectionClassParentSlugCache:  m.ClassClassParentSlugCache,
				ClassSectionClassParentLevelCache: m.ClassClassParentLevelCache,

				// Snapshot term
				ClassSectionAcademicTermID:                m.ClassAcademicTermID,
				ClassSectionAcademicTermNameCache:         m.ClassAcademicTermNameCache,
				ClassSectionAcademicTermSlugCache:         m.ClassAcademicTermSlugCache,
				ClassSectionAcademicTermAcademicYearCache: m.ClassAcademicTermAcademicYearCache,
				ClassSectionAcademicTermAngkatanCache:     termAngkatanPtr,

				// 🔹 Wali kelas & asisten
				ClassSectionSchoolTeacherID:          teacherID,
				ClassSectionAssistantSchoolTeacherID: assistantID,

				// ✅ Status awal: active (pakai enum)
				ClassSectionStatus: classSectionModel.ClassStatusActive,
			}

			if err := tx.Create(sec).Error; err != nil {
				_ = tx.Rollback().Error
				log.Printf("[CLASSES][CREATE] ❌ insert class_section error (idx=%d): %v", idx, err)
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat class_section default")
			}
			log.Printf("[CLASSES][CREATE] 💾 created class_section_id=%s slug=%s (idx=%d)", sec.ClassSectionID, sec.ClassSectionSlug, idx)

			// simpan ke list untuk dikirim di response
			createdSections = append(createdSections, *sec)
		}

	}

	// ===================================================
	//
	//	AUTO CREATE CSST (ClassSectionSubjectTeachers)
	//	- aktif kalau autoCSST == true
	//	- sumber: class_subjects untuk class_parent ini
	//
	// ===================================================
	if autoCSST && len(createdSections) > 0 {
		log.Printf("[CLASSES][CREATE] 🧠 auto create CSST for %d section(s)", len(createdSections))

		// 1) Ambil semua class_subjects untuk parent ini
		var classSubjects []csModel.ClassSubjectModel
		if err := tx.
			Where(`
			class_subject_school_id = ?
			AND class_subject_class_parent_id = ?
			AND class_subject_deleted_at IS NULL
			AND class_subject_is_active = TRUE
		`, schoolID, m.ClassClassParentID).
			Find(&classSubjects).Error; err != nil {

			_ = tx.Rollback().Error
			log.Printf("[CLASSES][CREATE] ❌ load class_subjects error: %v", err)
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil daftar mapel untuk parent ini")
		}

		log.Printf("[CLASSES][CREATE] 📚 found %d class_subject(s) for parent=%s", len(classSubjects), m.ClassClassParentID)
		if len(classSubjects) == 0 {
			log.Printf("[CLASSES][CREATE] ⚠️ no class_subjects found, skip CSST auto create")
		} else {
			// parse angkatan term → int lagi (scope baru)
			var termAngkatanPtr *int
			if m.ClassAcademicTermAngkatanCache != nil {
				s := strings.TrimSpace(*m.ClassAcademicTermAngkatanCache)
				if s != "" {
					if v, err := strconv.Atoi(s); err == nil {
						tmp := v
						termAngkatanPtr = &tmp
					} else {
						log.Printf("[CLASSES][CREATE] ⚠️ gagal parse ClassAcademicTermAngkatanCache='%s' ke int (for CSST): %v", s, err)
					}
				}
			}

			for _, sec := range createdSections {
				// 🔹 Sekarang: walaupun belum ada wali kelas, tetap bikin CSST
				var teacherID *uuid.UUID
				if sec.ClassSectionSchoolTeacherID != nil && *sec.ClassSectionSchoolTeacherID != uuid.Nil {
					teacherID = sec.ClassSectionSchoolTeacherID
				}
				assistantID := sec.ClassSectionAssistantSchoolTeacherID

				for _, cs := range classSubjects {
					// slug: section-slug + subject-slug
					secSlug := strings.TrimSpace(sec.ClassSectionSlug)

					subSlug := ""
					if cs.ClassSubjectSubjectSlugCache != nil {
						subSlug = strings.TrimSpace(*cs.ClassSubjectSubjectSlugCache)
					}

					if secSlug == "" {
						secSlug = fmt.Sprintf("section-%s", sec.ClassSectionID.String())
					}
					if subSlug == "" {
						subSlug = fmt.Sprintf("subject-%s", cs.ClassSubjectSubjectID.String())
					}

					rawSlug := strings.ToLower(secSlug + "-" + subSlug)
					rawSlug = strings.ReplaceAll(rawSlug, " ", "-")
					rawSlug = strings.ReplaceAll(rawSlug, "_", "-")

					uniqueCSSTSlug, err := helper.EnsureUniqueSlugCI(
						c.Context(), tx,
						"class_section_subject_teachers", "csst_slug",
						rawSlug,
						func(q *gorm.DB) *gorm.DB {
							return q.Where("csst_school_id = ? AND csst_deleted_at IS NULL", schoolID)
						},
						160,
					)
					if err != nil {
						_ = tx.Rollback().Error
						log.Printf("[CLASSES][CREATE] ❌ ensure unique csst slug error (section_id=%s subject_id=%s): %v",
							sec.ClassSectionID, cs.ClassSubjectSubjectID, err)
						return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghasilkan slug CSST unik")
					}

					// ambil beberapa cache dari section
					secName := sec.ClassSectionName
					secCode := sec.ClassSectionCode
					secSlugCache := sec.ClassSectionSlug

					secNamePtr := &secName
					secCodePtr := secCode
					secSlugPtr := &secSlugCache

					// room cache (kalau kamu pakai) – sekarang cuma ID saja, tanpa slug cache
					roomID := sec.ClassSectionClassRoomID

					// ================= DELIVERY MODE (from class) =================
					deliveryMode := csstModel.DeliveryModeOffline // default
					if m.ClassDeliveryMode != nil {
						dm := strings.TrimSpace(*m.ClassDeliveryMode)
						if dm != "" {
							deliveryMode = csstModel.ClassDeliveryMode(dm)
						}
					}

					// academic term caches (ambil dari class)
					acTermID := m.ClassAcademicTermID
					acTermName := m.ClassAcademicTermNameCache
					acTermSlug := m.ClassAcademicTermSlugCache
					acYear := m.ClassAcademicTermAcademicYearCache

					// min passing score dari class_subject:
					minPassing := cs.ClassSubjectMinPassingScore

					now := time.Now()

					csst := &csstModel.ClassSectionSubjectTeacherModel{
						// ===== PK & Tenant =====
						CSSTID:       uuid.New(),
						CSSTSchoolID: schoolID,

						// ===== Identitas & Fasilitas =====
						CSSTSlug:        &uniqueCSSTSlug,
						CSSTDescription: nil,
						CSSTGroupURL:    nil,

						// ===== Agregat & Quota =====
						CSSTTotalAttendance:       0,
						CSSTTotalMeetingsTarget:   nil,
						CSSTQuotaTotal:            nil,
						CSSTQuotaTaken:            0,
						CSSTTotalAssessments:      0,
						CSSTTotalAssessmentsTrain: 0,
						CSSTTotalAssessmentsDaily: 0,
						CSSTTotalAssessmentsExam:  0,
						CSSTTotalStudentsPassed:   0,

						// Delivery mode
						CSSTDeliveryMode:                   deliveryMode,
						CSSTSchoolAttendanceEntryModeCache: nil,

						// SECTION cache
						CSSTClassSectionID:        sec.ClassSectionID,
						CSSTClassSectionSlugCache: secSlugPtr,
						CSSTClassSectionNameCache: secNamePtr,
						CSSTClassSectionCodeCache: secCodePtr,
						CSSTClassSectionURLCache:  nil,

						// ROOM cache
						CSSTClassRoomID:        roomID,
						CSSTClassRoomSlugCache: nil,
						CSSTClassRoomCache:     nil,

						// PEOPLE cache — teacher boleh kosong (nil)
						CSSTSchoolTeacherID:        teacherID,
						CSSTSchoolTeacherSlugCache: nil,
						CSSTSchoolTeacherCache:     nil,

						CSSTAssistantSchoolTeacherID:        assistantID,
						CSSTAssistantSchoolTeacherSlugCache: nil,
						CSSTAssistantSchoolTeacherCache:     nil,

						// SUBJECT cache
						CSSTTotalBooks:       0,
						CSSTClassSubjectID:   cs.ClassSubjectID,
						CSSTSubjectID:        &cs.ClassSubjectSubjectID,
						CSSTSubjectNameCache: cs.ClassSubjectSubjectNameCache,
						CSSTSubjectCodeCache: cs.ClassSubjectSubjectCodeCache,
						CSSTSubjectSlugCache: cs.ClassSubjectSubjectSlugCache,

						// ACADEMIC TERM cache
						CSSTAcademicTermID:            acTermID,
						CSSTAcademicTermNameCache:     acTermName,
						CSSTAcademicTermSlugCache:     acTermSlug,
						CSSTAcademicYearCache:         acYear,
						CSSTAcademicTermAngkatanCache: termAngkatanPtr,

						// KKM cache per CSST
						CSSTMinPassingScoreClassSubjectCache: minPassing,

						// Status & audit
						CSSTStatus:      csstModel.ClassStatusActive,
						CSSTCreatedAt:   now,
						CSSTUpdatedAt:   now,
						CSSTCompletedAt: nil,
					}

					if err := tx.Create(csst).Error; err != nil {
						_ = tx.Rollback().Error
						log.Printf("[CLASSES][CREATE] ❌ insert CSST error (section_id=%s subject_id=%s): %v",
							sec.ClassSectionID, cs.ClassSubjectSubjectID, err)
						return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat pengajar mapel default (CSST)")
					}

					log.Printf("[CLASSES][CREATE] 💾 created CSST id=%s for section=%s subject=%s",
						csst.CSSTID, sec.ClassSectionID, cs.ClassSubjectSubjectID)

					createdCSSTs = append(createdCSSTs, *csst)
				}
			}
		}
	}

	/* ---- Optional upload image (class) ---- */
	uploadedURL := ""
	if fh := pickImageFile(c, "image", "file", "class_image"); fh != nil {
		log.Printf("[CLASSES][CREATE] 📤 uploading image filename=%s size=%d", fh.Filename, fh.Size)
		svc, er := helperOSS.NewOSSServiceFromEnv("")
		if er == nil {
			ctx, cancel := context.WithTimeout(c.Context(), 45*time.Second)
			defer cancel()

			keyPrefix := fmt.Sprintf("schools/%s/classes", schoolID.String())
			if url, upErr := svc.UploadAsWebP(ctx, fh, keyPrefix); upErr == nil {
				uploadedURL = url

				objKey := ""
				if k, e := helperOSS.ExtractKeyFromPublicURL(uploadedURL); e == nil {
					objKey = k
				} else if k2, e2 := helperOSS.KeyFromPublicURL(uploadedURL); e2 == nil {
					objKey = k2
				}

				m.ClassImageURL = &uploadedURL
				m.ClassImageObjectKey = &objKey
				if err := tx.Model(&classmodel.ClassModel{}).
					Where("class_id = ?", m.ClassID).
					Updates(&classmodel.ClassModel{
						ClassImageURL:       m.ClassImageURL,
						ClassImageObjectKey: m.ClassImageObjectKey,
					}).Error; err != nil {
					log.Printf("[CLASSES][CREATE] ⚠️ persist image fields failed: %v", err)
				} else {
					log.Printf("[CLASSES][CREATE] ✅ image set url=%s key=%s", uploadedURL, objKey)
				}
			} else {
				log.Printf("[CLASSES][CREATE] ❌ upload error: %v", upErr)
			}
		} else {
			log.Printf("[CLASSES][CREATE] ❌ init OSS svc error: %v", er)
		}
	}

	/* ---- Update lembaga_stats bila active ---- */
	if m.ClassStatus == classmodel.ClassStatusActive {
		log.Printf("[CLASSES][CREATE] 📊 updating lembaga_stats (active +1)")
		statsSvc := service.NewLembagaStatsService()
		if err := statsSvc.EnsureForSchool(tx, schoolID); err != nil {
			_ = tx.Rollback().Error
			log.Printf("[CLASSES][CREATE] ❌ ensure stats error: %v", err)
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal inisialisasi lembaga_stats: "+err.Error())
		}
		if err := statsSvc.IncActiveClasses(tx, schoolID, +1); err != nil {
			_ = tx.Rollback().Error
			log.Printf("[CLASSES][CREATE] ❌ inc active classes error: %v", err)
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal update lembaga_stats: "+err.Error())
		}
	}

	/* ---- Commit ---- */
	if err := tx.Commit().Error; err != nil {
		log.Printf("[CLASSES][CREATE] ❌ commit error: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	log.Printf("[CLASSES][CREATE] ✅ done in %s", time.Since(start))

	// 🔚 Response:
	// - Selalu kirim "class" (TZ-aware)
	// - Tambahan "class_sections" & "class_section_subject_teachers" kalau ada yang otomatis dibuat
	resp := fiber.Map{
		"class": dto.FromModel(m).WithSchoolTime(c),
	}
	if len(createdSections) > 0 {
		resp["class_sections"] = classSectionDto.FromSectionModels(createdSections)
	}
	if len(createdCSSTs) > 0 {
		resp["class_section_subject_teachers"] = csstDto.FromCSSTModels(createdCSSTs)
	}

	return helper.JsonCreated(c, "Kelas berhasil dibuat", resp)
}

/* =========================== PATCH =========================== */

// PATCH /admin/classes/:id
func (ctrl *ClassController) PatchClass(c *fiber.Ctx) error {
	// ---- Path param ----
	classID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	// ---- Resolve school dari context ----
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err
	}

	// Ambil "now" versi DB/timezone sekolah untuk update image, dll
	nowDB, err := dbtime.GetDBTime(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mendapatkan waktu server")
	}

	// ---- Parse payload tri-state (JSON / multipart) ----
	var req dto.PatchClassRequest
	if err := dto.DecodePatchClassFromRequest(c, &req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if err := req.Validate(); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// ---- TX ----
	tx := ctrl.DB.WithContext(c.Context()).Begin()
	if tx.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback().Error
			panic(r)
		}
	}()

	// ---- Ambil existing + lock (tenant-safe) ----
	var existing classmodel.ClassModel
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&existing,
			"class_id = ? AND class_school_id = ? AND class_deleted_at IS NULL",
			classID, schoolID,
		).Error; err != nil {

		_ = tx.Rollback().Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Kelas tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// 🔒 Guard: hanya DKM/Admin di school terkait
	if err := helperAuth.EnsureDKMSchool(c, schoolID); err != nil {
		_ = tx.Rollback().Error
		return err
	}

	// ---- Snapshot sebelum apply (untuk deteksi perubahan & stats) ----
	prevParentID := existing.ClassClassParentID
	prevTermID := existing.ClassAcademicTermID
	wasActive := (existing.ClassStatus == classmodel.ClassStatusActive)

	// ---- Apply patch ke entity (field biasa) ----
	req.Apply(&existing)

	// ---- Track perubahan status active (setelah apply) ----
	newActive := (existing.ClassStatus == classmodel.ClassStatusActive)

	// ==== SLUG HANDLING ====
	uuidPtrChanged := func(a, b *uuid.UUID) bool {
		if a == nil && b == nil {
			return false
		}
		if (a == nil) != (b == nil) {
			return true
		}
		return *a != *b
	}

	parentChanged := (existing.ClassClassParentID != prevParentID)
	termChanged := uuidPtrChanged(existing.ClassAcademicTermID, prevTermID)

	// 1) Kalau user PATCH slug manual → hormati, tapi CI-unique per school.
	if req.ClassSlug.Present && req.ClassSlug.Value != nil {
		exp := slugifySafe(existing.ClassSlug, 160) // existing.ClassSlug sudah dari apply()
		if exp == "" {
			_ = tx.Rollback().Error
			return fiber.NewError(fiber.StatusBadRequest, "class_slug tidak boleh kosong")
		}
		uniq, gErr := helper.EnsureUniqueSlugCI(
			c.Context(), tx,
			"classes", "class_slug",
			exp,
			func(q *gorm.DB) *gorm.DB {
				return q.Where(
					"class_school_id = ? AND class_id <> ? AND class_deleted_at IS NULL",
					existing.ClassSchoolID, existing.ClassID,
				)
			},
			160,
		)
		if gErr != nil {
			_ = tx.Rollback().Error
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
		}
		if uniq != exp {
			_ = tx.Rollback().Error
			return fiber.NewError(fiber.StatusConflict, "Slug sudah digunakan di school ini")
		}
		existing.ClassSlug = uniq
	} else {
		// 2) Slug tidak dipatch → regen jika parent/term berubah
		if parentChanged || termChanged {
			baseSlug, gErr := buildClassBaseSlug(
				c.Context(), tx,
				existing.ClassSchoolID,
				existing.ClassClassParentID,
				existing.ClassAcademicTermID,
				"", // paksa komposit
				160,
			)
			if gErr != nil {
				_ = tx.Rollback().Error
				return fiber.NewError(fiber.StatusBadRequest, "Gagal membentuk slug dasar: "+gErr.Error())
			}
			uniq, gErr := helper.EnsureUniqueSlugCI(
				c.Context(), tx,
				"classes", "class_slug",
				baseSlug,
				func(q *gorm.DB) *gorm.DB {
					return q.Where(
						"class_school_id = ? AND class_id <> ? AND class_deleted_at IS NULL",
						existing.ClassSchoolID, existing.ClassID,
					)
				},
				160,
			)
			if gErr != nil {
				_ = tx.Rollback().Error
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
			}
			existing.ClassSlug = uniq
		}
	}

	// ---- Refresh snapshot (parent + term) & class_name jika parent/term berubah ----
	if parentChanged || termChanged {
		if err := classParentSnapshot.HydrateClassParentSnapshot(c.Context(), tx, existing.ClassSchoolID, &existing); err != nil {
			_ = tx.Rollback().Error
			return err
		}
		if err := academicTermsService.HydrateAcademicTermCache(c.Context(), tx, existing.ClassSchoolID, &existing); err != nil {
			_ = tx.Rollback().Error
			return err
		}
		// recompute class_name: "<Parent> — <Term>" (atau hanya parent jika term nil/empty)
		parent := ""
		if existing.ClassClassParentNameCache != nil {
			parent = *existing.ClassClassParentNameCache
		}
		existing.ClassName = strPtrOrNil(dto.ComposeClassNameSpace(parent, existing.ClassAcademicTermNameCache))
	}

	// ---- Simpan ----
	if err := tx.Model(&classmodel.ClassModel{}).
		Where("class_id = ?", existing.ClassID).
		Updates(&existing).Error; err != nil {

		_ = tx.Rollback().Error
		low := strings.ToLower(err.Error())
		switch {
		case strings.Contains(low, "uq_classes_slug_per_school_alive") ||
			(strings.Contains(low, "duplicate") && strings.Contains(low, "class_slug")):
			return fiber.NewError(fiber.StatusConflict, "Slug sudah digunakan di school ini")
		default:
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui data")
		}
	}

	// ---- Optional: upload gambar baru → pindahkan lama ke spam ----
	if fh := pickImageFile(c, "image", "file", "class_image"); fh != nil {
		svc, er := helperOSS.NewOSSServiceFromEnv("")
		if er == nil {
			ctx, cancel := context.WithTimeout(c.Context(), 45*time.Second)
			defer cancel()

			keyPrefix := fmt.Sprintf("schools/%s/classes", existing.ClassSchoolID.String())
			if url, upErr := svc.UploadAsWebP(ctx, fh, keyPrefix); upErr == nil {

				// object key baru
				newObjKey := ""
				if k, e := helperOSS.ExtractKeyFromPublicURL(url); e == nil {
					newObjKey = k
				} else if k2, e2 := helperOSS.KeyFromPublicURL(url); e2 == nil {
					newObjKey = k2
				}

				// ambil url lama dari DB (best effort)
				var oldURL, oldObjKey string
				{
					type row struct {
						URL string `gorm:"column:class_image_url"`
						Key string `gorm:"column:class_image_object_key"`
					}
					var r row
					_ = tx.Table("classes").
						Select("class_image_url, class_image_object_key").
						Where("class_id = ?", existing.ClassID).
						Take(&r).Error
					oldURL = strings.TrimSpace(r.URL)
					oldObjKey = strings.TrimSpace(r.Key)
				}

				movedURL := ""
				if oldURL != "" {
					if mv, mvErr := helperOSS.MoveToSpamByPublicURLENV(oldURL, 0); mvErr == nil {
						movedURL = mv
						// sinkronkan key lama ke lokasi baru
						if k, e := helperOSS.ExtractKeyFromPublicURL(movedURL); e == nil {
							oldObjKey = k
						} else if k2, e2 := helperOSS.KeyFromPublicURL(movedURL); e2 == nil {
							oldObjKey = k2
						}
					}
				}

				deletePendingUntil := nowDB.Add(30 * 24 * time.Hour)

				_ = tx.Model(&classmodel.ClassModel{}).
					Where("class_id = ?", existing.ClassID).
					Updates(map[string]any{
						"class_image_url":        url,
						"class_image_object_key": newObjKey,
						"class_image_url_old": func() any {
							if movedURL == "" {
								return gorm.Expr("NULL")
							}
							return movedURL
						}(),
						"class_image_object_key_old": func() any {
							if oldObjKey == "" {
								return gorm.Expr("NULL")
							}
							return oldObjKey
						}(),
						"class_image_delete_pending_until": deletePendingUntil,
						"class_updated_at":                 nowDB,
					})
			}
		}
	}

	// ---- Update lembaga_stats jika transisi active berubah ----
	if wasActive != newActive {
		stats := service.NewLembagaStatsService()
		if err := stats.EnsureForSchool(tx, existing.ClassSchoolID); err != nil {
			_ = tx.Rollback().Error
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal inisialisasi lembaga_stats: "+err.Error())
		}
		delta := -1
		if newActive {
			delta = +1
		}
		if err := stats.IncActiveClasses(tx, existing.ClassSchoolID, delta); err != nil {
			_ = tx.Rollback().Error
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal update lembaga_stats: "+err.Error())
		}
	}

	// ---- Commit ----
	if err := tx.Commit().Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	// ✅ Response: pakai WithSchoolTime supaya timestamp sudah timezone sekolah
	return helper.JsonUpdated(c, "Kelas berhasil diperbarui", dto.FromModel(&existing).WithSchoolTime(c))
}

/* =========================== DELETE (soft) =========================== */

// DELETE /admin/classes/:id
func (ctrl *ClassController) DeleteClass(c *fiber.Ctx) error {
	classID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	// ---- Resolve school dari context ----
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err
	}

	// Ambil "now" dari dbtime (timezone sekolah)
	now, err := dbtime.GetDBTime(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mendapatkan waktu server")
	}

	// Lock row + cek school_id untuk guard
	tx := ctrl.DB.Begin()
	if tx.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
	}()

	var m classmodel.ClassModel
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where(
			"class_id = ? AND class_school_id = ? AND class_deleted_at IS NULL",
			classID, schoolID,
		).
		First(&m).Error; err != nil {

		_ = tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Kelas tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// 🔒 Guard: hanya DKM/Admin pada school terkait
	if err := helperAuth.EnsureDKMSchool(c, schoolID); err != nil {
		_ = tx.Rollback()
		return err
	}

	// 🔒 GUARD: masih dipakai di class_sections?
	var sectionCount int64
	if err := tx.WithContext(c.Context()).
		Model(&classSectionModel.ClassSectionModel{}).
		Where(`
			class_section_school_id = ?
			AND class_section_class_id = ?
			AND class_section_deleted_at IS NULL
		`, m.ClassSchoolID, m.ClassID).
		Count(&sectionCount).Error; err != nil {

		_ = tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengecek relasi class sections")
	}
	if sectionCount > 0 {
		_ = tx.Rollback()
		return fiber.NewError(
			fiber.StatusBadRequest,
			"Kelas tidak dapat dihapus karena masih digunakan oleh rombongan belajar (class sections). Mohon hapus/ubah class section yang terkait terlebih dahulu.",
		)
	}

	wasActive := (m.ClassStatus == classmodel.ClassStatusActive)

	updates := map[string]any{
		"class_deleted_at": &now,
		"class_updated_at": now,
		"class_status":     "inactive", // opsional
	}
	if err := tx.Model(&classmodel.ClassModel{}).
		Where("class_id = ?", m.ClassID).
		Updates(updates).Error; err != nil {

		_ = tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus data")
	}

	// Decrement stats jika sebelumnya ACTIVE
	if wasActive {
		stats := service.NewLembagaStatsService()
		if err := stats.EnsureForSchool(tx, m.ClassSchoolID); err != nil {
			_ = tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal inisialisasi lembaga_stats: "+err.Error())
		}
		if err := stats.IncActiveClasses(tx, m.ClassSchoolID, -1); err != nil {
			_ = tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal update lembaga_stats: "+err.Error())
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonDeleted(c, "Kelas berhasil dihapus", fiber.Map{
		"class_id": m.ClassID,
	})
}

/* =========================== Util =========================== */

func pickImageFile(c *fiber.Ctx, names ...string) *multipart.FileHeader {
	for _, n := range names {
		if fh, err := c.FormFile(n); err == nil && fh != nil {
			return fh
		}
	}
	return nil
}
