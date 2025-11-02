// controllers/user_subject_summary_controller_fiber.go
package controllers

// import (
// 	"strconv"
// 	"strings"
// 	"time"

// 	"github.com/go-playground/validator/v10"
// 	"github.com/gofiber/fiber/v2"
// 	"github.com/google/uuid"
// 	"gorm.io/gorm"

// 	"schoolku_backend/internals/features/school/academics/certificates/dto"
// 	models "schoolku_backend/internals/features/school/academics/certificates/model" // sesuaikan bila path model berbeda

// 	helper "schoolku_backend/internals/helpers"
// )

// type UserSubjectSummaryController struct {
// 	DB        *gorm.DB
// 	Validator *validator.Validate
// }

// func NewUserSubjectSummaryController(db *gorm.DB) *UserSubjectSummaryController {
// 	return &UserSubjectSummaryController{
// 		DB:        db,
// 		Validator: validator.New(),
// 	}
// }

// // ---------------- Helpers ----------------

// func parseUUIDParam(c *fiber.Ctx, key string) (uuid.UUID, error) {
// 	idStr := c.Params(key)
// 	return uuid.Parse(idStr)
// }

// func parseUUIDQuery(c *fiber.Ctx, key string) (*uuid.UUID, error) {
// 	val := strings.TrimSpace(c.Query(key))
// 	if val == "" {
// 		return nil, nil
// 	}
// 	u, err := uuid.Parse(val)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &u, nil
// }

// func parseBoolQuery(c *fiber.Ctx, key string) (*bool, error) {
// 	val := strings.TrimSpace(c.Query(key))
// 	if val == "" {
// 		return nil, nil
// 	}
// 	b, err := strconv.ParseBool(val)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &b, nil
// }

// func parseFloatQuery(c *fiber.Ctx, key string) (*float64, error) {
// 	val := strings.TrimSpace(c.Query(key))
// 	if val == "" {
// 		return nil, nil
// 	}
// 	f, err := strconv.ParseFloat(val, 64)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &f, nil
// }

// func parseIntWithDefault(c *fiber.Ctx, key string, def int) int {
// 	v := strings.TrimSpace(c.Query(key))
// 	if v == "" {
// 		return def
// 	}
// 	i, err := strconv.Atoi(v)
// 	if err != nil || i <= 0 {
// 		return def
// 	}
// 	return i
// }

// // whitelist order by untuk cegah SQL injection
// func sanitizeOrderBy(in string) string {
// 	if in == "" {
// 		return "user_subject_summary_updated_at DESC"
// 	}
// 	s := strings.ToLower(strings.TrimSpace(in))
// 	allowed := map[string]string{
// 		"created_at":  "user_subject_summary_created_at",
// 		"updated_at":  "user_subject_summary_updated_at",
// 		"final_score": "user_subject_summary_final_score",
// 		"passed":      "user_subject_summary_passed",
// 	}
// 	parts := strings.Split(s, ",")
// 	out := make([]string, 0, len(parts))
// 	for _, p := range parts {
// 		p = strings.TrimSpace(p)
// 		if p == "" {
// 			continue
// 		}
// 		dir := "ASC"
// 		if strings.HasSuffix(p, " desc") {
// 			dir = "DESC"
// 			p = strings.TrimSpace(strings.TrimSuffix(p, " desc"))
// 		} else if strings.HasSuffix(p, " asc") {
// 			dir = "ASC"
// 			p = strings.TrimSpace(strings.TrimSuffix(p, " asc"))
// 		}
// 		if col, ok := allowed[p]; ok {
// 			out = append(out, col+" "+dir)
// 		}
// 	}
// 	if len(out) == 0 {
// 		return "user_subject_summary_updated_at DESC"
// 	}
// 	return strings.Join(out, ", ")
// }

// // ---------------- Handlers ----------------

// // GET /api/user-subject-summaries
// func (ctl *UserSubjectSummaryController) List(c *fiber.Ctx) error {
// 	var q dto.ListUserSubjectSummaryFilter

// 	// required: school_id
// 	schoolStr := strings.TrimSpace(c.Query("school_id"))
// 	if schoolStr == "" {
// 		return helper.JsonError(c, fiber.StatusBadRequest, "school_id is required")
// 	}
// 	schoolID, err := uuid.Parse(schoolStr)
// 	if err != nil {
// 		return helper.JsonError(c, fiber.StatusBadRequest, "invalid school_id")
// 	}
// 	q.SchoolID = schoolID

