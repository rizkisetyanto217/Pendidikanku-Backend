// file: internals/features/school/class_rooms/controller/class_room_controller.go
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

	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	csstModel "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/model"
	classSectionModel "madinahsalam_backend/internals/features/school/classes/class_sections/model"

	dto "madinahsalam_backend/internals/features/school/academics/rooms/dto"
	model "madinahsalam_backend/internals/features/school/academics/rooms/model"
	helperOSS "madinahsalam_backend/internals/helpers/oss"
)

/* =======================================================
   CONTROLLER
   ======================================================= */

type ClassRoomController struct {
	DB       *gorm.DB
	Validate *validator.Validate
}

func NewClassRoomController(db *gorm.DB, v *validator.Validate) *ClassRoomController {
	if v == nil {
		v = validator.New(validator.WithRequiredStructEnabled())
	}
	return &ClassRoomController{DB: db, Validate: v}
}

// jaga-jaga kalau ada controller lama yang di-init tanpa validator
func (ctl *ClassRoomController) ensureValidator() {
	if ctl.Validate == nil {
		ctl.Validate = validator.New(validator.WithRequiredStructEnabled())
	}
}

// ambil context standar (kalau Fiber mendukung UserContext)
func reqCtx(c *fiber.Ctx) context.Context {
	if uc := c.UserContext(); uc != nil {
		return uc
	}
	return context.Background()
}

// util kecil untuk ambil file dari beberapa nama umum
func pickImageFile(c *fiber.Ctx, names ...string) *multipart.FileHeader {
	for _, n := range names {
		if fh, err := c.FormFile(n); err == nil && fh != nil && fh.Size > 0 {
			return fh
		}
	}
	return nil
}

