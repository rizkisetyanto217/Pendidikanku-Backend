// file: internals/features/school/classes/attendance/controller/class_attendance_session_type_controller.go
package controller

import (
	"errors"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	dto "madinahsalam_backend/internals/features/school/class_others/class_attendance_sessions/dto"
	model "madinahsalam_backend/internals/features/school/class_others/class_attendance_sessions/model"
	helper "madinahsalam_backend/internals/helpers"
)

/* ======================================================
   Controller
====================================================== */

type ClassAttendanceSessionTypeController struct {
	DB        *gorm.DB
	Validator *validator.Validate
}

// ✅ constructor biar bisa dipanggil: uaCtrl.NewClassAttendanceSessionTypeController(db)
func NewClassAttendanceSessionTypeController(db *gorm.DB) *ClassAttendanceSessionTypeController {
	return &ClassAttendanceSessionTypeController{
		DB:        db,
		Validator: validator.New(),
	}
}

/* ======================================================
   Helpers
====================================================== */

func getSchoolIDFromCtx(c *fiber.Ctx) (uuid.UUID, error) {
	raw := c.Locals("school_id")
	if raw == nil {
		return uuid.Nil, errors.New("school scope not found")
	}

	switch v := raw.(type) {
	case uuid.UUID:
		return v, nil
	case string:
		v = strings.TrimSpace(v)
		if v == "" {
			return uuid.Nil, errors.New("school scope is empty")
		}
		id, err := uuid.Parse(v)
		if err != nil {
			return uuid.Nil, errors.New("invalid school scope")
		}
		return id, nil
	default:
		return uuid.Nil, errors.New("invalid school scope type")
	}
}

func parseBoolQuery(c *fiber.Ctx, key string) (*bool, error) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return nil, nil
	}
	val, err := strconv.ParseBool(raw)
	if err != nil {
		return nil, err
	}
	return &val, nil
}

// mapping error validator → fieldErrors
func buildFieldErrors(err error) map[string][]string {
	out := map[string][]string{}
	if err == nil {
		return out
	}

	verrs, ok := err.(validator.ValidationErrors)
	if !ok {
		// fallback generic
		out["_error"] = []string{err.Error()}
		return out
	}

	for _, fe := range verrs {
		field := fe.Field()
		tag := fe.Tag()

		msg := ""
		switch tag {
		case "required":
			msg = "field is required"
		case "max":
			msg = "value is too long"
		case "min":
			msg = "value is too short"
		default:
			msg = "invalid value"
		}

		out[field] = append(out[field], msg)
	}
	return out
}

/* ======================================================
   Request DTO (local)
   👉 slug dihapus dari request, digenerate otomatis dari Name
====================================================== */

type classAttendanceSessionTypeUpsertRequest struct {
	// identitas
	ClassAttendanceSessionTypeName        string  `json:"class_attendance_session_type_name" validate:"required"`
	ClassAttendanceSessionTypeDescription *string `json:"class_attendance_session_type_description"`

	// tampilan
	ClassAttendanceSessionTypeColor *string `json:"class_attendance_session_type_color"`
	ClassAttendanceSessionTypeIcon  *string `json:"class_attendance_session_type_icon"`

	// control umum
	ClassAttendanceSessionTypeIsActive  *bool `json:"class_attendance_session_type_is_active"`
	ClassAttendanceSessionTypeSortOrder *int  `json:"class_attendance_session_type_sort_order"`

	// konfigurasi attendance
	ClassAttendanceSessionTypeAllowStudentSelfAttendance *bool    `json:"class_attendance_session_type_allow_student_self_attendance"`
	ClassAttendanceSessionTypeAllowTeacherMarkAttendance *bool    `json:"class_attendance_session_type_allow_teacher_mark_attendance"`
	ClassAttendanceSessionTypeRequireTeacherAttendance   *bool    `json:"class_attendance_session_type_require_teacher_attendance"`
	ClassAttendanceSessionTypeRequireAttendanceReason    []string `json:"class_attendance_session_type_require_attendance_reason"`

	// meta fleksibel
	ClassAttendanceSessionTypeMeta map[string]any `json:"class_attendance_session_type_meta"`
}