// 	// optional queries
// 	if q.StudentID, err = parseUUIDQuery(c, "student_id"); err != nil {
// 		return helper.JsonError(c, fiber.StatusBadRequest, "invalid student_id")
// 	}
// 	if q.ClassSubjectsID, err = parseUUIDQuery(c, "class_subjects_id"); err != nil {
// 		return helper.JsonError(c, fiber.StatusBadRequest, "invalid class_subjects_id")
// 	}
// 	if q.TermID, err = parseUUIDQuery(c, "term_id"); err != nil {
// 		return helper.JsonError(c, fiber.StatusBadRequest, "invalid term_id")
// 	}
// 	if q.Passed, err = parseBoolQuery(c, "passed"); err != nil {
// 		return helper.JsonError(c, fiber.StatusBadRequest, "invalid passed")
// 	}
// 	if q.MinScore, err = parseFloatQuery(c, "min_score"); err != nil {
// 		return helper.JsonError(c, fiber.StatusBadRequest, "invalid min_score")
// 	}
// 	if q.MaxScore, err = parseFloatQuery(c, "max_score"); err != nil {
// 		return helper.JsonError(c, fiber.StatusBadRequest, "invalid max_score")
// 	}
// 	q.Page = parseIntWithDefault(c, "page", 1)
// 	q.PageSize = parseIntWithDefault(c, "page_size", 20)
// 	q.OrderBy = c.Query("order_by")

// 	// validate
// 	if err := ctl.Validator.Struct(q); err != nil {
// 		return helper.JsonError(c, fiber.StatusBadRequest, "validation error: "+err.Error())
// 	}

// 	db := ctl.DB.Model(&models.UserSubjectSummary{}).
// 		Where("user_subject_summary_deleted_at IS NULL").
// 		Where("user_subject_summary_school_id = ?", q.SchoolID)

// 	if q.StudentID != nil {
// 		db = db.Where("user_subject_summary_school_student_id = ?", *q.StudentID)
// 	}
// 	if q.ClassSubjectsID != nil {
// 		db = db.Where("user_subject_summary_class_subjects_id = ?", *q.ClassSubjectsID)
// 	}
// 	if q.TermID != nil {
// 		db = db.Where("user_subject_summary_term_id = ?", *q.TermID)
// 	}
// 	if q.Passed != nil {
// 		db = db.Where("user_subject_summary_passed = ?", *q.Passed)
// 	}
// 	if q.MinScore != nil {
// 		db = db.Where("user_subject_summary_final_score >= ?", *q.MinScore)
// 	}
// 	if q.MaxScore != nil {
// 		db = db.Where("user_subject_summary_final_score <= ?", *q.MaxScore)
// 	}

// 	var total int64
// 	if err := db.Count(&total).Error; err != nil {
// 		return helper.JsonError(c, fiber.StatusInternalServerError, "count failed: "+err.Error())
// 	}

// 	orderBy := sanitizeOrderBy(q.OrderBy)
// 	offset := (q.Page - 1) * q.PageSize

// 	var rows []models.UserSubjectSummary
// 	if err := db.Order(orderBy).Limit(q.PageSize).Offset(offset).Find(&rows).Error; err != nil {
// 		return helper.JsonError(c, fiber.StatusInternalServerError, "query failed: "+err.Error())
// 	}

// 	items := make([]dto.UserSubjectSummaryResponse, 0, len(rows))
// 	for _, m := range rows {
// 		items = append(items, dto.FromModelUserSubjectSummary(m))
// 	}
// 	totalPages := int((total + int64(q.PageSize) - 1) / int64(q.PageSize))

// 	pagination := fiber.Map{
// 		"total":       total,
// 		"page":        q.Page,
// 		"page_size":   q.PageSize,
// 		"total_pages": totalPages,
// 		"order_by":    orderBy,
// 	}
// 	return helper.JsonList(c, items, pagination)
// }

// // POST /api/user-subject-summaries
// func (ctl *UserSubjectSummaryController) Create(c *fiber.Ctx) error {
// 	var body dto.CreateUserSubjectSummaryDTO
// 	if err := c.BodyParser(&body); err != nil {
// 		return helper.JsonError(c, fiber.StatusBadRequest, "invalid json: "+err.Error())
// 	}
// 	if err := ctl.Validator.Struct(body); err != nil {
// 		return helper.JsonError(c, fiber.StatusBadRequest, "validation error: "+err.Error())
// 	}

