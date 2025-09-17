// internals/features/lembaga/teachers_students/controller/masjid_teacher_controller.go
package controller

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	yDTO "masjidku_backend/internals/features/lembaga/teachers_students/dto"
	yModel "masjidku_backend/internals/features/lembaga/teachers_students/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	statsSvc "masjidku_backend/internals/features/lembaga/stats/lembaga_stats/service"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

/*
Routes (scope admin/teacher/dkm/bendahara di masjid terkait):

GET    /api/a/masjid-teachers            -> List
GET    /api/a/masjid-teachers/:id        -> Detail
POST   /api/a/masjid-teachers            -> Create
PATCH  /api/a/masjid-teachers/:id        -> Update (tri-state)
DELETE /api/a/masjid-teachers/:id        -> Soft delete
*/

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

// --- tiny helpers

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

func isUniqueViolation(err error) bool {
	// Postgres code 23505; gorm error message mengandung nama constraint
	return err != nil && strings.Contains(err.Error(), "duplicate key value")
}

func uniqueMessage(err error) string {
	if err == nil {
		return ""
	}
	s := err.Error()
	switch {
	case strings.Contains(s, "ux_mtj_masjid_user_alive"):
		return "User sudah terdaftar sebagai pengajar di masjid ini"
	case strings.Contains(s, "ux_mtj_code_alive_ci"):
		return "Kode pengajar sudah digunakan di masjid ini"
	case strings.Contains(s, "ux_mtj_nip_alive_ci"):
		return "NIP sudah digunakan di masjid ini"
	// kalau kamu punya constraint unik lain (slug), tambahkan di sini
	case strings.Contains(strings.ToLower(s), "slug"):
		return "Slug pengajar sudah digunakan"
	default:
		return "Data unik sudah digunakan"
	}
}

func parseDateYYYYMMDD(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil, fmt.Errorf("format tanggal harus YYYY-MM-DD")
	}
	return &t, nil
}

/* ===================== LIST ===================== */
// GET /api/a/masjid-teachers
// Query: page, per_page|limit, sort_by(created_at|updated_at), order
//        id, user_id, employment, active, verified, public,
//        joined_from, joined_to (YYYY-MM-DD), q (notes/code/slug ILIKE)
//        include=user
func (ctrl *MasjidTeacherController) List(c *fiber.Ctx) error {
	masjidUUID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

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

	// filters
	idStr := strings.TrimSpace(c.Query("id"))
	userIDStr := strings.TrimSpace(c.Query("user_id"))
	employment := strings.ToLower(strings.TrimSpace(c.Query("employment")))
	activeStr := strings.TrimSpace(c.Query("active"))
	verifiedStr := strings.TrimSpace(c.Query("verified"))
	publicStr := strings.TrimSpace(c.Query("public"))
	joinedFromStr := strings.TrimSpace(c.Query("joined_from"))
	joinedToStr := strings.TrimSpace(c.Query("joined_to"))
	q := strings.TrimSpace(c.Query("q"))

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

	tx := ctrl.DB.WithContext(c.Context()).
		Model(&yModel.MasjidTeacherModel{}).
		Where("masjid_teacher_masjid_id = ? AND masjid_teacher_deleted_at IS NULL", masjidUUID)

	if rowID != uuid.Nil {
		tx = tx.Where("masjid_teacher_id = ?", rowID)
	}
	if userID != uuid.Nil {
		tx = tx.Where("masjid_teacher_user_id = ?", userID)
	}
	if employment != "" {
		tx = tx.Where("masjid_teacher_employment = ?", employment)
	}
	if activeStr != "" {
		if b, er := strconv.ParseBool(activeStr); er == nil {
			tx = tx.Where("masjid_teacher_is_active = ?", b)
		}
	}
	if verifiedStr != "" {
		if b, er := strconv.ParseBool(verifiedStr); er == nil {
			tx = tx.Where("masjid_teacher_is_verified = ?", b)
		}
	}
	if publicStr != "" {
		if b, er := strconv.ParseBool(publicStr); er == nil {
			tx = tx.Where("masjid_teacher_is_public = ?", b)
		}
	}
	if joinedFromStr != "" {
		if t, er := parseDateYYYYMMDD(joinedFromStr); er == nil {
			tx = tx.Where("masjid_teacher_joined_at IS NOT NULL AND masjid_teacher_joined_at >= ?", t)
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "joined_from invalid (YYYY-MM-DD)")
		}
	}
	if joinedToStr != "" {
		if t, er := parseDateYYYYMMDD(joinedToStr); er == nil {
			tx = tx.Where("masjid_teacher_joined_at IS NOT NULL AND masjid_teacher_joined_at <= ?", t)
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "joined_to invalid (YYYY-MM-DD)")
		}
	}
	if q != "" {
		pat := "%" + q + "%"
		tx = tx.Where(`(masjid_teacher_notes ILIKE ? OR masjid_teacher_code ILIKE ? OR masjid_teacher_slug ILIKE ?)`, pat, pat, pat)
	}

	// count
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// data
	var rows []yModel.MasjidTeacherModel
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
	base := make([]*yDTO.MasjidTeacherResponse, 0, len(rows))
	userIDsSet := make(map[uuid.UUID]struct{}, len(rows))
	for i := range rows {
		base = append(base, yDTO.NewMasjidTeacherResponse(&rows[i]))
		if wantUser && rows[i].MasjidTeacherUserID != uuid.Nil {
			userIDsSet[rows[i].MasjidTeacherUserID] = struct{}{}
		}
	}

	if !wantUser {
		meta := helper.BuildMeta(total, p)
		return helper.JsonList(c, base, meta)
	}

	// bulk fetch users
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

	type Item struct {
		*yDTO.MasjidTeacherResponse `json:",inline"`
		User                        *UserLite `json:"user,omitempty"`
	}
	out := make([]Item, 0, len(base))
	for i := range rows {
		var u *UserLite
		if v, ok := userMap[rows[i].MasjidTeacherUserID]; ok {
			tmp := v
			u = &tmp
		}
		out = append(out, Item{MasjidTeacherResponse: base[i], User: u})
	}

	meta := helper.BuildMeta(total, p)
	return helper.JsonList(c, out, meta)
}