/* ============================ CREATE ============================ */
func (ctl *ClassRoomController) Create(c *fiber.Ctx) error {
	ctl.ensureValidator()

	// Pastikan DB tersedia di context (dipakai helper auth / slug helper)
	if c.Locals("DB") == nil {
		c.Locals("DB", ctl.DB)
	}

	// 🔒 Ambil school_id dari token, wajib ada
	schoolID, err := helperAuth.GetSchoolIDFromToken(c)
	if err != nil || schoolID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "School context tidak ditemukan di token")
	}

	// 🔒 Hanya DKM/Admin school ini
	if err := helperAuth.EnsureDKMSchool(c, schoolID); err != nil {
		return err
	}

	// 👀 Deteksi multipart
	ct := strings.ToLower(strings.TrimSpace(c.Get("Content-Type")))
	isMultipart := strings.HasPrefix(ct, "multipart/form-data")
	log.Printf("[CLASSROOM][CREATE] Content-Type=%q isMultipart=%v", ct, isMultipart)

	// 📦 Parse payload
	var req dto.CreateClassRoomRequest
	if isMultipart {
		// ===== Multipart: isi SEMUA kolom sesuai DTO =====
		req.ClassRoomName = strings.TrimSpace(c.FormValue("class_room_name"))

		if v := strings.TrimSpace(c.FormValue("class_room_slug")); v != "" {
			s := helper.Slugify(v, 50)
			req.ClassRoomSlug = &s
		}
		if v := strings.TrimSpace(c.FormValue("class_room_code")); v != "" {
			req.ClassRoomCode = &v
		}
		if v := strings.TrimSpace(c.FormValue("class_room_location")); v != "" {
			req.ClassRoomLocation = &v
		}
		if v := strings.TrimSpace(c.FormValue("class_room_description")); v != "" {
			req.ClassRoomDescription = &v
		}

		// int (pointer)
		if v := strings.TrimSpace(c.FormValue("class_room_capacity")); v != "" {
			if n, er := strconv.Atoi(v); er == nil {
				req.ClassRoomCapacity = &n
			}
		}

		// bool → pointer bool
		parseBool := func(s string, def bool) bool {
			s = strings.ToLower(strings.TrimSpace(s))
			switch s {
			case "1", "true", "yes", "y":
				return true
			case "0", "false", "no", "n":
				return false
			default:
				return def
			}
		}
		if v := c.FormValue("class_room_is_virtual"); v != "" {
			b := parseBool(v, false)
			req.ClassRoomIsVirtual = &b
		}
		if v := c.FormValue("class_room_is_active"); v != "" {
			b := parseBool(v, true)
			req.ClassRoomIsActive = &b
		}

		// features: JSON array atau CSV → []string
		if v := strings.TrimSpace(c.FormValue("class_room_features")); v != "" {
			var arr []string
			if strings.HasPrefix(v, "[") {
				_ = json.Unmarshal([]byte(v), &arr)
			} else {
				for _, p := range strings.Split(v, ",") {
					p = strings.TrimSpace(p)
					if p != "" {
						arr = append(arr, p)
					}
				}
			}
			req.ClassRoomFeatures = arr
		}

		// ONLINE fields (opsional)
		if v := strings.TrimSpace(c.FormValue("class_room_platform")); v != "" {
			req.ClassRoomPlatform = &v
		}
		if v := strings.TrimSpace(c.FormValue("class_room_join_url")); v != "" {
			req.ClassRoomJoinURL = &v
		}
		if v := strings.TrimSpace(c.FormValue("class_room_meeting_id")); v != "" {
			req.ClassRoomMeetingID = &v
		}
		if v := strings.TrimSpace(c.FormValue("class_room_passcode")); v != "" {
			req.ClassRoomPasscode = &v
		}

		// schedule & notes
		if v := strings.TrimSpace(c.FormValue("class_room_schedule")); v != "" {
			var arr []dto.AnyObject
			if err := json.Unmarshal([]byte(v), &arr); err == nil {
				req.ClassRoomSchedule = arr
			}
		}
		if v := strings.TrimSpace(c.FormValue("class_room_notes")); v != "" {
			var arr []dto.AnyObject
			if err := json.Unmarshal([]byte(v), &arr); err == nil {
				req.ClassRoomNotes = arr
			}
		}
	} else {
		// JSON / x-www-form-urlencoded
		if err := c.BodyParser(&req); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
		}
	}

	// 🚩 Inject tenant dari server
	req.ClassRoomSchoolID = schoolID

	// ✅ Validasi payload
	if err := ctl.Validate.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Validasi gagal: "+err.Error())
	}

	// 🔁 Slug unik
	base := ""
	if req.ClassRoomSlug != nil {
		base = strings.TrimSpace(*req.ClassRoomSlug)
	}
	if base == "" {
		base = helper.SuggestSlugFromName(req.ClassRoomName)
		if base == "" {
			base = helper.Slugify(req.ClassRoomName, 50)
		}
	}
	slug, err := helper.EnsureUniqueSlugCI(
		reqCtx(c), ctl.DB,
		"class_rooms", "class_room_slug",
		base,
		func(q *gorm.DB) *gorm.DB {
			return q.Where("class_room_school_id = ? AND class_room_deleted_at IS NULL", schoolID)
		},
		50,
	)
	if err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat slug unik")
	}

	// 🧭 Map DTO → model
	m, err := req.ToModel()
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid (features/schedule/notes/virtual_links)")
	}

	// 🔒 Pastikan dari server
	m.ClassRoomSchoolID = schoolID
	m.ClassRoomSlug = &slug

	// 💾 Simpan awal (tanpa image)
	if err := ctl.DB.WithContext(reqCtx(c)).Create(&m).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Slug ruang sudah digunakan, silakan ubah nama/slug")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan data")
	}

	// 🖼️ Upload image (opsional, jika multipart dan ada file)
	uploadedURL := ""
	if isMultipart {
		if fh := pickImageFile(c, "image", "file", "cover"); fh != nil {
			log.Printf("[CLASSROOM][CREATE] will upload file: name=%q size=%d", fh.Filename, fh.Size)

			keyPrefix := fmt.Sprintf("schools/%s/school/class-rooms", schoolID.String())

			svc, er := helperOSS.NewOSSServiceFromEnv("") // gunakan env default
			if er != nil {
				log.Printf("[CLASSROOM][CREATE] OSS init error: %T %v", er, er)
			} else {
				ctx, cancel := context.WithTimeout(c.Context(), 45*time.Second)
				defer cancel()

				url, upErr := svc.UploadAsWebP(ctx, fh, keyPrefix)
				if upErr != nil {
					log.Printf("[CLASSROOM][CREATE] upload error: %T %v", upErr, upErr)
				} else {
					uploadedURL = url

					// object key (opsional)
					objKey := ""
					if k, e := helperOSS.ExtractKeyFromPublicURL(uploadedURL); e == nil {
						objKey = k
					} else if k2, e2 := helperOSS.KeyFromPublicURL(uploadedURL); e2 == nil {
						objKey = k2
					}

					// Update kolom image
					if err := ctl.DB.WithContext(reqCtx(c)).
						Model(&model.ClassRoomModel{}).
						Where("class_room_id = ?", m.ClassRoomID).
						Updates(map[string]any{
							"class_room_image_url":        uploadedURL,
							"class_room_image_object_key": objKey,
						}).Error; err != nil {
						log.Printf("[CLASSROOM][CREATE] DB.Updates image err: %T %v", err, err)
					} else {
						m.ClassRoomImageURL = &uploadedURL
						if objKey != "" {
							m.ClassRoomImageObjectKey = &objKey
						} else {
							m.ClassRoomImageObjectKey = nil
						}
					}
				}
			}
		} else {
			log.Printf("[CLASSROOM][CREATE] no image file found in multipart (tried: image,file,cover)")
		}
	}

	// (Best-effort) refresh entity
	_ = ctl.DB.WithContext(reqCtx(c)).First(&m, "class_room_id = ?", m.ClassRoomID).Error

	// 🎯 Response → pakai versi timezone sekolah
	resp := dto.ToClassRoomResponseWithSchoolTime(c, m)
	return helper.JsonCreated(c, "Created", resp)
}