// 	m := body.ToModel()
// 	if err := ctl.DB.Create(&m).Error; err != nil {
// 		return helper.JsonError(c, fiber.StatusInternalServerError, "insert failed: "+err.Error())
// 	}

// 	return helper.JsonCreated(c, "created", dto.FromModelUserSubjectSummary(m))
// }

// // PATCH /api/user-subject-summaries/:id
// func (ctl *UserSubjectSummaryController) Update(c *fiber.Ctx) error {
// 	id, err := parseUUIDParam(c, "id")
// 	if err != nil {
// 		return helper.JsonError(c, fiber.StatusBadRequest, "invalid id")
// 	}

// 	var body dto.UpdateUserSubjectSummaryDTO
// 	if err := c.BodyParser(&body); err != nil {
// 		return helper.JsonError(c, fiber.StatusBadRequest, "invalid json: "+err.Error())
// 	}
// 	if err := ctl.Validator.Struct(body); err != nil {
// 		return helper.JsonError(c, fiber.StatusBadRequest, "validation error: "+err.Error())
// 	}

// 	var m models.UserSubjectSummary
// 	if err := ctl.DB.Where("user_subject_summary_id = ? AND user_subject_summary_deleted_at IS NULL", id).
// 		First(&m).Error; err != nil {
// 		if err == gorm.ErrRecordNotFound {
// 			return helper.JsonError(c, fiber.StatusNotFound, "not found")
// 		}
// 		return helper.JsonError(c, fiber.StatusInternalServerError, "query failed: "+err.Error())
// 	}

// 	body.PatchModel(&m)
// 	m.UserSubjectSummaryUpdatedAt = time.Now()

// 	if err := ctl.DB.Save(&m).Error; err != nil {
// 		return helper.JsonError(c, fiber.StatusInternalServerError, "update failed: "+err.Error())
// 	}

// 	return helper.JsonUpdated(c, "updated", dto.FromModelUserSubjectSummary(m))
// }

// // DELETE /api/user-subject-summaries/:id (soft delete)
// func (ctl *UserSubjectSummaryController) Delete(c *fiber.Ctx) error {
// 	id, err := parseUUIDParam(c, "id")
// 	if err != nil {
// 		return helper.JsonError(c, fiber.StatusBadRequest, "invalid id")
// 	}
// 	now := time.Now()

// 	res := ctl.DB.Model(&models.UserSubjectSummary{}).
// 		Where("user_subject_summary_id = ? AND user_subject_summary_deleted_at IS NULL", id).
// 		Updates(map[string]any{
// 			"user_subject_summary_deleted_at": now,
// 			"user_subject_summary_updated_at": now,
// 		})

// 	if res.Error != nil {
// 		return helper.JsonError(c, fiber.StatusInternalServerError, "delete failed: "+res.Error.Error())
// 	}
// 	if res.RowsAffected == 0 {
// 		return helper.JsonError(c, fiber.StatusNotFound, "not found or already deleted")
// 	}

// 	return helper.JsonDeleted(c, "deleted", fiber.Map{"id": id})
// }

// // POST /api/user-subject-summaries/:id/restore
// func (ctl *UserSubjectSummaryController) Restore(c *fiber.Ctx) error {
// 	id, err := parseUUIDParam(c, "id")
// 	if err != nil {
// 		return helper.JsonError(c, fiber.StatusBadRequest, "invalid id")
// 	}
// 	now := time.Now()

// 	res := ctl.DB.Model(&models.UserSubjectSummary{}).
// 		Where("user_subject_summary_id = ? AND user_subject_summary_deleted_at IS NOT NULL", id).
// 		Updates(map[string]any{
// 			"user_subject_summary_deleted_at": nil,
// 			"user_subject_summary_updated_at": now,
// 		})

// 	if res.Error != nil {
// 		return helper.JsonError(c, fiber.StatusInternalServerError, "restore failed: "+res.Error.Error())
// 	}
// 	if res.RowsAffected == 0 {
// 		return helper.JsonError(c, fiber.StatusNotFound, "not found or not deleted")
// 	}

// 	var m models.UserSubjectSummary
// 	if err := ctl.DB.Where("user_subject_summary_id = ?", id).First(&m).Error; err != nil {
// 		// kalau gagal ambil objeknya, balikin flag restored saja
// 		return helper.JsonOK(c, "restored", fiber.Map{"id": id})
// 	}
// 	return helper.JsonOK(c, "restored", dto.FromModelUserSubjectSummary(m))
// }
