// file: internals/features/school/academics/classes/controller/student_class_enrollment_list_controller.go
package controller

import (
	"context"
	"reflect"
	"strings"

	dto "madinahsalam_backend/internals/features/school/classes/classes/dto"
	emodel "madinahsalam_backend/internals/features/school/classes/classes/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	csstDTO "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/dto"
	csstModel "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/model"
	csDTO "madinahsalam_backend/internals/features/school/classes/class_sections/dto"
	csModel "madinahsalam_backend/internals/features/school/classes/class_sections/model"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (ctl *StudentClassEnrollmentController) List(c *fiber.Ctx) error {
	// ========== tenant dari TOKEN (bukan dari path) ==========
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err // helper sudah balikin JsonError
	}

	// ❗ hanya DKM/Admin (boleh tambah bendahara kalau mau)
	if er := helperAuth.EnsureDKMSchool(c, schoolID); er != nil {
		return er
	}

	// ========== special case: student_id=me ==========
	rawStudentID := strings.TrimSpace(c.Query("student_id"))
	isMe := strings.EqualFold(rawStudentID, "me")

	if isMe {
		// Hapus dulu dari query supaya QueryParser nggak gagal parse "me" ke UUID
		c.Request().URI().QueryArgs().Del("student_id")
	}

	// ========== query (struct) ==========
	var q dto.ListStudentClassEnrollmentQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid query")
	}

	// Kalau student_id=me → ambil dari token & override q.StudentID
	if isMe {
		sid, er := helperAuth.GetPrimarySchoolStudentID(c)
		if er != nil || sid == uuid.Nil {
			return helper.JsonError(c, fiber.StatusUnauthorized, "student_id (me) tidak ditemukan di token")
		}
		q.StudentID = &sid
	}

	// ========== explicit filter id dari query ==========
	idStr := strings.TrimSpace(c.Query("id"))
	var rowID uuid.UUID
	if idStr != "" {
		v, er := uuid.Parse(idStr)
		if er != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "id invalid")
		}
		rowID = v
	}

	// status_in (comma-separated → slice) — override ke q.StatusIn
	if raw := strings.TrimSpace(c.Query("status_in")); raw != "" {
		sts, er := parseStatusInParam(raw)
		if er != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, er.Error())
		}
		q.StatusIn = sts
	}

	// ====== CATEGORY filter (registration / spp / dll) ======
	category := strings.TrimSpace(c.Query("category"))

	// ====== PAYMENT STATUS filter (paid / pending / dll) ======
	paymentStatus := strings.ToLower(strings.TrimSpace(c.Query("payment_status")))

	// view mode
	view := strings.ToLower(strings.TrimSpace(c.Query("view"))) // "", "compact", "summary", "full"

	// include mode (?include=class_sections,csst)
	includeRaw := strings.ToLower(strings.TrimSpace(c.Query("include")))
	wantClassSections := false
	wantCSST := false
	if includeRaw != "" {
		for _, p := range strings.Split(includeRaw, ",") {
			switch strings.TrimSpace(p) {
			case "class_sections":
				wantClassSections = true
			case "csst", "cssts", "class_section_subject_teachers":
				wantCSST = true
			}
		}
	}
	// kalau minta CSST otomatis butuh class_sections juga
	if wantCSST {
		wantClassSections = true
	}

	// paging (masih pakai helper page/per_page)
	pg := helper.ResolvePaging(c, 20, 200)

	// ========== base query ==========
	base := ctl.DB.WithContext(c.Context()).
		Model(&emodel.StudentClassEnrollmentModel{}).
		Where("student_class_enrollments_school_id = ?", schoolID)

	// OnlyAlive default: true (filter soft-delete)
	onlyAlive := true
	if q.OnlyAlive != nil {
		onlyAlive = *q.OnlyAlive
	}
	if onlyAlive {
		base = base.Where("student_class_enrollments_deleted_at IS NULL")
	}

	// ========== filters ==========
	// filter by primary key (id)
	if rowID != uuid.Nil {
		base = base.Where("student_class_enrollments_id = ?", rowID)
	}

	// filter by student_id (bisa dari query biasa, bisa dari "me")
	if q.StudentID != nil && *q.StudentID != uuid.Nil {
		base = base.Where("student_class_enrollments_school_student_id = ?", *q.StudentID)
	}

	// filter by class_id (dari DTO)
	if q.ClassID != nil && *q.ClassID != uuid.Nil {
		base = base.Where("student_class_enrollments_class_id = ?", *q.ClassID)
	}

	// filter by status_in
	if len(q.StatusIn) > 0 {
		base = base.Where("student_class_enrollments_status IN ?", q.StatusIn)
	}

	// filter applied_from / applied_to (kalau diisi)
	if q.AppliedFrom != nil {
		base = base.Where("student_class_enrollments_applied_at >= ?", *q.AppliedFrom)
	}
	if q.AppliedTo != nil {
		base = base.Where("student_class_enrollments_applied_at <= ?", *q.AppliedTo)
	}

	// ===== TERM FILTERS =====
	// Prioritas: academic_term_id (baru), kalau kosong pakai term_id (legacy)
	var termID *uuid.UUID
	if q.AcademicTermID != nil && *q.AcademicTermID != uuid.Nil {
		termID = q.AcademicTermID
	} else if q.TermID != nil && *q.TermID != uuid.Nil {
		termID = q.TermID
	}

	if termID != nil {
		base = base.Where("student_class_enrollments_term_id = ?", *termID)
	}

	if strings.TrimSpace(q.AcademicYear) != "" {
		base = base.Where(
			"student_class_enrollments_term_academic_year_cache = ?",
			strings.TrimSpace(q.AcademicYear),
		)
	}
	if q.Angkatan != nil {
		base = base.Where("student_class_enrollments_term_angkatan_cache = ?", *q.Angkatan)
	}

	// ===== CATEGORY filter (JSONB) =====
	if category != "" {
		base = base.Where(`
			(student_class_enrollments_payment_snapshot->'payment_meta'->>'fee_rule_gbk_category_snapshot' = ?
			 OR student_class_enrollments_preferences->'registration'->>'category_snapshot' = ?)
		`, category, category)
	}

	// ===== PAYMENT STATUS filter (JSONB) =====
	if paymentStatus != "" {
		// contoh: ?payment_status=paid → hanya yang payment_status = 'paid' di snapshot
		base = base.Where(`
			LOWER(student_class_enrollments_payment_snapshot->>'payment_status') = ?
		`, paymentStatus)
	}

	// ===== Q search (nama siswa / nama kelas / nama term) =====
	if strings.TrimSpace(q.Q) != "" {
		pat := "%" + strings.TrimSpace(q.Q) + "%"
		base = base.Where(`
			student_class_enrollments_user_profile_name_cache ILIKE ?
			OR student_class_enrollments_class_name_cache ILIKE ?
			OR COALESCE(student_class_enrollments_term_name_cache, '') ILIKE ?
		`, pat, pat, pat)
	}

	// ========== count ==========
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to count")
	}

	// ========== data ==========
	tx := base

	// optimisasi kolom saat compact/summary
	if view == "compact" || view == "summary" {
		tx = tx.Select([]string{
			// id & status & nominal
			"student_class_enrollments_id",
			"student_class_enrollments_status",
			"student_class_enrollments_total_due_idr",

			// convenience (mirror cache & ids)
			"student_class_enrollments_school_student_id",
			"student_class_enrollments_user_profile_name_cache",
			"student_class_enrollments_class_id",
			"student_class_enrollments_class_name_cache",
			"student_class_enrollments_class_slug_cache",

			// CACHE MURID LENGKAP
			"student_class_enrollments_user_profile_avatar_url_cache",
			"student_class_enrollments_user_profile_whatsapp_url_cache",
			"student_class_enrollments_user_profile_parent_name_cache",
			"student_class_enrollments_user_profile_parent_whatsapp_url_cache",
			"student_class_enrollments_user_profile_gender_cache",
			"student_class_enrollments_student_code_cache",
			"student_class_enrollments_student_slug_cache",

			// term (denormalized, optional; cache)
			"student_class_enrollments_term_id",
			"student_class_enrollments_term_name_cache",
			"student_class_enrollments_term_academic_year_cache",
			"student_class_enrollments_term_angkatan_cache",
			"student_class_enrollments_term_slug_cache",

			// CLASS SECTION (optional; cache)
			"student_class_enrollments_class_section_id",
			"student_class_enrollments_class_section_name_cache",
			"student_class_enrollments_class_section_slug_cache",

			// payment snapshot + preferences (JSONB)
			"student_class_enrollments_payment_snapshot",
			"student_class_enrollments_preferences",

			// jejak penting
			"student_class_enrollments_applied_at",
		})
	}

	var rows []emodel.StudentClassEnrollmentModel
	if err := tx.
		Order(orderClause(q.OrderBy, q.Sort)).
		Offset(pg.Offset).
		Limit(pg.Limit).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to fetch")
	}

	pagination := helper.BuildPaginationFromOffset(total, pg.Offset, pg.Limit)

	// ========== mapping sesuai view ==========
	if view == "compact" || view == "summary" {
		compact := dto.FromModelsCompact(rows)
		// NOTE: include=class_sections/csst saat ini hanya dipakai di view=full
		return helper.JsonList(c, "ok", compact, pagination)
	}

	// default: full payload
	resp := dto.FromModels(rows)

	// (opsional) enrich convenience fields tambahan (Username, dll.)
	enrichEnrollmentExtras(c.Context(), ctl.DB, schoolID, resp)

	// (opsional) include class_sections (+csst)
	if wantClassSections {
		if err := enrichEnrollmentClassSections(c.Context(), ctl.DB, schoolID, resp, wantCSST); err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "failed to load class sections")
		}
	}

	return helper.JsonList(c, "ok", resp, pagination)
}