/*
	======================================================
	  List
	  GET /api/.../class-attendance-session-types
	  Query:
	    - q          : optional, search by name / slug
	    - is_active  : optional, true/false
	    - page       : default 1
	    - per_page   : default 20, max 100

======================================================
*/
func (ctl *ClassAttendanceSessionTypeController) List(c *fiber.Ctx) error {
	schoolID, err := getSchoolIDFromCtx(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	isActive, err := parseBoolQuery(c, "is_active")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid is_active query param")
	}

	// paging standard
	paging := helper.ResolvePaging(c, 20, 100)

	q := strings.TrimSpace(c.Query("q"))
	name := strings.TrimSpace(c.Query("name")) // 🔍 filter khusus by name

	dbq := ctl.DB.
		Model(&model.ClassAttendanceSessionTypeModel{}).
		Where("class_attendance_session_type_school_id = ?", schoolID)

	if isActive != nil {
		dbq = dbq.Where("class_attendance_session_type_is_active = ?", *isActive)
	}

	// 🔍 full-text sederhana: q → name + slug
	if q != "" {
		pattern := "%" + strings.ToLower(q) + "%"
		dbq = dbq.Where(
			"(LOWER(class_attendance_session_type_name) LIKE ? OR LOWER(class_attendance_session_type_slug) LIKE ?)",
			pattern, pattern,
		)
	}

	// 🔍 filter spesifik by name: ?name=
	if name != "" {
		pattern := "%" + strings.ToLower(name) + "%"
		dbq = dbq.Where(
			"LOWER(class_attendance_session_type_name) LIKE ?",
			pattern,
		)
	}

	// total untuk pagination
	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to count attendance session types")
	}

	var rows []*model.ClassAttendanceSessionTypeModel
	if err := dbq.
		Order("class_attendance_session_type_sort_order ASC, class_attendance_session_type_name ASC").
		Limit(paging.Limit).
		Offset(paging.Offset).
		Find(&rows).Error; err != nil {

		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to fetch attendance session types")
	}

	pagination := helper.BuildPaginationFromPage(total, paging.Page, paging.PerPage)
	return helper.JsonList(c, "attendance session types list", dto.NewClassAttendanceSessionTypeDTOs(rows), pagination)
}

/* ======================================================
   Detail
   GET /api/.../class-attendance-session-types/:id
====================================================== */

func (ctl *ClassAttendanceSessionTypeController) Detail(c *fiber.Ctx) error {
	schoolID, err := getSchoolIDFromCtx(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	idParam := strings.TrimSpace(c.Params("id"))
	if idParam == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "missing id param")
	}

	id, err := uuid.Parse(idParam)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid id param")
	}

	var row model.ClassAttendanceSessionTypeModel
	if err := ctl.DB.
		Where("class_attendance_session_type_id = ? AND class_attendance_session_type_school_id = ?", id, schoolID).
		First(&row).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "attendance session type not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to fetch attendance session type")
	}

	return helper.JsonOK(c, "attendance session type detail", dto.NewClassAttendanceSessionTypeDTO(&row))
}

/* ======================================================
   Create
   POST /api/.../class-attendance-session-types
   👉 Slug digenerate otomatis dari Name dan dijamin unik per school
====================================================== */

