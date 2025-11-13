// file: internals/features/school/sessions/sessions/controller/student_attendance_list_controller.go
package controller

import (
	"strings"
	"time"

	attDTO "schoolku_backend/internals/features/school/classes/class_attendance_sessions/dto"
	attModel "schoolku_backend/internals/features/school/classes/class_attendance_sessions/model"
	helper "schoolku_backend/internals/helpers"
	helperAuth "schoolku_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* =================================================================================
   LIST / GET BY ID (Participant)
   ================================================================================= */

// Build list query (tenant-aware) — versi participant
func (ctl *StudentAttendanceController) buildListQuery(
	c *fiber.Ctx,
	q attDTO.ListClassAttendanceSessionParticipantQuery,
	schoolID uuid.UUID,
) (*gorm.DB, error) {
	tx := ctl.DB.WithContext(c.Context()).
		Model(&attModel.ClassAttendanceSessionParticipantModel{}).
		Where(`
			class_attendance_session_participant_school_id = ?
			AND class_attendance_session_participant_deleted_at IS NULL
		`, schoolID)

	// Search di desc / notes
	if len(q.Search) > 0 {
		// gabung semua term search jadi satu string
		searchStr := strings.Join(q.Search, " ")
		if s := strings.TrimSpace(searchStr); s != "" {
			like := "%" + s + "%"
			tx = tx.Where(`
				COALESCE(class_attendance_session_participant_desc,'') ILIKE ? OR
				COALESCE(class_attendance_session_participant_user_note,'') ILIKE ? OR
				COALESCE(class_attendance_session_participant_teacher_note,'') ILIKE ?
			`, like, like, like)
		}
	}

	// state_in (present|absent|late|excused|sick|leave)
	if len(q.StateIn) > 0 {
		valid := make([]string, 0, len(q.StateIn))
		for _, v := range q.StateIn {
			vv := strings.ToLower(strings.TrimSpace(v))
			switch vv {
			case "present", "absent", "late", "excused", "sick", "leave":
				valid = append(valid, vv)
			}
		}
		if len(valid) > 0 {
			tx = tx.Where("class_attendance_session_participant_state IN ?", valid)
		}
	}

	// method_in
	if len(q.MethodIn) > 0 {
		valid := make([]string, 0, len(q.MethodIn))
		for _, v := range q.MethodIn {
			vv := strings.ToLower(strings.TrimSpace(v))
			switch vv {
			case "manual", "qr", "geo", "import", "api", "self":
				valid = append(valid, vv)
			}
		}
		if len(valid) > 0 {
			tx = tx.Where("class_attendance_session_participant_method IN ?", valid)
		}
	}

	// kind_in (student|teacher|assistant|guest)
	if len(q.KindIn) > 0 {
		valid := make([]string, 0, len(q.KindIn))
		for _, v := range q.KindIn {
			vv := strings.ToLower(strings.TrimSpace(v))
			switch vv {
			case "student", "teacher", "assistant", "guest":
				valid = append(valid, vv)
			}
		}
		if len(valid) > 0 {
			tx = tx.Where("class_attendance_session_participant_kind IN ?", valid)
		}
	}

	// Filter ID (string → uuid)
	if s := strings.TrimSpace(q.SessionID); s != "" {
		if id, e := uuid.Parse(s); e == nil {
			tx = tx.Where("class_attendance_session_participant_session_id = ?", id)
		} else {
			return nil, helper.JsonError(c, fiber.StatusBadRequest, "session_id tidak valid")
		}
	}
	if s := strings.TrimSpace(q.SchoolStudentID); s != "" {
		if id, e := uuid.Parse(s); e == nil {
			tx = tx.Where("class_attendance_session_participant_school_student_id = ?", id)
		} else {
			return nil, helper.JsonError(c, fiber.StatusBadRequest, "school_student_id tidak valid")
		}
	}
	if s := strings.TrimSpace(q.SchoolTeacherID); s != "" {
		if id, e := uuid.Parse(s); e == nil {
			tx = tx.Where("class_attendance_session_participant_school_teacher_id = ?", id)
		} else {
			return nil, helper.JsonError(c, fiber.StatusBadRequest, "school_teacher_id tidak valid")
		}
	}
	if s := strings.TrimSpace(q.TypeID); s != "" {
		if id, e := uuid.Parse(s); e == nil {
			tx = tx.Where("class_attendance_session_participant_type_id = ?", id)
		} else {
			return nil, helper.JsonError(c, fiber.StatusBadRequest, "type_id tidak valid")
		}
	}
	if s := strings.TrimSpace(q.MarkedByTID); s != "" {
		if id, e := uuid.Parse(s); e == nil {
			tx = tx.Where("class_attendance_session_participant_marked_by_teacher_id = ?", id)
		} else {
			return nil, helper.JsonError(c, fiber.StatusBadRequest, "marked_by_teacher_id tidak valid")
		}
	}

	// Rentang waktu created_at
	if s := strings.TrimSpace(q.CreatedGE); s != "" {
		t, err := time.Parse(dateLayout, s)
		if err != nil {
			return nil, helper.JsonError(c, fiber.StatusBadRequest, "created_ge invalid format, expected YYYY-MM-DD")
		}
		tx = tx.Where("class_attendance_session_participant_created_at >= ?", t)
	}
	if s := strings.TrimSpace(q.CreatedLE); s != "" {
		t, err := time.Parse(dateLayout, s)
		if err != nil {
			return nil, helper.JsonError(c, fiber.StatusBadRequest, "created_le invalid format, expected YYYY-MM-DD")
		}
		tx = tx.Where("class_attendance_session_participant_created_at < ?", t.Add(24*time.Hour))
	}

	// Rentang waktu marked_at
	if s := strings.TrimSpace(q.MarkedGE); s != "" {
		t, err := time.Parse(dateLayout, s)
		if err != nil {
			return nil, helper.JsonError(c, fiber.StatusBadRequest, "marked_ge invalid format, expected YYYY-MM-DD")
		}
		tx = tx.Where(`
			class_attendance_session_participant_marked_at IS NOT NULL
			AND class_attendance_session_participant_marked_at >= ?
		`, t)
	}
	if s := strings.TrimSpace(q.MarkedLE); s != "" {
		t, err := time.Parse(dateLayout, s)
		if err != nil {
			return nil, helper.JsonError(c, fiber.StatusBadRequest, "marked_le invalid format, expected YYYY-MM-DD")
		}
		tx = tx.Where(`
			class_attendance_session_participant_marked_at IS NOT NULL
			AND class_attendance_session_participant_marked_at < ?
		`, t.Add(24*time.Hour))
	}

	// default order
	return tx.Order("class_attendance_session_participant_created_at DESC"), nil
}

// ================= LIST & GET BY ID =================

func (ctl *StudentAttendanceController) List(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	// ✅ Resolve school context
	mc, er := helperAuth.ResolveSchoolContext(c)
	if er != nil {
		if fe, ok := er.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, er.Error())
	}

	// ✅ Tentukan schoolID + authorize
	var schoolID uuid.UUID
	if helperAuth.IsOwner(c) || helperAuth.IsDKM(c) {
		id, er := helperAuth.EnsureSchoolAccessDKM(c, mc)
		if er != nil {
			if fe, ok := er.(*fiber.Error); ok {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return helper.JsonError(c, fiber.StatusForbidden, er.Error())
		}
		schoolID = id
	} else {
		switch {
		case mc.ID != uuid.Nil:
			schoolID = mc.ID
		case strings.TrimSpace(mc.Slug) != "":
			id, er := helperAuth.GetSchoolIDBySlug(c, mc.Slug)
			if er != nil {
				return helper.JsonError(c, fiber.StatusNotFound, "School (slug) tidak ditemukan")
			}
			schoolID = id
		default:
			if id, er := helperAuth.GetActiveSchoolID(c); er == nil && id != uuid.Nil {
				schoolID = id
			}
		}
		if schoolID == uuid.Nil || !helperAuth.UserHasSchool(c, schoolID) {
			return helper.JsonError(c, fiber.StatusForbidden, "Anda tidak terdaftar pada school ini (membership).")
		}
	}

	// include flags (nantinya untuk join URL kalau perlu)
	includeParam := strings.ToLower(strings.TrimSpace(c.Query("include")))
	includeURLs := includeParam == "urls"
	if !includeURLs {
		legacy := strings.ToLower(strings.TrimSpace(c.Query("include_urls")))
		includeURLs = legacy == "1" || legacy == "true" || legacy == "yes"
	}

	// --- GET BY ID mode ---
	rawID := strings.TrimSpace(c.Params("id"))
	if rawID == "" {
		rawID = strings.TrimSpace(c.Query("id"))
	}
	if rawID != "" {
		id, perr := uuid.Parse(rawID)
		if perr != nil || id == uuid.Nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "id tidak valid")
		}

		var m attModel.ClassAttendanceSessionParticipantModel
		if err := ctl.DB.WithContext(c.Context()).
			Where(`
				class_attendance_session_participant_id = ?
				AND class_attendance_session_participant_school_id = ?
				AND class_attendance_session_participant_deleted_at IS NULL
			`, id, schoolID).
			First(&m).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
		}

		if !includeURLs {
			return helper.JsonOK(c, "OK", m)
		}
		return helper.JsonOK(c, "OK", fiber.Map{
			"attendance": m,
			// "urls": urls, // bisa diisi nanti dengan query ke URL table
		})
	}

	// --- LIST mode ---

	// ✅ Pagination (jsonresponse)
	p := helper.ResolvePaging(c, 20, 200)

	// ✅ Sorting whitelist
	sortBy := strings.ToLower(strings.TrimSpace(c.Query("sort_by")))
	order := strings.ToLower(strings.TrimSpace(c.Query("order")))
	if order != "asc" && order != "desc" {
		// fallback kompatibel dengan param "sort" lama
		if v := strings.ToLower(strings.TrimSpace(c.Query("sort"))); v != "" {
			switch v {
			case "created_at_asc":
				sortBy, order = "created_at", "asc"
			case "created_at_desc":
				sortBy, order = "created_at", "desc"
			case "id_asc":
				sortBy, order = "id", "asc"
			case "id_desc":
				sortBy, order = "id", "desc"
			default:
				order = "desc"
			}
		} else {
			order = "desc"
		}
	}
	col := "class_attendance_session_participant_created_at"
	switch sortBy {
	case "id":
		col = "class_attendance_session_participant_id"
	case "created_at", "":
		col = "class_attendance_session_participant_created_at"
	}
	orderExpr := col + " " + strings.ToUpper(order)

	// ✅ Parse query → DTO
	var q attDTO.ListClassAttendanceSessionParticipantQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}

	// ✅ Build base query (tenant-aware)
	tx, err := ctl.buildListQuery(c, q, schoolID)
	if err != nil {
		return err
	}

	// filter ids (opsional)
	if raw := strings.TrimSpace(c.Query("ids")); raw != "" {
		parts := strings.Split(raw, ",")
		ids := make([]uuid.UUID, 0, len(parts))
		for _, s := range parts {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			u, e := uuid.Parse(s)
			if e != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "id tidak valid: "+s)
			}
			ids = append(ids, u)
		}
		if len(ids) > 0 {
			tx = tx.Where("class_attendance_session_participant_id IN ?", ids)
		}
	}

	// ✅ Total
	var total int64
	if err := tx.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// ✅ Page window + order
	qdb := tx.Order(orderExpr).Limit(p.Limit).Offset(p.Offset)

	var rows []attModel.ClassAttendanceSessionParticipantModel
	if err := qdb.Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// ✅ Pagination object (jsonresponse)
	pg := helper.BuildPaginationFromOffset(total, p.Offset, p.Limit)

	// ✅ Response (tanpa/ dengan URLs)
	if !includeURLs {
		return helper.JsonList(c, "ok", rows, pg)
	}

	items := make([]fiber.Map, 0, len(rows))
	for i := range rows {
		items = append(items, fiber.Map{
			"attendance": rows[i],
			// "urls": urlsByParticipant[rows[i].ClassAttendanceSessionParticipantID],
		})
	}
	return helper.JsonList(c, "ok", items, pg)
}

