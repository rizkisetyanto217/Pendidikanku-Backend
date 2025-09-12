package controller

import (
	"errors"
	"log"
	"strings"

	"masjidku_backend/internals/features/lembaga/teachers_students/dto"
	"masjidku_backend/internals/features/lembaga/teachers_students/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	statsSvc "masjidku_backend/internals/features/lembaga/stats/lembaga_stats/service"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type MasjidTeacherController struct {
	DB    *gorm.DB
	Stats *statsSvc.LembagaStatsService
}

func NewMasjidTeacherController(db *gorm.DB) *MasjidTeacherController {
	return &MasjidTeacherController{
		DB:    db,
		Stats: statsSvc.NewLembagaStatsService(),
	}
}

// translate error → JsonError
func toJSONErr(c *fiber.Ctx, err error) error {
	if err == nil {
		return nil
	}
	var fe *fiber.Error
	if errors.As(err, &fe) {
		return helper.JsonError(c, fe.Code, fe.Message)
	}
	return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
}


// GET /api/a/masjid-teachers
// Query:
//   page, per_page|limit, sort_by, order
//   id, user_id
//   include=user
func (ctrl *MasjidTeacherController) List(c *fiber.Ctx) error {
	// 🔐 Scope masjid dari token
	masjidUUID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	// Pagination & sorting
	p := helper.ParseFiber(c, "created_at", "desc", helper.DefaultOpts)
	allowedSort := map[string]string{
		"created_at": "masjid_teacher_created_at",
		"updated_at": "masjid_teacher_updated_at",
	}
	orderClause, err := p.SafeOrderClause(allowedSort, "created_at")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid sort_by")
	}
	orderExpr := strings.TrimPrefix(orderClause, "ORDER BY ")

	// Filters
	idStr := strings.TrimSpace(c.Query("id"))
	userIDStr := strings.TrimSpace(c.Query("user_id"))

	var (
		rowID  uuid.UUID
		userID uuid.UUID
	)
	if idStr != "" {
		v, er := uuid.Parse(idStr)
		if er != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "id invalid")
		}
		rowID = v
	}
	if userIDStr != "" {
		v, er := uuid.Parse(userIDStr)
		if er != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "user_id invalid")
		}
		userID = v
	}

	// Base query
	tx := ctrl.DB.WithContext(c.Context()).
		Model(&model.MasjidTeacherModel{}).
		Where("masjid_teacher_masjid_id = ? AND masjid_teacher_deleted_at IS NULL", masjidUUID)

	if rowID != uuid.Nil {
		tx = tx.Where("masjid_teacher_id = ?", rowID)
	}
	if userID != uuid.Nil {
		tx = tx.Where("masjid_teacher_user_id = ?", userID)
	}

	// Count
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Data
	var rows []model.MasjidTeacherModel
	if err := tx.Order(orderExpr).Limit(p.Limit()).Offset(p.Offset()).Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// include=user ?
	wantUser := false
	if inc := strings.ToLower(strings.TrimSpace(c.Query("include"))); inc != "" {
		for _, part := range strings.Split(inc, ",") {
			if strings.TrimSpace(part) == "user" {
				wantUser = true
				break
			}
		}
	}

	// DTO dasar
	base := make([]dto.MasjidTeacherResponse, 0, len(rows))
	userIDsSet := make(map[uuid.UUID]struct{}, len(rows))
	for _, r := range rows {
		base = append(base, dto.MasjidTeacherResponse{
			MasjidTeacherID:        r.MasjidTeacherID.String(),
			MasjidTeacherMasjidID:  r.MasjidTeacherMasjidID.String(),
			MasjidTeacherUserID:    r.MasjidTeacherUserID.String(),
			MasjidTeacherCreatedAt: r.MasjidTeacherCreatedAt,
			MasjidTeacherUpdatedAt: r.MasjidTeacherUpdatedAt,
			// DeletedAt tidak dikirim (sudah difilter IS NULL)
		})
		if wantUser && r.MasjidTeacherUserID != uuid.Nil {
			userIDsSet[r.MasjidTeacherUserID] = struct{}{}
		}
	}

	// Tidak minta user -> return
	if !wantUser {
		meta := helper.BuildMeta(total, p)
		return helper.JsonList(c, base, meta)
	}

	// Bulk fetch users
	type UserLite struct {
		ID       uuid.UUID `json:"id"`
		UserName string    `json:"user_name"`
		FullName *string   `json:"full_name,omitempty"`
		Email    string    `json:"email"`
		IsActive bool      `json:"is_active"`
	}

	userIDs := make([]uuid.UUID, 0, len(userIDsSet))
	for id := range userIDsSet {
		userIDs = append(userIDs, id)
	}

	userMap := make(map[uuid.UUID]UserLite, len(userIDs))
	if len(userIDs) > 0 {
		var urows []UserLite
		if err := ctrl.DB.
			Table("users").
			Select("id, user_name, full_name, email, is_active").
			Where("id IN ?", userIDs).
			Where("deleted_at IS NULL").
			Find(&urows).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
		}
		for _, u := range urows {
			userMap[u.ID] = u
		}
	}

	// Gabungkan
	type Item struct {
		dto.MasjidTeacherResponse `json:",inline"`
		User                      *UserLite `json:"user,omitempty"`
	}
	out := make([]Item, 0, len(base))
	for i, r := range rows {
		var u *UserLite
		if v, ok := userMap[r.MasjidTeacherUserID]; ok {
			tmp := v
			u = &tmp
		}
		out = append(out, Item{
			MasjidTeacherResponse: base[i],
			User:                  u,
		})
	}

	meta := helper.BuildMeta(total, p)
	return helper.JsonList(c, out, meta)
}