/* ============================ UPDATE (PUT/PATCH semantics) ============================ */
func (ctl *ClassRoomController) Patch(c *fiber.Ctx) error {
	ctl.ensureValidator()

	// Pastikan DB available di Locals (buat helper lain kalau butuh)
	if c.Locals("DB") == nil {
		c.Locals("DB", ctl.DB)
	}

	// 1) Ambil school_id dari token
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err
	}
	if schoolID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "School context tidak ditemukan di token")
	}

	// 2) Hanya DKM/Admin school ini
	if err := helperAuth.EnsureDKMSchool(c, schoolID); err != nil {
		return err
	}

	// 3) Param id
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	// 4) Parse payload & validasi
	var req dto.UpdateClassRoomRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.Validate.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Validasi gagal: "+err.Error())
	}

	// 5) Ambil data lama
	var m model.ClassRoomModel
	if err := ctl.DB.WithContext(reqCtx(c)).
		Where("class_room_id = ? AND class_room_school_id = ? AND class_room_deleted_at IS NULL", id, schoolID).
		First(&m).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// 6) Apply patch
	if err := req.ApplyPatch(&m); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Gagal menerapkan perubahan: "+err.Error())
	}

	// 7) Slug logic (auto/ganti)
	if req.ClassRoomName != nil {
		if req.ClassRoomSlug == nil || strings.TrimSpace(*req.ClassRoomSlug) == "" {
			// generate dari nama
			base := helper.Slugify(*req.ClassRoomName, 50)
			slug, err := helper.EnsureUniqueSlugCI(
				reqCtx(c), ctl.DB,
				"class_rooms", "class_room_slug",
				base,
				func(q *gorm.DB) *gorm.DB {
					return q.Where(
						"class_room_school_id = ? AND class_room_id <> ? AND class_room_deleted_at IS NULL",
						schoolID, id,
					)
				},
				50,
			)
			if err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat slug unik")
			}
			m.ClassRoomSlug = &slug
		} else {
			// user kirim slug eksplisit → slugify + unik
			base := helper.Slugify(strings.TrimSpace(*req.ClassRoomSlug), 50)
			slug, err := helper.EnsureUniqueSlugCI(
				reqCtx(c), ctl.DB,
				"class_rooms", "class_room_slug",
				base,
				func(q *gorm.DB) *gorm.DB {
					return q.Where(
						"class_room_school_id = ? AND class_room_id <> ? AND class_room_deleted_at IS NULL",
						schoolID, id,
					)
				},
				50,
			)
			if err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat slug unik")
			}
			m.ClassRoomSlug = &slug
		}
	}
	// === END slug logic ===

	// 8) Save DB
	if err := ctl.DB.WithContext(reqCtx(c)).Save(&m).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Slug atau kode ruang sudah digunakan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengubah data")
	}

	// 9) Response → pakai timezone sekolah
	return helper.JsonUpdated(c, "Updated", dto.ToClassRoomResponseWithSchoolTime(c, m))
}

