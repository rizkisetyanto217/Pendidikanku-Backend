// file: internals/features/attendance/controller/class_attendance_session_participant_type_controller.go
package controller

import (
	"errors"
	"strings"
	"time"

	dto "schoolku_backend/internals/features/school/classes/class_attendance_sessions/dto"
	model "schoolku_backend/internals/features/school/classes/class_attendance_sessions/model"
	helper "schoolku_backend/internals/helpers"
	helperAuth "schoolku_backend/internals/helpers/auth"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClassAttendanceSessionParticipantTypeController struct {
	DB        *gorm.DB
	Validator *validator.Validate
}

func NewClassAttendanceSessionParticipantTypeController(db *gorm.DB) *ClassAttendanceSessionParticipantTypeController {
	return &ClassAttendanceSessionParticipantTypeController{
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
func (ctl *ClassAttendanceSessionParticipantTypeController) Create(c *fiber.Ctx) error {
	// School context via helpers (slug/ID di path/header/query/host) → wajib DKM/Admin
	c.Locals("DB", ctl.DB)
	var schoolID uuid.UUID
	if mc, err := helperAuth.ResolveSchoolContext(c); err == nil && (mc.ID != uuid.Nil || strings.TrimSpace(mc.Slug) != "") {
		if id, er := helperAuth.EnsureSchoolAccessDKM(c, mc); er == nil {
			schoolID = id
		} else {
			if fe, ok := er.(*fiber.Error); ok {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return helper.JsonError(c, fiber.StatusForbidden, er.Error())
		}
	} else {
		// Fallback: prefer TEACHER → DKM/Admin (single-tenant token)
		if id, err := helperAuth.GetSchoolIDFromTokenPreferTeacher(c); err == nil && id != uuid.Nil {
			schoolID = id
		} else {
			return helper.JsonError(c, fiber.StatusUnauthorized, "Scope school tidak ditemukan")
		}
	}

	var in dto.ClassAttendanceSessionParticipantTypeCreateDTO
	if err := c.BodyParser(&in); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	// Enforce tenant dari context
	in.ClassAttendanceSessionParticipantTypeSchoolID = schoolID

	if err := dto.ValidateStruct(ctl.Validator, &in); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	m := in.ToModel()
	if err := ctl.DB.WithContext(c.Context()).Create(m).Error; err != nil {
		if isDuplicateKey(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Kode sudah terpakai untuk school ini (aktif)")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonCreated(c, "Berhasil membuat participant type", dto.FromModel(m))
}

// GET /
func (ctl *ClassAttendanceSessionParticipantTypeController) List(c *fiber.Ctx) error {
	// School context (DKM/Admin jika eksplisit; jika tidak, fallback token/teacher)
	c.Locals("DB", ctl.DB)
	var schoolID uuid.UUID
	if mc, err := helperAuth.ResolveSchoolContext(c); err == nil && (mc.ID != uuid.Nil || strings.TrimSpace(mc.Slug) != "") {
		if id, er := helperAuth.EnsureSchoolAccessDKM(c, mc); er == nil {
			schoolID = id
		} else {
			if fe, ok := er.(*fiber.Error); ok {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return helper.JsonError(c, fiber.StatusForbidden, er.Error())
		}
	} else {
		if id, err := helperAuth.GetSchoolIDFromTokenPreferTeacher(c); err == nil && id != uuid.Nil {
			schoolID = id
		} else {
			return helper.JsonError(c, fiber.StatusUnauthorized, "Scope school tidak ditemukan")
		}
	}

	// ===== Paging standar (jsonresponse)
	p := helper.ResolvePaging(c, 20, 200)

	// ===== Query filter dari querystring
	q := dto.ClassAttendanceSessionParticipantTypeListQuery{
		ClassAttendanceSessionParticipantTypeSchoolID: schoolID,
		Limit:  p.Limit,
		Offset: p.Offset,
	}

	if v := strings.TrimSpace(c.Query("code_eq")); v != "" {
		q.CodeEq = &v
	}
	if v := strings.TrimSpace(c.Query("label_query")); v != "" {
		q.LabelQueryILK = &v
	}
	if v := strings.TrimSpace(c.Query("slug_eq")); v != "" {
		q.SlugEq = &v
	}
	if v := strings.TrimSpace(c.Query("only_active")); v != "" {
		val := strings.EqualFold(v, "true") || v == "1" || strings.EqualFold(v, "yes")
		q.OnlyActive = &val
	}

	// Mapping sort_by/order → q.Sort (tetap dukung param lama ?sort=)
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
		q.Sort = v
	}

	if err := dto.ValidateStruct(ctl.Validator, &q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// ===== Build query, count, fetch
	g := q.BuildQuery(ctl.DB.WithContext(c.Context()))

	var total int64
	if err := g.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	var rows []model.ClassAttendanceSessionParticipantTypeModel
	if err := g.Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	items := make([]dto.ClassAttendanceSessionParticipantTypeItem, len(rows))
	for i := range rows {
		items[i] = dto.FromModel(&rows[i])
	}

	// ===== Meta & response (jsonresponse)
	pg := helper.BuildPaginationFromOffset(total, p.Offset, p.Limit)
	return helper.JsonList(c, "ok", items, pg)
}

// PATCH /:id
func (ctl *ClassAttendanceSessionParticipantTypeController) Patch(c *fiber.Ctx) error {
	// School context
	c.Locals("DB", ctl.DB)
	var schoolID uuid.UUID
	if mc, err := helperAuth.ResolveSchoolContext(c); err == nil && (mc.ID != uuid.Nil || strings.TrimSpace(mc.Slug) != "") {
		if id, er := helperAuth.EnsureSchoolAccessDKM(c, mc); er == nil {
			schoolID = id
		} else {
			if fe, ok := er.(*fiber.Error); ok {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return helper.JsonError(c, fiber.StatusForbidden, er.Error())
		}
	} else {
		if id, err := helperAuth.GetSchoolIDFromTokenPreferTeacher(c); err == nil && id != uuid.Nil {
			schoolID = id
		} else {
			return helper.JsonError(c, fiber.StatusUnauthorized, "Scope school tidak ditemukan")
		}
	}

	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	// ambil data existing
	var m model.ClassAttendanceSessionParticipantTypeModel
	if err := ctl.DB.WithContext(c.Context()).
		Where(`
			class_attendance_session_participant_type_id = ?
			AND class_attendance_session_participant_type_school_id = ?
			AND class_attendance_session_participant_type_deleted_at IS NULL
		`, id, schoolID).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	var in dto.ClassAttendanceSessionParticipantTypePatchDTO
	if err := c.BodyParser(&in); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	// enforce key dari path + context
	in.ClassAttendanceSessionParticipantTypeID = id
	in.ClassAttendanceSessionParticipantTypeSchoolID = schoolID

	if err := dto.ValidateStruct(ctl.Validator, &in); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	in.ApplyPatch(&m)

	// Simpan perubahan (select kolom yang diperbolehkan)
	if err := ctl.DB.WithContext(c.Context()).
		Model(&m).
		Select(
			"class_attendance_session_participant_type_code",
			"class_attendance_session_participant_type_label",
			"class_attendance_session_participant_type_slug",
			"class_attendance_session_participant_type_color",
			"class_attendance_session_participant_type_desc",
			"class_attendance_session_participant_type_is_active",
			"class_attendance_session_participant_type_updated_at",
		).
		Updates(&m).Error; err != nil {
		if isDuplicateKey(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Kode sudah terpakai untuk school ini (aktif)")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonUpdated(c, "Berhasil mengubah participant type", dto.FromModel(&m))
}

// DELETE /:id (soft delete)
func (ctl *ClassAttendanceSessionParticipantTypeController) Delete(c *fiber.Ctx) error {
	// School context
	c.Locals("DB", ctl.DB)
	var schoolID uuid.UUID
	if mc, err := helperAuth.ResolveSchoolContext(c); err == nil && (mc.ID != uuid.Nil || strings.TrimSpace(mc.Slug) != "") {
		if id, er := helperAuth.EnsureSchoolAccessDKM(c, mc); er == nil {
			schoolID = id
		} else {
			if fe, ok := er.(*fiber.Error); ok {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return helper.JsonError(c, fiber.StatusForbidden, er.Error())
		}
	} else {
		if id, err := helperAuth.GetSchoolIDFromTokenPreferTeacher(c); err == nil && id != uuid.Nil {
			schoolID = id
		} else {
			return helper.JsonError(c, fiber.StatusUnauthorized, "Scope school tidak ditemukan")
		}
	}

	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	res := ctl.DB.WithContext(c.Context()).
		Model(&model.ClassAttendanceSessionParticipantTypeModel{}).
		Where(`
			class_attendance_session_participant_type_id = ?
			AND class_attendance_session_participant_type_school_id = ?
			AND class_attendance_session_participant_type_deleted_at IS NULL
		`, id, schoolID).
		Updates(map[string]any{
			"class_attendance_session_participant_type_deleted_at": time.Now(),
			"class_attendance_session_participant_type_updated_at": time.Now(),
		})
	if res.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, res.Error.Error())
	}
	if res.RowsAffected == 0 {
		return helper.JsonError(c, fiber.StatusNotFound, "Tidak ditemukan / sudah dihapus")
	}

	return helper.JsonDeleted(c, "Berhasil menghapus participant type", fiber.Map{"id": id})
}

// POST /:id/restore
func (ctl *ClassAttendanceSessionParticipantTypeController) Restore(c *fiber.Ctx) error {
	// School context
	c.Locals("DB", ctl.DB)
	var schoolID uuid.UUID
	if mc, err := helperAuth.ResolveSchoolContext(c); err == nil && (mc.ID != uuid.Nil || strings.TrimSpace(mc.Slug) != "") {
		if id, er := helperAuth.EnsureSchoolAccessDKM(c, mc); er == nil {
			schoolID = id
		} else {
			if fe, ok := er.(*fiber.Error); ok {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return helper.JsonError(c, fiber.StatusForbidden, er.Error())
		}
	} else {
		if id, err := helperAuth.GetSchoolIDFromTokenPreferTeacher(c); err == nil && id != uuid.Nil {
			schoolID = id
		} else {
			return helper.JsonError(c, fiber.StatusUnauthorized, "Scope school tidak ditemukan")
		}
	}

	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	res := ctl.DB.WithContext(c.Context()).
		Model(&model.ClassAttendanceSessionParticipantTypeModel{}).
		Where(`
			class_attendance_session_participant_type_id = ?
			AND class_attendance_session_participant_type_school_id = ?
			AND class_attendance_session_participant_type_deleted_at IS NOT NULL
		`, id, schoolID).
		Updates(map[string]any{
			"class_attendance_session_participant_type_deleted_at": nil,
			"class_attendance_session_participant_type_updated_at": time.Now(),
		})
	if res.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, res.Error.Error())
	}
	if res.RowsAffected == 0 {
		return helper.JsonError(c, fiber.StatusNotFound, "Tidak ditemukan / sudah aktif")
	}

	return helper.JsonOK(c, "Berhasil restore participant type", fiber.Map{"id": id})
}