func (ctl *ClassAttendanceSessionTypeController) Create(c *fiber.Ctx) error {
	schoolID, err := getSchoolIDFromCtx(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var body classAttendanceSessionTypeUpsertRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid request body")
	}

	if ctl.Validator != nil {
		if err := ctl.Validator.Struct(body); err != nil {
			fieldErrors := buildFieldErrors(err)
			return helper.JsonValidationError(c, fieldErrors)
		}
	}

	// default control values (selaras migration)
	isActive := true
	if body.ClassAttendanceSessionTypeIsActive != nil {
		isActive = *body.ClassAttendanceSessionTypeIsActive
	}

	sortOrder := 0
	if body.ClassAttendanceSessionTypeSortOrder != nil {
		sortOrder = *body.ClassAttendanceSessionTypeSortOrder
	}

	allowSelf := false
	if body.ClassAttendanceSessionTypeAllowStudentSelfAttendance != nil {
		allowSelf = *body.ClassAttendanceSessionTypeAllowStudentSelfAttendance
	}

	allowTeacherMark := true
	if body.ClassAttendanceSessionTypeAllowTeacherMarkAttendance != nil {
		allowTeacherMark = *body.ClassAttendanceSessionTypeAllowTeacherMarkAttendance
	}

	requireTeacher := false
	if body.ClassAttendanceSessionTypeRequireTeacherAttendance != nil {
		requireTeacher = *body.ClassAttendanceSessionTypeRequireTeacherAttendance
	}

	requireReason := body.ClassAttendanceSessionTypeRequireAttendanceReason
	if requireReason == nil {
		requireReason = []string{}
	}

	var meta datatypes.JSONMap
	if body.ClassAttendanceSessionTypeMeta != nil {
		meta = datatypes.JSONMap(body.ClassAttendanceSessionTypeMeta)
	}

	// 🔹 Generate base slug from name (ex: "Pertemuan Kelas" → "pertemuan-kelas")
	baseSlug := helper.SuggestSlugFromName(body.ClassAttendanceSessionTypeName)

	// 🔹 Ensure slug unik per tenant (school)
	uniqueSlug, err := helper.EnsureUniqueSlugCI(
		c.UserContext(),
		ctl.DB,
		"class_attendance_session_types",
		"class_attendance_session_type_slug",
		baseSlug,
		func(q *gorm.DB) *gorm.DB {
			return q.Where("class_attendance_session_type_school_id = ?", schoolID)
		},
		160, // max length
	)
	if err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to generate unique slug")
	}

	row := &model.ClassAttendanceSessionTypeModel{
		ClassAttendanceSessionTypeSchoolID:    schoolID,
		ClassAttendanceSessionTypeSlug:        uniqueSlug,
		ClassAttendanceSessionTypeName:        strings.TrimSpace(body.ClassAttendanceSessionTypeName),
		ClassAttendanceSessionTypeDescription: body.ClassAttendanceSessionTypeDescription,
		ClassAttendanceSessionTypeColor:       body.ClassAttendanceSessionTypeColor,
		ClassAttendanceSessionTypeIcon:        body.ClassAttendanceSessionTypeIcon,
		ClassAttendanceSessionTypeIsActive:    isActive,
		ClassAttendanceSessionTypeSortOrder:   sortOrder,

		ClassAttendanceSessionTypeAllowStudentSelfAttendance: allowSelf,
		ClassAttendanceSessionTypeAllowTeacherMarkAttendance: allowTeacherMark,
		ClassAttendanceSessionTypeRequireTeacherAttendance:   requireTeacher,
		ClassAttendanceSessionTypeRequireAttendanceReason:    requireReason,
		ClassAttendanceSessionTypeMeta:                       meta,
	}

	if err := ctl.DB.Create(row).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to create attendance session type")
	}

	return helper.JsonCreated(c, "attendance session type created", dto.NewClassAttendanceSessionTypeDTO(row))
}

/* ======================================================
   Update
   PUT /api/.../class-attendance-session-types/:id
   👉 Slug TIDAK diubah, supaya stabil
====================================================== */

