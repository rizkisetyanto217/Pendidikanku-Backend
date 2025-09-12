package controller

import (
	dto "masjidku_backend/internals/features/school/submissions_assesments/assesments/dto"
	model "masjidku_backend/internals/features/school/submissions_assesments/assesments/model"
	helperAuth "masjidku_backend/internals/helpers/auth"
	"strings"

	helper "masjidku_backend/internals/helpers"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)


func (ctl *AssessmentUrlsController) List(c *fiber.Ctx) error {
    // 1) Tenant & auth: semua role anggota masjid boleh
    mid, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
    if err != nil || mid == uuid.Nil {
        return helper.JsonError(c, fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
    }
    if err := helperAuth.EnsureMemberMasjid(c, mid); err != nil {
        return err
    }

    // 2) Pagination (default: created_at desc)
    p := helper.ParseFiber(c, "created_at", "desc", helper.AdminOpts)

    // 3) Optional filter by path param
    var assessmentID *uuid.UUID
    if s := strings.TrimSpace(c.Params("assessment_id")); s != "" {
        if id, e := uuid.Parse(s); e == nil {
            assessmentID = &id
        } else {
            return helper.JsonError(c, fiber.StatusBadRequest, "assessment_id pada path tidak valid")
        }
    }

    // 4) Query string filters
    if assessmentID == nil {
        if s := strings.TrimSpace(c.Query("assessment_id")); s != "" {
            if id, e := uuid.Parse(s); e == nil {
                assessmentID = &id
            } else {
                return helper.JsonError(c, fiber.StatusBadRequest, "assessment_id tidak valid")
            }
        }
    }
    q := strings.TrimSpace(c.Query("q"))
    isPublishedStr := strings.TrimSpace(c.Query("is_published"))
    isActiveStr := strings.TrimSpace(c.Query("is_active"))

    // 5) Base query + MASJID SCOPE
    db := ctl.DB.WithContext(c.Context()).Model(&model.AssessmentUrlsModel{})

    // === PILIH SALAH SATU: ===
    // OPTION A: kalau tabel assessment_urls punya kolom masjid_id
    // db = db.Where("assessment_urls_masjid_id = ?", mid)

    // OPTION B: kalau TIDAK ada masjid_id di assessment_urls, join ke assessments
    db = db.Joins("JOIN assessments a ON a.assessments_id = assessment_urls_assessment_id").
        Where("a.assessments_masjid_id = ?", mid)
    // === END PILIHAN ===

    if assessmentID != nil {
        db = db.Where("assessment_urls_assessment_id = ?", *assessmentID)
    }
    if q != "" {
        like := "%" + strings.ToLower(q) + "%"
        db = db.Where("(LOWER(assessment_urls_label) LIKE ? OR LOWER(assessment_urls_href) LIKE ?)", like, like)
    }
    if isPublishedStr != "" {
        switch strings.ToLower(isPublishedStr) {
        case "true", "1", "t", "yes", "y":
            db = db.Where("assessment_urls_is_published = ?", true)
        case "false", "0", "f", "no", "n":
            db = db.Where("assessment_urls_is_published = ?", false)
        default:
            return helper.JsonError(c, fiber.StatusBadRequest, "is_published harus boolean")
        }
    }
    if isActiveStr != "" {
        switch strings.ToLower(isActiveStr) {
        case "true", "1", "t", "yes", "y":
            db = db.Where("assessment_urls_is_active = ?", true)
        case "false", "0", "f", "no", "n":
            db = db.Where("assessment_urls_is_active = ?", false)
        default:
            return helper.JsonError(c, fiber.StatusBadRequest, "is_active harus boolean")
        }
    }

    // 6) total
    var total int64
    if err := db.Count(&total).Error; err != nil {
        return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
    }

    // 7) fetch
    if !p.All {
        db = db.Limit(p.Limit()).Offset(p.Offset())
    }
    var rows []model.AssessmentUrlsModel
    if err := db.Order("assessment_urls_created_at DESC").Find(&rows).Error; err != nil {
        return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
    }

    items := make([]dto.AssessmentUrlsResponse, 0, len(rows))
    for i := range rows {
        items = append(items, dto.ToAssessmentUrlsResponse(&rows[i]))
    }

    return helper.JsonList(c, items, helper.BuildMeta(total, p))
}
