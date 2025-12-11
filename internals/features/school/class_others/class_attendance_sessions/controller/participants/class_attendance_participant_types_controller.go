package controller

import (
	"errors"
	"strings"
	"time"

	dto "madinahsalam_backend/internals/features/school/class_others/class_attendance_sessions/dto"
	model "madinahsalam_backend/internals/features/school/class_others/class_attendance_sessions/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

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

// helper: resolve schoolID dari context (id / slug) + enforce DKM/admin
// helper: resolve schoolID dari token + enforce DKM/admin
func (ctl *ClassAttendanceSessionParticipantTypeController) resolveDKMSchoolID(c *fiber.Ctx) (uuid.UUID, error) {
	// pastikan DB ditaruh di Locals, karena helper auth biasa baca dari sini
	c.Locals("DB", ctl.DB)

	// 1) ambil schoolID dari token (prefer teacher, lalu active-school)
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return uuid.Nil, err // helper sudah balikin JsonError yang proper
	}

	// 2) enforce role: hanya DKM/Admin di school ini
	if err := helperAuth.EnsureDKMSchool(c, schoolID); err != nil {
		return uuid.Nil, err
	}

	return schoolID, nil
}

// =============== Handlers ===============

// POST /
func (ctl *ClassAttendanceSessionParticipantTypeController) Create(c *fiber.Ctx) error {
	schoolID, err := ctl.resolveDKMSchoolID(c)
	if err != nil {
		return err
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

	// ✅ balikan jam sudah di-convert ke timezone sekolah
	return helper.JsonCreated(c, "Berhasil membuat participant type", dto.FromModelWithSchoolTime(c, m))
}

// GET /
func (ctl *ClassAttendanceSessionParticipantTypeController) List(c *fiber.Ctx) error {
	schoolID, err := ctl.resolveDKMSchoolID(c)
	if err != nil {
		return err
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
		// ✅ convert timestamps per item
		items[i] = dto.FromModelWithSchoolTime(c, &rows[i])
	}

	// ===== Meta & response (jsonresponse)
	pg := helper.BuildPaginationFromOffset(total, p.Offset, p.Limit)
	return helper.JsonList(c, "ok", items, pg)
}

// PATCH /:id
func (ctl *ClassAttendanceSessionParticipantTypeController) Patch(c *fiber.Ctx) error {
	schoolID, err := ctl.resolveDKMSchoolID(c)
	if err != nil {
		return err
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

	// Apply patch ke model (termasuk config & meta)
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

			"class_attendance_session_participant_type_allow_student_self_attendance",
			"class_attendance_session_participant_type_allow_teacher_mark_attendance",
			"class_attendance_session_participant_type_require_teacher_attendance",
			"class_attendance_session_participant_type_require_attendance_reason",
			"class_attendance_session_participant_type_meta",

			"class_attendance_session_participant_type_updated_at",
		).
		Updates(&m).Error; err != nil {
		if isDuplicateKey(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Kode sudah terpakai untuk school ini (aktif)")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// ✅ balikan jam sudah di-convert
	return helper.JsonUpdated(c, "Berhasil mengubah participant type", dto.FromModelWithSchoolTime(c, &m))
}

// DELETE /:id (soft delete)
func (ctl *ClassAttendanceSessionParticipantTypeController) Delete(c *fiber.Ctx) error {
	schoolID, err := ctl.resolveDKMSchoolID(c)
	if err != nil {
		return err
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
	schoolID, err := ctl.resolveDKMSchoolID(c)
	if err != nil {
		return err
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