/*
=================================================================================
CREATE — POST /student-attendance
Body: attDTO.ClassAttendanceSessionParticipantCreateRequest
- URLs: []attDTO.ClassAttendanceSessionParticipantURLOpDTO (di dalam DTO)
=================================================================================
*/
func (ctl *StudentAttendanceController) Create(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	// Guard role (owner/DKM/teacher)
	if !(helperAuth.IsOwner(c) || helperAuth.IsDKM(c) || helperAuth.IsTeacher(c)) {
		return helper.JsonError(c, fiber.StatusForbidden, "Akses ditolak")
	}

	// Resolve school context (mirip List)
	mc, er := helperAuth.ResolveSchoolContext(c)
	if er != nil {
		if fe, ok := er.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, er.Error())
	}

	var schoolID uuid.UUID
	if helperAuth.IsOwner(c) || helperAuth.IsDKM(c) {
		id, er := helperAuth.EnsureSchoolAccessDKM(c, mc)
		if er != nil {
			if fe, ok := er.(*fiber.Error); ok {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return helper.JsonError(c, fiber.StatusForbidden, er.Error())
		}
		schoolID = id
	} else {
		switch {
		case mc.ID != uuid.Nil:
			schoolID = mc.ID
		case strings.TrimSpace(mc.Slug) != "":
			id, er := helperAuth.GetSchoolIDBySlug(c, mc.Slug)
			if er != nil {
				return helper.JsonError(c, fiber.StatusNotFound, "School (slug) tidak ditemukan")
			}
			schoolID = id
		default:
			if id, er := helperAuth.GetActiveSchoolID(c); er == nil && id != uuid.Nil {
				schoolID = id
			}
		}
		if schoolID == uuid.Nil || !helperAuth.UserHasSchool(c, schoolID) {
			return helper.JsonError(c, fiber.StatusForbidden, "Anda tidak terdaftar pada school ini (membership).")
		}
	}

	var req attDTO.ClassAttendanceSessionParticipantCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid: "+err.Error())
	}

	// paksa tenant dari context
	req.SchoolID = schoolID

	// Basic required fields
	if req.SessionID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "session_id wajib diisi")
	}
	// minimal salah satu participant (student/teacher)
	hasStudent := req.SchoolStudentID != nil && *req.SchoolStudentID != uuid.Nil
	hasTeacher := req.SchoolTeacherID != nil && *req.SchoolTeacherID != uuid.Nil
	if !hasStudent && !hasTeacher {
		return helper.JsonError(c, fiber.StatusBadRequest, "Minimal salah satu dari school_student_id atau school_teacher_id wajib diisi")
	}

	// Validasi dengan validator di DTO (kalau controller punya)
	if ctl.Validator != nil {
		if err := ctl.Validator.Struct(&req); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
		}
	}

	// Pastikan session milik school ini (tenant-safe)
	if err := ctl.ensureSessionBelongsToSchool(c, req.SessionID, schoolID); err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Default-kan op URL ke upsert kalau kosong
	for i := range req.URLs {
		if req.URLs[i].Op == "" {
			req.URLs[i].Op = attDTO.URLOpUpsert
		}
	}

	var out attModel.ClassAttendanceSessionParticipantModel

	if err := ctl.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		// 1) create participant
		m := req.ToModel()
		if err := tx.Create(&m).Error; err != nil {
			if isDuplicateKey(err) {
				return fiber.NewError(fiber.StatusConflict, "Kehadiran sudah tercatat (duplikat)")
			}
			return err
		}

		// 2) URL create (gunakan DTO URLMutations baru)
		if len(req.URLs) > 0 {
			muts, err := attDTO.BuildURLMutations(m.ClassAttendanceSessionParticipantID, req.SchoolID, req.URLs)
			if err != nil {
				return err
			}
			if len(muts.ToCreate) > 0 {
				if err := tx.Create(&muts.ToCreate).Error; err != nil {
					return err
				}
			}
			// 3) enforce primary unik per (participant, kind)
			if err := ensurePrimaryUnique(tx, m.ClassAttendanceSessionParticipantID); err != nil {
				return err
			}
		}

		out = m
		return nil
	}); err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonCreated(c, "Created", out)
}