/* ===================== DETAIL ===================== */
// GET /api/a/masjid-teachers/:id
func (ctrl *MasjidTeacherController) Detail(c *fiber.Ctx) error {
	masjidUUID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}
	id := strings.TrimSpace(c.Params("id"))
	if id == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak boleh kosong")
	}
	rowID, err := uuid.Parse(id)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var row yModel.MasjidTeacherModel
	if err := ctrl.DB.WithContext(c.Context()).
		First(&row,
			"masjid_teacher_id = ? AND masjid_teacher_masjid_id = ? AND masjid_teacher_deleted_at IS NULL",
			rowID, masjidUUID,
		).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Pengajar tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	return helper.JsonOK(c, "Detail pengajar", yDTO.NewMasjidTeacherResponse(&row))
}

/* ===================== CREATE ===================== */
// POST /api/a/masjid-teachers
func (ctrl *MasjidTeacherController) Create(c *fiber.Ctx) error {
	var body yDTO.CreateMasjidTeacherRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid request")
	}
	if err := validator.New(validator.WithRequiredStructEnabled()).Struct(body); err != nil {
		return helper.JsonError(c, fiber.StatusUnprocessableEntity, err.Error())
	}

	masjidUUID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	// Build model dari DTO (pakai masjid dari token; user dari body)
	rec, err := body.ToModel(masjidUUID.String(), body.MasjidTeacherUserID)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var created yModel.MasjidTeacherModel
	if err := ctrl.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		// 0) pastikan user ada & belum soft-deleted
		var exists bool
		if err := tx.Raw(`
			SELECT EXISTS(SELECT 1 FROM users WHERE id = ? AND deleted_at IS NULL)
		`, rec.MasjidTeacherUserID).Scan(&exists).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal membaca data user")
		}
		if !exists {
			return fiber.NewError(fiber.StatusNotFound, "User tidak ditemukan")
		}

		// 1) tolak jika sudah ada aktif
		var dup int64
		if err := tx.Model(&yModel.MasjidTeacherModel{}).
			Where("masjid_teacher_masjid_id = ? AND masjid_teacher_user_id = ? AND masjid_teacher_deleted_at IS NULL",
				rec.MasjidTeacherMasjidID, rec.MasjidTeacherUserID).
			Count(&dup).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi pengajar")
		}
		if dup > 0 {
			return fiber.NewError(fiber.StatusConflict, "Pengajar sudah terdaftar")
		}

		// 2) insert
		if err := tx.Create(&rec).Error; err != nil {
			if isUniqueViolation(err) {
				return fiber.NewError(fiber.StatusConflict, uniqueMessage(err))
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal menambahkan pengajar")
		}
		created = *rec

		// 3) grant role teacher (idempotent, biarkan function handle revive)
		actorUUID, _ := helperAuth.GetUserIDFromToken(c)
		var assignedBy any
		if actorUUID != uuid.Nil {
			assignedBy = actorUUID
		}
		if err := tx.Exec(
			`SELECT fn_grant_role(?, 'teacher', ?, ?)`,
			rec.MasjidTeacherUserID, rec.MasjidTeacherMasjidID, assignedBy,
		).Error; err != nil {
			log.Printf("[WARN] fn_grant_role error: %v", err)
			// tidak fatal untuk create; lanjutkan
		}

		// 4) statistik
		if err := ctrl.Stats.EnsureForMasjid(tx, rec.MasjidTeacherMasjidID); err == nil {
			if rec.MasjidTeacherIsActive {
				_ = ctrl.Stats.IncActiveTeachers(tx, rec.MasjidTeacherMasjidID, +1)
			}
		}

		return nil
	}); err != nil {
		return toJSONErr(c, err)
	}

	return helper.JsonCreated(c, "Pengajar berhasil ditambahkan", yDTO.NewMasjidTeacherResponse(&created))
}