func (ctl *ClassAttendanceSessionTypeController) Update(c *fiber.Ctx) error {
	schoolID, err := getSchoolIDFromCtx(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	idParam := strings.TrimSpace(c.Params("id"))
	if idParam == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "missing id param")
	}

	id, err := uuid.Parse(idParam)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid id param")
	}

	var body classAttendanceSessionTypeUpsertRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid request body")
	}

	if ctl.Validator != nil {
		if err := ctl.Validator.Struct(body); err != nil {
			fieldErrors := buildFieldErrors(err)
			return helper.JsonValidationError(c, fieldErrors)
		}
	}

	var row model.ClassAttendanceSessionTypeModel
	if err := ctl.DB.
		Where("class_attendance_session_type_id = ? AND class_attendance_session_type_school_id = ?", id, schoolID).
		First(&row).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "attendance session type not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to fetch attendance session type")
	}

	// ❌ slug tidak diubah, biarkan stabil
	row.ClassAttendanceSessionTypeName = strings.TrimSpace(body.ClassAttendanceSessionTypeName)
	row.ClassAttendanceSessionTypeDescription = body.ClassAttendanceSessionTypeDescription
	row.ClassAttendanceSessionTypeColor = body.ClassAttendanceSessionTypeColor
	row.ClassAttendanceSessionTypeIcon = body.ClassAttendanceSessionTypeIcon

	if body.ClassAttendanceSessionTypeIsActive != nil {
		row.ClassAttendanceSessionTypeIsActive = *body.ClassAttendanceSessionTypeIsActive
	}
	if body.ClassAttendanceSessionTypeSortOrder != nil {
		row.ClassAttendanceSessionTypeSortOrder = *body.ClassAttendanceSessionTypeSortOrder
	}

	// konfigurasi attendance
	if body.ClassAttendanceSessionTypeAllowStudentSelfAttendance != nil {
		row.ClassAttendanceSessionTypeAllowStudentSelfAttendance = *body.ClassAttendanceSessionTypeAllowStudentSelfAttendance
	}
	if body.ClassAttendanceSessionTypeAllowTeacherMarkAttendance != nil {
		row.ClassAttendanceSessionTypeAllowTeacherMarkAttendance = *body.ClassAttendanceSessionTypeAllowTeacherMarkAttendance
	}
	if body.ClassAttendanceSessionTypeRequireTeacherAttendance != nil {
		row.ClassAttendanceSessionTypeRequireTeacherAttendance = *body.ClassAttendanceSessionTypeRequireTeacherAttendance
	}

	if body.ClassAttendanceSessionTypeRequireAttendanceReason != nil {
		row.ClassAttendanceSessionTypeRequireAttendanceReason = body.ClassAttendanceSessionTypeRequireAttendanceReason
	}

	if body.ClassAttendanceSessionTypeMeta != nil {
		row.ClassAttendanceSessionTypeMeta = datatypes.JSONMap(body.ClassAttendanceSessionTypeMeta)
	}

	if err := ctl.DB.Save(&row).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to update attendance session type")
	}

	return helper.JsonUpdated(c, "attendance session type updated", dto.NewClassAttendanceSessionTypeDTO(&row))
}

/* ======================================================
   Delete (Soft Delete)
   DELETE /api/.../class-attendance-session-types/:id
====================================================== */

func (ctl *ClassAttendanceSessionTypeController) Delete(c *fiber.Ctx) error {
	schoolID, err := getSchoolIDFromCtx(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	idParam := strings.TrimSpace(c.Params("id"))
	if idParam == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "missing id param")
	}

	id, err := uuid.Parse(idParam)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid id param")
	}

	var row model.ClassAttendanceSessionTypeModel
	if err := ctl.DB.
		Where("class_attendance_session_type_id = ? AND class_attendance_session_type_school_id = ?", id, schoolID).
		First(&row).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "attendance session type not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to fetch attendance session type")
	}

	if err := ctl.DB.Delete(&row).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to delete attendance session type")
	}

	return helper.JsonDeleted(c, "attendance session type deleted", dto.NewClassAttendanceSessionTypeDTO(&row))
}