/* ============================ DELETE ============================ */
func (ctl *ClassRoomController) Delete(c *fiber.Ctx) error {
	// Pastikan DB di Locals (kalau ada helper lain yang pakai)
	if c.Locals("DB") == nil {
		c.Locals("DB", ctl.DB)
	}

	// 1) school_id dari token
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err
	}
	if schoolID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "School context tidak ditemukan di token")
	}

	// 2) Hanya DKM/Admin
	if err := helperAuth.EnsureDKMSchool(c, schoolID); err != nil {
		return err
	}

	// 3) Param id
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil || id == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	// GUARD pemakaian di class_sections
	var secCount int64
	if err := ctl.DB.Model(&classSectionModel.ClassSectionModel{}).
		Where(`
			class_section_school_id = ?
			AND class_section_class_room_id = ?
			AND class_section_deleted_at IS NULL
		`, schoolID, id).
		Count(&secCount).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengecek pemakaian room di kelas/rombel")
	}

	// GUARD pemakaian di CSST (schema baru: csst_*)
	var csstCount int64
	if err := ctl.DB.Model(&csstModel.ClassSectionSubjectTeacherModel{}).
		Where(`
		csst_school_id = ?
		AND csst_class_room_id = ?
		AND csst_deleted_at IS NULL
	`, schoolID, id).
		Count(&csstCount).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengecek pemakaian room di pengampu mapel")
	}

	if secCount > 0 || csstCount > 0 {
		return helper.JsonError(
			c,
			fiber.StatusBadRequest,
			"Tidak dapat menghapus ruang kelas karena masih digunakan di kelas/rombel atau pengampu mapel. Lepaskan relasi tersebut terlebih dahulu.",
		)
	}

	// Soft delete
	tx := ctl.DB.WithContext(reqCtx(c)).
		Model(&model.ClassRoomModel{}).
		Where("class_room_id = ? AND class_room_school_id = ? AND class_room_deleted_at IS NULL", id, schoolID).
		Update("class_room_deleted_at", time.Now())
	if tx.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus data")
	}
	if tx.RowsAffected == 0 {
		return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan / sudah terhapus")
	}

	return helper.JsonDeleted(c, "Ruang kelas berhasil dihapus", fiber.Map{
		"class_room_id": id,
	})
}

/* ============================ RESTORE ============================ */

func (ctl *ClassRoomController) Restore(c *fiber.Ctx) error {
	// Require DKM/Admin (owner boleh) + resolve schoolID
	mc, err := helperAuth.ResolveSchoolContext(c)
	if err != nil {
		return err
	}

	var schoolID uuid.UUID
	switch {
	case mc.ID != uuid.Nil:
		schoolID = mc.ID
	case strings.TrimSpace(mc.Slug) != "":
		id, er := helperAuth.GetSchoolIDBySlug(c, strings.TrimSpace(mc.Slug))
		if er != nil {
			if errors.Is(er, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, fiber.StatusNotFound, "School (slug) tidak ditemukan")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal resolve school dari slug")
		}
		schoolID = id
	default:
		return helperAuth.ErrSchoolContextMissing
	}

	if !helperAuth.IsOwner(c) {
		if err := helperAuth.EnsureDKMSchool(c, schoolID); err != nil {
			return err
		}
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	// Hanya bisa restore jika baris soft-deleted & tenant match
	tx := ctl.DB.WithContext(reqCtx(c)).Model(&model.ClassRoomModel{}).
		Where("class_room_id = ? AND class_room_school_id = ? AND class_room_deleted_at IS NOT NULL", id, schoolID).
		Updates(map[string]interface{}{
			"class_room_deleted_at": nil,
			"class_room_updated_at": time.Now(),
		})
	if tx.Error != nil {
		if isUniqueViolation(tx.Error) {
			// Restore bisa bentrok karena slug/kode terbentur entri alive lain
			return helper.JsonError(c, fiber.StatusConflict, "Gagal restore: slug atau kode sudah dipakai entri lain")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal restore data")
	}
	if tx.RowsAffected == 0 {
		return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan / tidak dalam keadaan terhapus")
	}

	// Return row terbaru
	var m model.ClassRoomModel
	if err := ctl.DB.WithContext(reqCtx(c)).
		Where("class_room_id = ? AND class_room_school_id = ? AND class_room_deleted_at IS NULL", id, schoolID).
		First(&m).Error; err != nil {
		// kalau gagal ambil ulang, minimal beri flag restored
		return helper.JsonOK(c, "Restored", fiber.Map{"restored": true})
	}

	// 🔹 pakai response timezone-aware
	return helper.JsonOK(c, "Restored", dto.ToClassRoomResponseWithSchoolTime(c, m))
}

/* =======================================================
   HELPERS (local)
   ======================================================= */

// Deteksi unique violation Postgres (kode "23505")
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "duplicate key") || strings.Contains(s, "unique constraint") || strings.Contains(s, "23505")
}
