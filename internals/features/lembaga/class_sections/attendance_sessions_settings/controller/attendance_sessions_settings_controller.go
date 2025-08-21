// internals/features/lembaga/class_sections/attendance_sessions_settings/controller/class_attendance_setting_controller.go
package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	dto "masjidku_backend/internals/features/lembaga/class_sections/attendance_sessions_settings/dto"
	mdl "masjidku_backend/internals/features/lembaga/class_sections/attendance_sessions_settings/model"
	helper "masjidku_backend/internals/helpers"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClassAttendanceSettingController struct {
	DB *gorm.DB
}

func NewClassAttendanceSettingController(db *gorm.DB) *ClassAttendanceSettingController {
	return &ClassAttendanceSettingController{DB: db}
}

// ========== GET ==========
// GET /class_attendance_settings
// - Ambil settings utk masjid di token (prefer teacher → union → admin)
func (ctl *ClassAttendanceSettingController) GetBySchool(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}

	var m mdl.ClassAttendanceSetting
	if err := ctl.DB.WithContext(c.Context()).
		Where("class_attendance_setting_masjid_id = ?", masjidID).
		Limit(1).
		Find(&m).Error; err != nil {
		return fiber.NewError(http.StatusInternalServerError, err.Error())
	}

	if m.ClassAttendanceSettingID == uuid.Nil {
		// belum ada → kembalikan null (FE bisa putuskan untuk create)
		return c.Status(http.StatusOK).JSON(fiber.Map{"data": nil})
	}
	return c.Status(http.StatusOK).JSON(fiber.Map{"data": dto.FromModel(&m)})
}

// ========== POST (CREATE) ==========
// POST /class_attendance_settings
// - Membuat baris baru untuk masjid di token (admin-only).
// - Jika sudah ada → 409 Conflict.
func (ctl *ClassAttendanceSettingController) CreateBySchool(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c) // admin-only
	if err != nil {
		return err
	}

	// 1) Wajib JSON
	ct := c.Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		return fiber.NewError(fiber.StatusUnsupportedMediaType, "use application/json")
	}

	// 2) Parse JSON tegas (hindari form behavior & field nyasar)
	var payload dto.ClassAttendanceSettingDTO
	dec := json.NewDecoder(bytes.NewReader(c.Body()))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&payload); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid JSON: "+err.Error())
	}

	// 3) Scope dari token & ID create
	payload.ClassAttendanceSettingMasjidID = masjidID
	payload.ClassAttendanceSettingID = uuid.Nil

	// 4) DTO -> Model
	m := payload.ToModel()

	// 5) Validasi "require ⇒ enable"
	if err := validateRequireImpliesEnable(m); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// 6) INSERT
	if err := ctl.DB.WithContext(c.Context()).Create(m).Error; err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "duplicate key value") || strings.Contains(msg, "unique") {
			return fiber.NewError(fiber.StatusConflict, "settings already exist for this school, use PUT to update")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "attendance settings created",
		"data":    dto.FromModel(m),
	})
}


