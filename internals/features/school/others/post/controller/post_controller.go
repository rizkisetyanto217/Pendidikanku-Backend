// file: internals/features/social/posts/controller/post_controller.go
package controller

import (
	"errors"
	"strconv"
	"strings"
	"time"

	validator "github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	dto "madinahsalam_backend/internals/features/school/others/post/dto"
	model "madinahsalam_backend/internals/features/school/others/post/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"
)

/* ==============================
   Controller
============================== */

type PostController struct {
	DB        *gorm.DB
	Validator *validator.Validate
}

func NewPostController(db *gorm.DB) *PostController {
	return &PostController{
		DB:        db,
		Validator: validator.New(),
	}
}

/* ==============================
   Small helpers
============================== */

func atoiOr(def int, s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return def
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return i
}

func parseBoolPtr(s string) *bool {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return nil
	}
	switch s {
	case "1", "true", "t", "yes", "y":
		v := true
		return &v
	case "0", "false", "f", "no", "n":
		v := false
		return &v
	default:
		return nil
	}
}

func parseYMDLocal(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	t, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		return nil, err
	}
	tt := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
	return &tt, nil
}

func isUUID(s string) bool {
	_, err := uuid.Parse(strings.TrimSpace(s))
	return err == nil
}

func applySort(db *gorm.DB, sortBy, sortDir string) *gorm.DB {
	col := "post_created_at"
	switch strings.ToLower(strings.TrimSpace(sortBy)) {
	case "date":
		col = "post_date"
	case "title":
		col = "post_title"
	case "published_at":
		col = "post_published_at"
	case "created_at", "":
		col = "post_created_at"
	}
	dir := "DESC"
	if strings.EqualFold(strings.TrimSpace(sortDir), "asc") {
		dir = "ASC"
	}
	return db.Order(col + " " + dir).Order("post_id DESC")
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "sqlstate 23505") ||
		strings.Contains(s, "duplicate key") ||
		strings.Contains(s, "unique constraint")
}

/* ==============================
   Auth & school context
============================== */

// Resolve school via context (id/slug) dan pastikan user adalah member
func resolveSchoolForRead(c *fiber.Ctx, db *gorm.DB) (uuid.UUID, error) {
	c.Locals("DB", db) // untuk helper slug→id
	mc, err := helperAuth.ResolveSchoolContext(c)
	if err != nil {
		return uuid.Nil, err
	}

	var mid uuid.UUID
	if mc.ID != uuid.Nil {
		mid = mc.ID
	} else if s := strings.TrimSpace(mc.Slug); s != "" {
		id, er := helperAuth.GetSchoolIDBySlug(c, s)
		if er != nil || id == uuid.Nil {
			return uuid.Nil, fiber.NewError(fiber.StatusNotFound, "School (slug) tidak ditemukan")
		}
		mid = id
	} else {
		return uuid.Nil, helperAuth.ErrSchoolContextMissing
	}

	// minimal member
	if !helperAuth.UserHasSchool(c, mid) {
		return uuid.Nil, fiber.NewError(fiber.StatusForbidden, "Anda bukan member school ini")
	}
	return mid, nil
}

// Resolve school untuk write (DKM/Teacher/Owner)
func resolveSchoolForWrite(c *fiber.Ctx, db *gorm.DB) (uuid.UUID, error) {
	c.Locals("DB", db)
	mc, err := helperAuth.ResolveSchoolContext(c)
	if err != nil {
		return uuid.Nil, err
	}

	var mid uuid.UUID
	if mc.ID != uuid.Nil {
		mid = mc.ID
	} else if s := strings.TrimSpace(mc.Slug); s != "" {
		id, er := helperAuth.GetSchoolIDBySlug(c, s)
		if er != nil || id == uuid.Nil {
			return uuid.Nil, fiber.NewError(fiber.StatusNotFound, "School (slug) tidak ditemukan")
		}
		mid = id
	} else {
		return uuid.Nil, helperAuth.ErrSchoolContextMissing
	}

	// DKM/Teacher (Owner juga boleh)
	if err := helperAuth.EnsureDKMOrTeacherSchool(c, mid); err != nil && !helperAuth.IsOwner(c) {
		return uuid.Nil, err
	}
	return mid, nil
}

/* ==============================
   Handlers
============================== */