// include: class_sections (group by class_id)
// kalau withCSST = true → sekalian tempel array CSST per section
func enrichEnrollmentClassSections(
	ctx context.Context,
	db *gorm.DB,
	schoolID uuid.UUID,
	items any, // biar nggak tergantung nama tipe DTO
	withCSST bool,
) error {
	v := reflect.ValueOf(items)
	if v.Kind() != reflect.Slice {
		return nil
	}

	// 1) Kumpulkan distinct class_id dari field `ClassID`
	classSet := make(map[uuid.UUID]struct{})

	for i := 0; i < v.Len(); i++ {
		elem := v.Index(i)
		if elem.Kind() == reflect.Ptr {
			if elem.IsNil() {
				continue
			}
			elem = elem.Elem()
		}
		if !elem.IsValid() {
			continue
		}

		f := elem.FieldByName("ClassID")
		if !f.IsValid() || !f.CanInterface() {
			continue
		}

		cid, ok := f.Interface().(uuid.UUID)
		if !ok || cid == uuid.Nil {
			continue
		}
		classSet[cid] = struct{}{}
	}

	if len(classSet) == 0 {
		return nil
	}

	classIDs := make([]uuid.UUID, 0, len(classSet))
	for id := range classSet {
		classIDs = append(classIDs, id)
	}

	// 2) Query class_sections by class_id
	var secs []csModel.ClassSectionModel
	if err := db.WithContext(ctx).
		Model(&csModel.ClassSectionModel{}).
		Where("class_section_school_id = ?", schoolID).
		Where("class_section_class_id IN ?", classIDs).
		Where("class_section_deleted_at IS NULL").
		Order("class_section_name ASC").
		Find(&secs).Error; err != nil {
		return err
	}
	if len(secs) == 0 {
		return nil
	}

	// 3) Konversi ke DTO compact
	compact := csDTO.FromModelsClassSectionCompact(secs)

	// 4) Kalau diminta CSST, query CSST & group by section_id (pakai MODEL dulu)
	var csstBySection map[uuid.UUID][]csstModel.ClassSectionSubjectTeacherModel
	if withCSST {
		csstBySection = make(map[uuid.UUID][]csstModel.ClassSectionSubjectTeacherModel)

		sectionIDs := make([]uuid.UUID, 0, len(secs))
		for _, s := range secs {
			sectionIDs = append(sectionIDs, s.ClassSectionID)
		}

		var csstRows []csstModel.ClassSectionSubjectTeacherModel
		if err := db.WithContext(ctx).
			Model(&csstModel.ClassSectionSubjectTeacherModel{}).
			Where("class_section_subject_teacher_school_id = ?", schoolID).
			Where("class_section_subject_teacher_deleted_at IS NULL").
			Where("class_section_subject_teacher_class_section_id IN ?", sectionIDs).
			Find(&csstRows).Error; err != nil {
			return err
		}

		for _, r := range csstRows {
			secID := r.ClassSectionSubjectTeacherClassSectionID
			csstBySection[secID] = append(csstBySection[secID], r)
		}
	}

	// 5) Group per class_id (pakai ClassSectionClassID dari compact)
	byClass := make(map[uuid.UUID][]csDTO.ClassSectionCompact)
	for i := range compact {
		s := &compact[i]
		if s.ClassSectionClassID == nil || *s.ClassSectionClassID == uuid.Nil {
			continue
		}

		// kalau withCSST → tempel dulu ke field SubjectTeachers di compact
		if withCSST {
			// diasumsikan urutan compact sama dengan urutan secs
			secModel := secs[i]
			if list, ok := csstBySection[secModel.ClassSectionID]; ok && len(list) > 0 {
				// 🔴 FIX: convert []Model → []CSSTItemLite sebelum assign
				s.SubjectTeachers = csstDTO.CSSTLiteSliceFromModels(list)
			}
		}

		byClass[*s.ClassSectionClassID] = append(byClass[*s.ClassSectionClassID], *s)
	}

	// 6) Tempel ke tiap enrollment lewat field `ClassSections`
	sectionsSliceType := reflect.TypeOf([]csDTO.ClassSectionCompact{})

	for i := 0; i < v.Len(); i++ {
		elem := v.Index(i)
		if elem.Kind() == reflect.Ptr {
			if elem.IsNil() {
				continue
			}
			elem = elem.Elem()
		}
		if !elem.IsValid() {
			continue
		}

		fID := elem.FieldByName("ClassID")
		if !fID.IsValid() || !fID.CanInterface() {
			continue
		}
		cid, ok := fID.Interface().(uuid.UUID)
		if !ok || cid == uuid.Nil {
			continue
		}

		list, ok := byClass[cid]
		if !ok {
			continue
		}

		fSec := elem.FieldByName("ClassSections")
		if !fSec.IsValid() || !fSec.CanSet() {
			continue
		}
		if fSec.Type() != sectionsSliceType {
			// tipe field-nya beda → skip
			continue
		}

		fSec.Set(reflect.ValueOf(list))
	}

	return nil
}