// ========== PUT (UPDATE) ==========
// PUT /class_attendance_settings
// - Mengubah baris yang sudah ada utk masjid di token (admin-only).
// - Jika belum ada → 404 Not Found.
// - Partial update: hanya field yang dikirim yang di-update.
func (ctl *ClassAttendanceSettingController) UpdateBySchool(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c) // admin-only
	if err != nil {
		return err
	}

	// pastikan sudah ada record
	var existing mdl.ClassAttendanceSetting
	if err := ctl.DB.WithContext(c.Context()).
		Where("class_attendance_setting_masjid_id = ?", masjidID).
		Limit(1).
		Find(&existing).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if existing.ClassAttendanceSettingID == uuid.Nil {
		return fiber.NewError(fiber.StatusNotFound, "settings not found for this school, use POST to create")
	}

	// baca raw payload sebagai map agar bisa deteksi field mana yang dikirim
	var raw map[string]any
	if err := c.BodyParser(&raw); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid JSON payload")
	}

	// whitelist jsonKey -> dbColumn
	fieldMap := map[string]string{
		"class_attendance_setting_enable_score":              "class_attendance_setting_enable_score",
		"class_attendance_setting_require_score":             "class_attendance_setting_require_score",
		"class_attendance_setting_enable_grade_passed":       "class_attendance_setting_enable_grade_passed",
		"class_attendance_setting_require_grade_passed":      "class_attendance_setting_require_grade_passed",
		"class_attendance_setting_enable_material_personal":  "class_attendance_setting_enable_material_personal",
		"class_attendance_setting_require_material_personal": "class_attendance_setting_require_material_personal",
		"class_attendance_setting_enable_personal_note":      "class_attendance_setting_enable_personal_note",
		"class_attendance_setting_require_personal_note":     "class_attendance_setting_require_personal_note",
		"class_attendance_setting_enable_memorization":       "class_attendance_setting_enable_memorization",
		"class_attendance_setting_require_memorization":      "class_attendance_setting_require_memorization",
		"class_attendance_setting_enable_homework":           "class_attendance_setting_enable_homework",
		"class_attendance_setting_require_homework":          "class_attendance_setting_require_homework",
	}

	// siapkan effective state (mulai dari existing)
	eff := existing

	// siapkan updates yang benar-benar dikirim
	updates := map[string]any{}

	// helper baca bool dari raw (handle bool atau angka 0/1 / string "true"/"false")
	readBool := func(v any) (bool, bool) {
		switch t := v.(type) {
		case bool:
			return t, true
		case float64: // JSON number
			return t != 0, true
		case string:
			s := strings.TrimSpace(strings.ToLower(t))
			if s == "true" || s == "1" {
				return true, true
			}
			if s == "false" || s == "0" {
				return false, true
			}
			return false, false
		default:
			return false, false
		}
	}

	// terapkan hanya key yang dikirim
	for jsonKey, col := range fieldMap {
		if val, ok := raw[jsonKey]; ok {
			if b, ok2 := readBool(val); ok2 {
				updates[col] = b
				// set ke effective state juga
				switch jsonKey {
				case "class_attendance_setting_enable_score":
					eff.ClassAttendanceSettingEnableScore = b
				case "class_attendance_setting_require_score":
					eff.ClassAttendanceSettingRequireScore = b
				case "class_attendance_setting_enable_grade_passed":
					eff.ClassAttendanceSettingEnableGradePassed = b
				case "class_attendance_setting_require_grade_passed":
					eff.ClassAttendanceSettingRequireGradePassed = b
				case "class_attendance_setting_enable_material_personal":
					eff.ClassAttendanceSettingEnableMaterialPersonal = b
				case "class_attendance_setting_require_material_personal":
					eff.ClassAttendanceSettingRequireMaterialPersonal = b
				case "class_attendance_setting_enable_personal_note":
					eff.ClassAttendanceSettingEnablePersonalNote = b
				case "class_attendance_setting_require_personal_note":
					eff.ClassAttendanceSettingRequirePersonalNote = b
				case "class_attendance_setting_enable_memorization":
					eff.ClassAttendanceSettingEnableMemorization = b
				case "class_attendance_setting_require_memorization":
					eff.ClassAttendanceSettingRequireMemorization = b
				case "class_attendance_setting_enable_homework":
					eff.ClassAttendanceSettingEnableHomework = b
				case "class_attendance_setting_require_homework":
					eff.ClassAttendanceSettingRequireHomework = b
				}
			} else {
				return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("invalid boolean for %s", jsonKey))
			}
		}
	}

	// jika tidak ada field yang dikirim
	if len(updates) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "no updatable fields provided")
	}

	// validasi require ⇒ enable pada effective state
	if err := validateRequireImpliesEnable(&eff); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// UPDATE hanya kolom yang dikirim
	res := ctl.DB.WithContext(c.Context()).
		Model(&mdl.ClassAttendanceSetting{}).
		Where("class_attendance_setting_masjid_id = ?", masjidID).
		Updates(updates)
	if res.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, res.Error.Error())
	}
	if res.RowsAffected == 0 {
		// jaga-jaga
		return fiber.NewError(fiber.StatusNotFound, "settings not found for this school")
	}

	// Ambil lagi hasil akhir untuk response
	var saved mdl.ClassAttendanceSetting
	if err := ctl.DB.WithContext(c.Context()).
		Where("class_attendance_setting_masjid_id = ?", masjidID).
		Limit(1).
		Find(&saved).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "attendance settings updated",
		"data":    dto.FromModel(&saved),
	})
}

// ========== helper ==========
func validateRequireImpliesEnable(m *mdl.ClassAttendanceSetting) error {
	type pair struct {
		req, en bool
		name    string
	}
	checks := []pair{
		{m.ClassAttendanceSettingRequireScore, m.ClassAttendanceSettingEnableScore, "score"},
		{m.ClassAttendanceSettingRequireGradePassed, m.ClassAttendanceSettingEnableGradePassed, "grade_passed"},
		{m.ClassAttendanceSettingRequireMaterialPersonal, m.ClassAttendanceSettingEnableMaterialPersonal, "material_personal"},
		{m.ClassAttendanceSettingRequirePersonalNote, m.ClassAttendanceSettingEnablePersonalNote, "personal_note"},
		{m.ClassAttendanceSettingRequireMemorization, m.ClassAttendanceSettingEnableMemorization, "memorization"},
		{m.ClassAttendanceSettingRequireHomework, m.ClassAttendanceSettingEnableHomework, "homework"},
	}
	for _, c := range checks {
		if c.req && !c.en {
			return fmt.Errorf("require_%s=true but enable_%s=false", c.name, c.name)
		}
	}
	return nil
}
