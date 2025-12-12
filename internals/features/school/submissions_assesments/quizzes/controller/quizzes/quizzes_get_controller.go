// file: internals/features/school/submissions_assesments/quizzes/controller/quiz_controller_list.go
package controller

import (
	"strings"

	assessmentdto "madinahsalam_backend/internals/features/school/submissions_assesments/assesments/dto"
	assessmentModel "madinahsalam_backend/internals/features/school/submissions_assesments/assesments/model"
	dto "madinahsalam_backend/internals/features/school/submissions_assesments/quizzes/dto"
	model "madinahsalam_backend/internals/features/school/submissions_assesments/quizzes/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// kecil-kecilan: parse CSV "a,b,c" → map[string]bool
func parseListParam(raw string) map[string]bool {
	out := make(map[string]bool)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return out
	}
	for _, part := range strings.Split(raw, ",") {
		k := strings.ToLower(strings.TrimSpace(part))
		if k != "" {
			out[k] = true
		}
	}
	return out
}

// GET /quizzes
// Catatan:
//   - Semua config + total soal (quiz_total_questions) sudah ikut ter-serialize
//     lewat dto.FromModelWithCtx.
//   - Support:
//   - include=assessment_type,quiz_questions
//   - nested=quiz_questions
//   - Param tambahan:
//   - recurring_type=false → include.assessment_types hanya 1 item saja
func (ctrl *QuizController) List(c *fiber.Ctx) error {
	// Inject DB buat helper slug→id & dbtime
	c.Locals("DB", ctrl.DB)

	// =====================================================
	// 1) Resolve school dari TOKEN dulu
	// =====================================================

	var schoolID uuid.UUID

	if sid, err := helperAuth.GetSchoolIDFromTokenPreferTeacher(c); err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	} else if sid != uuid.Nil {
		schoolID = sid
	} else {
		mc, err := helperAuth.ResolveSchoolContext(c)
		if err != nil {
			if fe, ok := err.(*fiber.Error); ok {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
		}

		switch {
		case mc.ID != uuid.Nil:
			schoolID = mc.ID

		case strings.TrimSpace(mc.Slug) != "":
			id, er := helperAuth.GetSchoolIDBySlug(c, strings.TrimSpace(mc.Slug))
			if er != nil || id == uuid.Nil {
				return helper.JsonError(c, fiber.StatusNotFound, "School (slug) tidak ditemukan")
			}
			schoolID = id

		default:
			return helper.JsonError(c, fiber.StatusBadRequest, "School context tidak ditemukan")
		}
	}

	// =====================================================
	// 2) Authorize: minimal member school
	// =====================================================
	if err := helperAuth.EnsureMemberSchool(c, schoolID); err != nil {
		return err
	}

	// =====================================================
	// 3) Query params + validation
	// =====================================================
	var q dto.ListQuizzesQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}

	// Tambahan safety: parse manual assessment_id (kalau dikirim string acak)
	if s := strings.TrimSpace(c.Query("assessment_id")); s != "" {
		aid, err := uuid.Parse(s)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "assessment_id tidak valid")
		}
		q.AssessmentID = &aid
	}

	if err := ctrl.Validator.Struct(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Force scope tenant
	q.SchoolID = &schoolID

	// =====================================================
	// 4) Include & nested flags (langsung dari query)
	// =====================================================
	includeFlags := parseListParam(c.Query("include"))
	nestedFlags := parseListParam(c.Query("nested"))

	// quiz_questions
	nestedQuestions := q.WithQuestions || nestedFlags["quiz_questions"]
	includeQuestions := includeFlags["quiz_questions"]

	// assessment_type → HANYA via include (tidak nested di tiap quiz)
	includeAssessmentType := includeFlags["assessment_type"]

	// recurring_type flag (khusus untuk assessment_types)
	// default: true → kirim semua jenis yang kepakai
	// recurring_type=false → cukup kirim 1 assessment_type saja
	recurringType := true
	if s := strings.ToLower(strings.TrimSpace(c.Query("recurring_type"))); s != "" {
		if s == "false" || s == "0" || s == "no" {
			recurringType = false
		}
	}

	// =====================================================
	// 5) Pagination
	// =====================================================
	p := helper.ResolvePaging(c, 20, 200)

	// =====================================================
	// 6) Base query + filters
	// =====================================================
	dbq := ctrl.DB.WithContext(c.Context()).Model(&model.QuizModel{})
	dbq = applyFiltersQuizzes(dbq, &q)

	// 7) Count
	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// 8) Sort + Page window
	dbq = applySort(dbq, q.Sort)
	if p.Limit > 0 {
		dbq = dbq.Offset(p.Offset).Limit(p.Limit)
	}

	// 9) Fetch quizzes
	var rows []model.QuizModel
	if err := dbq.Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// =====================================================
	// 10) Siapkan batch quiz_questions untuk nested / include
	// =====================================================
	var allQuestions []model.QuizQuestionModel
	questionsByQuizID := map[uuid.UUID][]model.QuizQuestionModel{}

	if (nestedQuestions || includeQuestions) && len(rows) > 0 {
		quizIDs := make([]uuid.UUID, 0, len(rows))
		for i := range rows {
			quizIDs = append(quizIDs, rows[i].QuizID)
		}

		if err := ctrl.DB.WithContext(c.Context()).
			Where("quiz_question_school_id = ?", schoolID).
			Where("quiz_question_quiz_id IN ?", quizIDs).
			Where("quiz_question_deleted_at IS NULL").
			Order("quiz_question_created_at ASC").
			Find(&allQuestions).Error; err != nil {

			return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
		}

		for _, qq := range allQuestions {
			qid := qq.QuizQuestionQuizID
			questionsByQuizID[qid] = append(questionsByQuizID[qid], qq)
		}
	}

	// 11) Siapkan lookup untuk assessment_type (include only)
	// ganti alias ke FULL DTO
	type AssessmentTypeDTO = assessmentdto.AssessmentTypeResponse

	assessmentDTOByID := map[uuid.UUID]AssessmentTypeDTO{}
	var assessmentList []AssessmentTypeDTO

	if includeAssessmentType {
		typeIDSet := map[uuid.UUID]struct{}{}
		for i := range rows {
			if rows[i].QuizAssessmentTypeID != nil && *rows[i].QuizAssessmentTypeID != uuid.Nil {
				typeIDSet[*rows[i].QuizAssessmentTypeID] = struct{}{}
			}
		}

		if len(typeIDSet) > 0 {
			typeIDs := make([]uuid.UUID, 0, len(typeIDSet))
			for id := range typeIDSet {
				typeIDs = append(typeIDs, id)
			}

			var typeRows []assessmentModel.AssessmentTypeModel
			if err := ctrl.DB.WithContext(c.Context()).
				Where("assessment_type_school_id = ?", schoolID).
				Where("assessment_type_id IN ?", typeIDs).
				Where("assessment_type_deleted_at IS NULL").
				Find(&typeRows).Error; err != nil {

				return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
			}

			// ⬇️ di sini pakai mapper FULL, bukan Compact
			// kalau di package-mu namanya beda (misal NewAssessmentTypeDTOsWithSchoolTime),
			// tinggal sesuaikan pemanggilannya
			assessmentList = assessmentdto.FromModels(typeRows)

			for _, at := range assessmentList {
				assessmentDTOByID[at.AssessmentTypeID] = at
			}
		}
	}

	// =====================================================
	// 12) Build quiz DTO (timezone-aware + nested questions)
	// =====================================================
	out := make([]dto.QuizResponse, 0, len(rows))

	for i := range rows {
		resp := dto.FromModelWithCtx(c, &rows[i])

		// WithQuestionsCount → pakai denorm quiz_total_questions
		if q.WithQuestionsCount {
			n := rows[i].QuizTotalQuestions
			resp.QuestionsCount = &n
		}

		// ===========================
		// Hitung jumlah soal AKTUAL
		// ===========================
		actualQuestionCount := 0

		// Kalau sebelumnya kita sudah fetch questions (nestedQuestions || includeQuestions),
		// pasti questionsByQuizID sudah keisi → ambil len dari sana.
		if qs, ok := questionsByQuizID[rows[i].QuizID]; ok {
			actualQuestionCount = len(qs)

			// kalau memang NESTED diminta, baru tempel questions ke resp
			if nestedQuestions && actualQuestionCount > 0 {
				resp.Questions = dto.FromModelsQuizQuestionsWithCtx(c, qs)
			}
		}

		// fallback kalau tidak ada fetch questions sama sekali
		if actualQuestionCount == 0 {
			actualQuestionCount = rows[i].QuizTotalQuestions
		}

		// ✅ quiz_time_limit_sec_all
		if resp.QuizTimeLimitSec != nil && actualQuestionCount > 0 {
			v := (*resp.QuizTimeLimitSec) * actualQuestionCount
			resp.QuizTimeLimitSecAll = &v
		}

		out = append(out, resp)
	}

	// =====================================================
	// 13) Build include object
	// =====================================================
	include := fiber.Map{}

	// include.assessment_types
	if includeAssessmentType && len(assessmentList) > 0 {
		atList := assessmentList

		// Kalau recurring_type=false → cukup kirim satu assessment_type saja
		if !recurringType {
			// Coba pilih berdasarkan quiz pertama di halaman (lebih “relevan”)
			if len(rows) > 0 && rows[0].QuizAssessmentTypeID != nil {
				if at, ok := assessmentDTOByID[*rows[0].QuizAssessmentTypeID]; ok {
					atList = []AssessmentTypeDTO{at}
				}
			} else {
				// fallback: potong jadi 1 aja
				atList = atList[:1]
			}
		}

		include["assessment_types"] = atList
	}

	// include.quiz_questions → pakai semua allQuestions (normal)
	if includeQuestions && len(allQuestions) > 0 {
		include["quiz_questions"] = dto.FromModelsQuizQuestionsWithCtx(c, allQuestions)
	}

	// =====================================================
	// 14) Response (pakai helper JSON standard)
	// =====================================================
	pg := helper.BuildPaginationFromOffset(total, p.Offset, p.Limit)

	// selalu pakai JsonListWithInclude biar shape:
	// { success, message, data, include: {}, pagination: {...} }
	return helper.JsonListWithInclude(c, "ok", out, include, pg)
}
