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

	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	dto "masjidku_backend/internals/features/school/academics/rooms/dto"
	model "masjidku_backend/internals/features/school/academics/rooms/model"
	helperOSS "masjidku_backend/internals/helpers/oss"
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

	// 🔒 Ambil context masjid & guard
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
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

		// schedule & notes: harap kirim JSON array of objects → []dto.AnyObject
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
	req.ClassRoomMasjidID = masjidID

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
			return q.Where("class_room_masjid_id = ? AND class_room_deleted_at IS NULL", masjidID)
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
	m.ClassRoomMasjidID = masjidID
	m.ClassRoomSlug = &slug

	// 💾 Simpan awal (tanpa image)
	if err := ctl.DB.WithContext(reqCtx(c)).Create(&m).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Nama/Kode/Slug ruang sudah digunakan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan data")
	}

	// 🖼️ Upload image (opsional, jika multipart dan ada file)
	uploadedURL := ""
	if isMultipart {
		if fh := pickImageFile(c, "image", "file", "cover"); fh != nil {
			log.Printf("[CLASSROOM][CREATE] will upload file: name=%q size=%d", fh.Filename, fh.Size)

			keyPrefix := fmt.Sprintf("masjids/%s/school/class-rooms", masjidID.String())

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

	// 🎯 Response
	resp := dto.ToClassRoomResponse(m)
	return helper.JsonCreated(c, "Created", resp)
}

/* ============================ UPDATE (PUT/PATCH semantics) ============================ */
func (ctl *ClassRoomController) Update(c *fiber.Ctx) error {
	ctl.ensureValidator()

	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var req dto.UpdateClassRoomRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.Validate.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Validasi gagal: "+err.Error())
	}

	var m model.ClassRoomModel
	if err := ctl.DB.WithContext(reqCtx(c)).
		Where("class_room_id = ? AND class_room_masjid_id = ? AND class_room_deleted_at IS NULL", id, masjidID).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// Terapkan patch (mutasi in-place)
	if err := req.ApplyPatch(&m); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Gagal menerapkan perubahan: "+err.Error())
	}

	// === NEW: Auto-update slug ketika nama berubah, kecuali slug diisi eksplisit ===
	if req.ClassRoomName != nil {
		// Hanya generate otomatis jika user TIDAK kirim slug baru
		if req.ClassRoomSlug == nil || strings.TrimSpace(*req.ClassRoomSlug) == "" {
			base := helper.Slugify(*req.ClassRoomName, 50)
			slug, err := helper.EnsureUniqueSlugCI(
				reqCtx(c), ctl.DB,
				"class_rooms", "class_room_slug",
				base,
				func(q *gorm.DB) *gorm.DB {
					// unik per masjid, exclude diri sendiri, hanya alive
					return q.Where("class_room_masjid_id = ? AND class_room_id <> ? AND class_room_deleted_at IS NULL", masjidID, id)
				},
				50,
			)
			if err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat slug unik")
			}
			m.ClassRoomSlug = &slug
		} else {
			// Jika user kirim slug, pastikan unik juga (normalisasi + unik)
			base := helper.Slugify(strings.TrimSpace(*req.ClassRoomSlug), 50)
			slug, err := helper.EnsureUniqueSlugCI(
				reqCtx(c), ctl.DB,
				"class_rooms", "class_room_slug",
				base,
				func(q *gorm.DB) *gorm.DB {
					return q.Where("class_room_masjid_id = ? AND class_room_id <> ? AND class_room_deleted_at IS NULL", masjidID, id)
				},
				50,
			)
			if err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat slug unik")
			}
			m.ClassRoomSlug = &slug
		}
	}
	// === END NEW ===

	if err := ctl.DB.WithContext(reqCtx(c)).Save(&m).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Nama/Kode/Slug ruang sudah digunakan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengubah data")
	}

	return helper.JsonUpdated(c, "Updated", dto.ToClassRoomResponse(m))
}

/* ============================ PATCH (alias Update) ============================ */

func (ctl *ClassRoomController) Patch(c *fiber.Ctx) error {
	// Gunakan payload yang sama dengan Update
	return ctl.Update(c)
}

/* ============================ DELETE ============================ */

func (ctl *ClassRoomController) Delete(c *fiber.Ctx) error {
	// Require DKM/Admin + resolve masjidID
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	// Pastikan tenant match & alive → soft delete
	tx := ctl.DB.WithContext(reqCtx(c)).Model(&model.ClassRoomModel{}).
		Where("class_room_id = ? AND class_room_masjid_id = ? AND class_room_deleted_at IS NULL", id, masjidID).
		Update("class_room_deleted_at", time.Now())
	if tx.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus data")
	}
	if tx.RowsAffected == 0 {
		return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan / sudah terhapus")
	}
	return helper.JsonDeleted(c, "Deleted", fiber.Map{"deleted": true})
}

/* ============================ RESTORE ============================ */

func (ctl *ClassRoomController) Restore(c *fiber.Ctx) error {
	// Require DKM/Admin + resolve masjidID
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	// Hanya bisa restore jika baris soft-deleted & tenant match
	tx := ctl.DB.WithContext(reqCtx(c)).Model(&model.ClassRoomModel{}).
		Where("class_room_id = ? AND class_room_masjid_id = ? AND class_room_deleted_at IS NOT NULL", id, masjidID).
		Updates(map[string]interface{}{
			"class_room_deleted_at": nil,
			"class_room_updated_at": time.Now(),
		})
	if tx.Error != nil {
		if isUniqueViolation(tx.Error) {
			// Restore bisa bentrok dengan partial unique (nama/kode/slug sudah dipakai baris alive lain)
			return helper.JsonError(c, fiber.StatusConflict, "Gagal restore: nama/kode/slug sudah dipakai entri lain")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal restore data")
	}
	if tx.RowsAffected == 0 {
		return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan / tidak dalam keadaan terhapus")
	}

	// Return row terbaru
	var m model.ClassRoomModel
	if err := ctl.DB.WithContext(reqCtx(c)).
		Where("class_room_id = ? AND class_room_masjid_id = ? AND class_room_deleted_at IS NULL", id, masjidID).
		First(&m).Error; err != nil {
		// kalau gagal ambil ulang, minimal beri flag restored
		return helper.JsonOK(c, "Restored", fiber.Map{"restored": true})
	}
	return helper.JsonOK(c, "Restored", dto.ToClassRoomResponse(m))
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