// POST /posts — Create
func (ctl *PostController) Create(c *fiber.Ctx) error {
	// auth
	mid, err := resolveSchoolForWrite(c, ctl.DB)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var req dto.CreatePostRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.Validator.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// force tenant
	req.PostSchoolID = mid

	m := req.ToModel()

	// slugify + ensure unique per tenant (alive only)
	base := ""
	if m.PostSlug != nil && strings.TrimSpace(*m.PostSlug) != "" {
		base = helper.Slugify(*m.PostSlug, 160)
	} else {
		base = helper.Slugify(m.PostTitle, 160)
	}
	uniq, err := helper.EnsureUniqueSlugCI(
		c.Context(),
		ctl.DB,
		"posts",
		"post_slug",
		base,
		func(q *gorm.DB) *gorm.DB {
			return q.Where("post_school_id = ? AND post_deleted_at IS NULL", mid)
		},
		160,
	)
	if err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyiapkan slug")
	}
	m.PostSlug = &uniq

	// simpan
	if err := ctl.DB.WithContext(c.Context()).Create(m).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Slug/Key sudah dipakai")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonCreated(c, "Post dibuat", dto.FromModelPost(m))
}

// PATCH /posts/:id — Update partial
func (ctl *PostController) Patch(c *fiber.Ctx) error {
	idOrSlug := strings.TrimSpace(c.Params("id"))
	if idOrSlug == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Param id kosong")
	}

	var row model.Post
	var mid uuid.UUID
	var err error

	// Jika param UUID → ambil langsung by ID; lalu authorize di school row tsb.
	if isUUID(idOrSlug) {
		id, _ := uuid.Parse(idOrSlug)
		if err := ctl.DB.WithContext(c.Context()).
			First(&row, "post_id = ? AND post_deleted_at IS NULL", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, fiber.StatusNotFound, "Post tidak ditemukan")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
		}
		mid = row.PostSchoolID
		// authorize
		if err := helperAuth.EnsureDKMOrTeacherSchool(c, mid); err != nil && !helperAuth.IsOwner(c) {
			if fe, ok := err.(*fiber.Error); ok {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return err
		}
	} else {
		// Param dianggap slug → wajib school context
		mid, err = resolveSchoolForWrite(c, ctl.DB)
		if err != nil {
			if fe, ok := err.(*fiber.Error); ok {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
		}
		if err := ctl.DB.WithContext(c.Context()).
			First(&row, "post_school_id = ? AND post_slug = ? AND post_deleted_at IS NULL", mid, idOrSlug).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, fiber.StatusNotFound, "Post tidak ditemukan")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
		}
	}

	var body dto.PatchPostRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.Validator.Struct(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	updates := body.ToUpdates()

	// Jika ada post_slug (string) → slugify + ensure unique (exclude diri sendiri)
	if v, ok := updates["post_slug"]; ok {
		if v == nil {
			// explicit null → ok
		} else if s, ok2 := v.(string); ok2 {
			base := helper.Slugify(s, 160)
			uniq, err := helper.EnsureUniqueSlugCI(
				c.Context(),
				ctl.DB,
				"posts",
				"post_slug",
				base,
				func(q *gorm.DB) *gorm.DB {
					return q.Where(
						"post_school_id = ? AND post_deleted_at IS NULL AND post_id <> ?",
						row.PostSchoolID, row.PostID,
					)
				},
				160,
			)
			if err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyiapkan slug")
			}
			updates["post_slug"] = uniq
		}
	}

	if len(updates) == 0 {
		return helper.JsonOK(c, "Tidak ada perubahan", dto.FromModelPost(&row))
	}

	if err := ctl.DB.WithContext(c.Context()).
		Model(&model.Post{}).
		Where("post_id = ? AND post_deleted_at IS NULL", row.PostID).
		Updates(updates).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Slug/Key sudah dipakai")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// reload
	if err := ctl.DB.WithContext(c.Context()).
		First(&row, "post_id = ?", row.PostID).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonUpdated(c, "Post diperbarui", dto.FromModelPost(&row))
}

// DELETE /posts/:id — soft delete
func (ctl *PostController) Delete(c *fiber.Ctx) error {
	idStr := strings.TrimSpace(c.Params("id"))
	id, err := uuid.Parse(idStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var row model.Post
	if err := ctl.DB.WithContext(c.Context()).
		First(&row, "post_id = ? AND post_deleted_at IS NULL", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Post tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// authorize
	if err := helperAuth.EnsureDKMOrTeacherSchool(c, row.PostSchoolID); err != nil && !helperAuth.IsOwner(c) {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return err
	}

	if err := ctl.DB.WithContext(c.Context()).Delete(&row).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonDeleted(c, "Post dihapus", fiber.Map{"post_id": id})
}