/* ===================== UPDATE ===================== */
// PATCH /api/a/masjid-teachers/:id
func (ctrl *MasjidTeacherController) Update(c *fiber.Ctx) error {
	masjidUUID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}
	id := strings.TrimSpace(c.Params("id"))
	if id == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak boleh kosong")
	}
	rowID, err := uuid.Parse(id)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var req yDTO.UpdateMasjidTeacherRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}

	var before yModel.MasjidTeacherModel
	if err := ctrl.DB.WithContext(c.Context()).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&before, "masjid_teacher_id = ? AND masjid_teacher_masjid_id = ? AND masjid_teacher_deleted_at IS NULL",
			rowID, masjidUUID).
		Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Pengajar tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// Larang pindah masjid (cross-tenant) lewat PATCH
	if req.MasjidTeacherMasjidID != nil {
		if reqID, er := uuid.Parse(strings.TrimSpace(*req.MasjidTeacherMasjidID)); er != nil || reqID != masjidUUID {
			return helper.JsonError(c, fiber.StatusForbidden, "Tidak boleh memindahkan pengajar ke masjid lain lewat endpoint ini")
		}
	}

	// apply tri-state
	wasActive := before.MasjidTeacherIsActive
	if err := req.ApplyToModel(&before); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	before.MasjidTeacherUpdatedAt = time.Now()

	if err := ctrl.DB.WithContext(c.Context()).Save(&before).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, uniqueMessage(err))
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui pengajar")
	}

	// Statistik: jika status aktif berubah, adjust counter
	if wasActive != before.MasjidTeacherIsActive {
		if err := ctrl.Stats.EnsureForMasjid(ctrl.DB, masjidUUID); err == nil {
			delta := -1
			if before.MasjidTeacherIsActive {
				delta = +1
			}
			_ = ctrl.Stats.IncActiveTeachers(ctrl.DB, masjidUUID, delta)
		}
	}

	return helper.JsonUpdated(c, "Pengajar diperbarui", yDTO.NewMasjidTeacherResponse(&before))
}

/* ===================== DELETE ===================== */
// DELETE /api/a/masjid-teachers/:id (soft)
func (ctrl *MasjidTeacherController) Delete(c *fiber.Ctx) error {
	id := strings.TrimSpace(c.Params("id"))
	if id == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak boleh kosong")
	}

	masjidUUID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	var rows int64
	if err := ctrl.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		var teacher yModel.MasjidTeacherModel
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
			Delete(&yModel.MasjidTeacherModel{}) // soft delete
		if res.Error != nil {
			log.Println("[ERROR] Failed to delete masjid teacher:", res.Error)
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus pengajar")
		}
		rows = res.RowsAffected

		if rows > 0 && teacher.MasjidTeacherIsActive {
			// turunkan statistik aktif
			if err := ctrl.Stats.EnsureForMasjid(tx, masjidUUID); err == nil {
				_ = ctrl.Stats.IncActiveTeachers(tx, masjidUUID, -1)
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