/* ============================================
   POST /api/a/masjid-teachers
   Body: { "masjid_teacher_user_id": "<uuid>" }
   (masjid didapat dari token)
   ============================================ */
func (ctrl *MasjidTeacherController) Create(c *fiber.Ctx) error {
	var body dto.CreateMasjidTeacherRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid request")
	}
	if err := validator.New(validator.WithRequiredStructEnabled()).Struct(body); err != nil {
		return helper.JsonError(c, fiber.StatusUnprocessableEntity, err.Error())
	}

	// 🔐 scope & actor
	masjidUUID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}
	actorUUID := uuid.Nil
	if u, err := helperAuth.GetUserIDFromToken(c); err == nil {
		actorUUID = u
	}

	// parse user_id
	userUUID, err := uuid.Parse(body.MasjidTeacherUserID)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "masjid_teacher_user_id tidak valid")
	}

	var created model.MasjidTeacherModel
	if err := ctrl.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		// 0) pastikan user ada (lock)
		var exists bool
		if err := tx.Raw(`
			SELECT EXISTS(
				SELECT 1 FROM users
				WHERE id = ? AND deleted_at IS NULL
				FOR UPDATE
			)
		`, userUUID).Scan(&exists).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal membaca data user")
		}
		if !exists {
			return fiber.NewError(fiber.StatusNotFound, "User tidak ditemukan")
		}

		// 1) idempotent: tolak jika sudah terdaftar aktif di masjid ini
		var dup int64
		if err := tx.Model(&model.MasjidTeacherModel{}).
			Where("masjid_teacher_masjid_id = ? AND masjid_teacher_user_id = ? AND masjid_teacher_deleted_at IS NULL",
				masjidUUID, userUUID).
			Count(&dup).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi pengajar")
		}
		if dup > 0 {
			return fiber.NewError(fiber.StatusConflict, "Pengajar sudah terdaftar")
		}

		// 2) create masjid_teacher
		rec := model.MasjidTeacherModel{
			MasjidTeacherMasjidID: masjidUUID,
			MasjidTeacherUserID:   userUUID,
		}
		if err := tx.Create(&rec).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal menambahkan pengajar")
		}
		created = rec

		// 3) grant role 'teacher' (scoped ke masjid), idempotent & revive-safe
		var assignedBy any // kirim NULL jika actor tidak ada → hindari FK 23503
		if actorUUID != uuid.Nil {
			assignedBy = actorUUID
		}
		if err := tx.Exec(
			`SELECT fn_grant_role(?, 'teacher', ?, ?)`,
			userUUID, masjidUUID, assignedBy,
		).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal menetapkan role 'teacher'")
		}

		// 4) statistik (opsional)
		if err := ctrl.Stats.EnsureForMasjid(tx, masjidUUID); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal memastikan baris statistik")
		}
		if err := ctrl.Stats.IncActiveTeachers(tx, masjidUUID, +1); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui statistik guru")
		}

		return nil
	}); err != nil {
		return toJSONErr(c, err)
	}

	resp := dto.MasjidTeacherResponse{
		MasjidTeacherID:        created.MasjidTeacherID.String(),
		MasjidTeacherMasjidID:  created.MasjidTeacherMasjidID.String(),
		MasjidTeacherUserID:    created.MasjidTeacherUserID.String(),
		MasjidTeacherCreatedAt: created.MasjidTeacherCreatedAt,
		MasjidTeacherUpdatedAt: created.MasjidTeacherUpdatedAt,
	}
	return helper.JsonCreated(c, "Pengajar berhasil ditambahkan & role 'teacher' diberikan", resp)
}

/* ============================================
   DELETE /api/a/masjid-teachers/:id
   Soft delete + update statistik
   ============================================ */
func (ctrl *MasjidTeacherController) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak boleh kosong")
	}

	// 🔐 Admin-only
	masjidUUID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	var rows int64
	if err := ctrl.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		var teacher model.MasjidTeacherModel
		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&teacher,
				"masjid_teacher_id = ? AND masjid_teacher_masjid_id = ? AND masjid_teacher_deleted_at IS NULL",
				id, masjidUUID,
			).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusNotFound, "Pengajar tidak ditemukan atau sudah dihapus")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data pengajar")
		}

		res := tx.Where("masjid_teacher_id = ?", teacher.MasjidTeacherID).
			Delete(&model.MasjidTeacherModel{}) // soft delete
		if res.Error != nil {
			log.Println("[ERROR] Failed to delete masjid teacher:", res.Error)
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus pengajar")
		}
		rows = res.RowsAffected

		if rows > 0 {
			if err := ctrl.Stats.EnsureForMasjid(tx, masjidUUID); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal memastikan baris statistik")
			}
			if err := ctrl.Stats.IncActiveTeachers(tx, masjidUUID, -1); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui statistik guru")
			}
		}
		return nil
	}); err != nil {
		return toJSONErr(c, err)
	}

	return helper.JsonDeleted(c, "Pengajar berhasil dihapus", fiber.Map{
		"masjid_teacher_id": id,
		"affected":          rows,
	})
}
