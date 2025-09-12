package controller

import (
	dto "masjidku_backend/internals/features/school/submissions_assesments/submissions/dto"
	model "masjidku_backend/internals/features/school/submissions_assesments/submissions/model"
	"net/http"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	helper "masjidku_backend/internals/helpers"
)

// List — filter + pagination helper (scope-guarded)
// - Jika TIDAK ada submission_id → otomatis batasi hasil ke submission yang berada di scope user
func (ctl *SubmissionUrlsController) List(c *fiber.Ctx) error {
	// Pagination (default: created_at DESC)
	p := helper.ParseWith(stdReqFromFiber(c), "created_at", "desc", helper.AdminOpts)

	// Whitelist kolom sorting
	allowedSort := map[string]string{
		"label":      "submission_urls_label",
		"href":       "submission_urls_href",
		"created_at": "submission_urls_created_at",
		"updated_at": "submission_urls_updated_at",
	}
	orderCol := allowedSort["created_at"]
	if col, ok := allowedSort[strings.ToLower(p.SortBy)]; ok {
		orderCol = col
	}
	orderDir := "DESC"
	if strings.ToLower(p.SortOrder) == "asc" {
		orderDir = "ASC"
	}

	submissionIDStr := strings.TrimSpace(c.Query("submission_id"))
	q := strings.TrimSpace(c.Query("q"))
	isActiveStr := strings.TrimSpace(c.Query("is_active"))

	studentIDs, masjidIDs, err := mustGetStudentAndMasjidScope(c)
	if err != nil {
		return helper.JsonError(c, http.StatusUnauthorized, err.Error())
	}

	db := ctl.DB.Model(&model.SubmissionUrlsModel{})

	// Scope filtering
	if submissionIDStr != "" {
		sid, err := uuid.Parse(submissionIDStr)
		if err != nil {
			return helper.JsonError(c, http.StatusBadRequest, "submission_id tidak valid")
		}
		if ok, err := assertSubmissionScope(ctl.DB, sid, studentIDs, masjidIDs); err != nil || !ok {
			return helper.JsonError(c, http.StatusForbidden, "Akses ditolak untuk submission tersebut")
		}
		db = db.Where("submission_urls_submission_id = ?", sid)
		} else {
			// Tanpa submission_id → batasi berdasar scope via subquery EXISTS + IN ?
			if len(studentIDs) == 0 && len(masjidIDs) == 0 {
				return helper.JsonError(c, http.StatusUnauthorized, "Scope tidak tersedia pada token")
			}

			subConds := []string{}
			args := []any{}

			if len(studentIDs) > 0 {
				subConds = append(subConds, "s.submissions_student_id IN ?")
				args = append(args, studentIDs)
			}
			if len(masjidIDs) > 0 {
				subConds = append(subConds, "s.submissions_masjid_id IN ?")
				args = append(args, masjidIDs)
			}

			whereSQL := "EXISTS (SELECT 1 FROM submissions s WHERE s.submissions_id = submission_urls_submission_id AND (" +
				strings.Join(subConds, " OR ") + "))"

			db = db.Where(whereSQL, args...)
		}


	if q != "" {
		like := "%" + q + "%"
		db = db.Where("(submission_urls_label ILIKE ? OR submission_urls_href ILIKE ?)", like, like)
	}

	if isActiveStr != "" {
		if v, err := strconv.ParseBool(isActiveStr); err == nil {
			db = db.Where("submission_urls_is_active = ?", v)
		} else {
			return helper.JsonError(c, http.StatusBadRequest, "is_active harus boolean")
		}
	}

	// Count total
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	// Sorting & pagination
	db = db.Order(orderCol + " " + orderDir)
	if !p.All {
		db = db.Limit(p.Limit()).Offset(p.Offset())
	}

	var rows []model.SubmissionUrlsModel
	if err := db.Find(&rows).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	out := make([]dto.SubmissionUrlResponse, 0, len(rows))
	for i := range rows {
		out = append(out, dto.ToSubmissionUrlResponse(&rows[i]))
	}

	meta := helper.BuildMeta(total, p)
	return helper.JsonList(c, out, meta)
}
