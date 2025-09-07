// file: internals/features/attendance/controller/user_attendance_type_controller.go
package controller

import (
	"errors"
	"strings"
	"time"

	dto "masjidku_backend/internals/features/school/sessions_assesment/sessions/dto"
	model "masjidku_backend/internals/features/school/sessions_assesment/sessions/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserAttendanceTypeController struct {
	DB        *gorm.DB
	Validator *validator.Validate
}

func NewUserAttendanceTypeController(db *gorm.DB) *UserAttendanceTypeController {
	return &UserAttendanceTypeController{
		DB:        db,
		Validator: validator.New(),
	}
}

// =============== Utils ===============


func parseUUIDParam(c *fiber.Ctx, name string) (uuid.UUID, error) {
	return uuid.Parse(strings.TrimSpace(c.Params(name)))
}


// =============== Handlers ===============

// POST /
func (ctl *UserAttendanceTypeController) Create(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	var in dto.UserAttendanceTypeCreateDTO
	if err := c.BodyParser(&in); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	// Enforce tenant dari token
	in.UserAttendanceTypeMasjidID = masjidID

	if err := dto.ValidateStruct(ctl.Validator, &in); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	m := in.ToModel()
	if err := ctl.DB.WithContext(c.Context()).Create(m).Error; err != nil {
		if isDuplicateKey(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Kode sudah terpakai untuk masjid ini (aktif)")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonCreated(c, "Berhasil membuat attendance type", dto.FromModel(m))
}

// GET /
func (ctl *UserAttendanceTypeController) List(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	// Pagination & sorting (pakai helper)
	p := helper.ParseFiber(c, "created_at", "desc", helper.DefaultOpts)

	// Query filter
	q := dto.UserAttendanceTypeListQuery{
		UserAttendanceTypeMasjidID: masjidID,
	}

	if v := strings.TrimSpace(c.Query("code_eq")); v != "" {
		q.CodeEq = &v
	}
	if v := strings.TrimSpace(c.Query("label_query")); v != "" {
		q.LabelQueryILK = &v
	}
	if v := strings.TrimSpace(c.Query("only_active")); v != "" {
		val := strings.EqualFold(v, "true") || v == "1" || strings.EqualFold(v, "yes")
		q.OnlyActive = &val
	}
	// Pakai limit/offset dari helper
	q.Limit = p.Limit()
	q.Offset = p.Offset()

	// Mapping sort_by/order → q.Sort (opsional; tetap support param "sort" lama)
	if sb := strings.TrimSpace(c.Query("sort_by")); sb != "" {
		order := strings.ToLower(strings.TrimSpace(c.Query("order")))
		if order != "asc" && order != "desc" {
			order = "desc"
		}
		switch sb {
		case "created_at":
			if order == "asc" {
				q.Sort = "created_at_asc"
			} else {
				q.Sort = "created_at_desc"
			}
		case "code":
			if order == "asc" {
				q.Sort = "code_asc"
			} else {
				q.Sort = "code_desc"
			}
		case "label":
			if order == "asc" {
				q.Sort = "label_asc"
			} else {
				q.Sort = "label_desc"
			}
		}
	} else if v := strings.TrimSpace(c.Query("sort")); v != "" {
		// fallback ke parameter lama "sort"
		q.Sort = v
	}

	if err := dto.ValidateStruct(ctl.Validator, &q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	g := q.BuildQuery(ctl.DB.WithContext(c.Context()))
	var total int64
	if err := g.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	var rows []model.UserAttendanceTypeModel
	if err := g.Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	items := make([]dto.UserAttendanceTypeItem, len(rows))
	for i := range rows {
		items[i] = dto.FromModel(&rows[i])
	}

	meta := helper.BuildMeta(total, p)
	return helper.JsonList(c, items, meta)
}

// GET /:id
func (ctl *UserAttendanceTypeController) GetByID(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var m model.UserAttendanceTypeModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("user_attendance_type_id = ? AND user_attendance_type_masjid_id = ? AND user_attendance_type_deleted_at IS NULL",
			id, masjidID).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonOK(c, "OK", dto.FromModel(&m))
}

// PATCH /:id
func (ctl *UserAttendanceTypeController) Patch(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	// ambil data existing
	var m model.UserAttendanceTypeModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("user_attendance_type_id = ? AND user_attendance_type_masjid_id = ? AND user_attendance_type_deleted_at IS NULL",
			id, masjidID).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	var in dto.UserAttendanceTypePatchDTO
	if err := c.BodyParser(&in); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	// enforce key dari path + token
	in.UserAttendanceTypeID = id
	in.UserAttendanceTypeMasjidID = masjidID

	if err := dto.ValidateStruct(ctl.Validator, &in); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	in.ApplyPatch(&m)

	// Simpan perubahan
	if err := ctl.DB.WithContext(c.Context()).
		Model(&m).
		Select(
			"user_attendance_type_code",
			"user_attendance_type_label",
			"user_attendance_type_desc",
			"user_attendance_type_is_active",
			"user_attendance_type_updated_at",
		).
		Updates(&m).Error; err != nil {
		if isDuplicateKey(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Kode sudah terpakai untuk masjid ini (aktif)")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonUpdated(c, "Berhasil mengubah attendance type", dto.FromModel(&m))
}

// DELETE /:id (soft delete)
func (ctl *UserAttendanceTypeController) Delete(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	res := ctl.DB.WithContext(c.Context()).
		Model(&model.UserAttendanceTypeModel{}).
		Where("user_attendance_type_id = ? AND user_attendance_type_masjid_id = ? AND user_attendance_type_deleted_at IS NULL",
			id, masjidID).
		Updates(map[string]any{
			"user_attendance_type_deleted_at": time.Now(),
			"user_attendance_type_updated_at": time.Now(),
		})
	if res.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, res.Error.Error())
	}
	if res.RowsAffected == 0 {
		return helper.JsonError(c, fiber.StatusNotFound, "Tidak ditemukan / sudah dihapus")
	}

	return helper.JsonDeleted(c, "Berhasil menghapus attendance type", fiber.Map{"id": id})
}

// POST /:id/restore
func (ctl *UserAttendanceTypeController) Restore(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	res := ctl.DB.WithContext(c.Context()).
		Model(&model.UserAttendanceTypeModel{}).
		Where("user_attendance_type_id = ? AND user_attendance_type_masjid_id = ? AND user_attendance_type_deleted_at IS NOT NULL",
			id, masjidID).
		Updates(map[string]any{
			"user_attendance_type_deleted_at": nil,
			"user_attendance_type_updated_at": time.Now(),
		})
	if res.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, res.Error.Error())
	}
	if res.RowsAffected == 0 {
		return helper.JsonError(c, fiber.StatusNotFound, "Tidak ditemukan / sudah aktif")
	}

	return helper.JsonOK(c, "Berhasil restore attendance type", fiber.Map{"id": id})
}
