// internals/features/lembaga/classes/sections/main/controller/class_section_list_controller.go
package controller

import (
	"errors"
	"strings"

	ucsDTO "masjidku_backend/internals/features/school/classes/class_sections/dto"
	secModel "masjidku_backend/internals/features/school/classes/class_sections/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	// <- helper pagination (alias biar tak bentrok)

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* ================= List ================= */

// GET /admin/class-sections
// GET /admin/class-sections
func (ctrl *ClassSectionController) ListClassSections(c *fiber.Ctx) error {
	// 🔁 multi-tenant read (semua klaim masjid)
	masjidIDs, err := helperAuth.GetMasjidIDsFromToken(c)
	if err != nil {
		return err
	}

	// ------------ Search term (gabungan q & search) ------------
	rawQ := strings.TrimSpace(c.Query("q"))
	rawSearch := strings.TrimSpace(c.Query("search"))
	searchTerm := rawSearch
	if rawQ != "" {
		searchTerm = rawQ
		// mengikuti perilaku Search lama → q minimal 2 karakter
		if len([]rune(searchTerm)) < 2 {
			return fiber.NewError(fiber.StatusBadRequest, "Parameter q minimal 2 karakter")
		}
	}

	// ------------ Pagination & sorting (dynamic default) ------------
	defaultSortBy := "created_at"
	defaultSortOrder := "desc"
	if searchTerm != "" {
		defaultSortBy = "name"
		defaultSortOrder = "asc"
	}
	p := helper.ParseFiber(c, defaultSortBy, defaultSortOrder, helper.AdminOpts)

	// kolom yang diizinkan untuk sort
	allowed := map[string]string{
		"name":       "class_sections_name",
		"created_at": "class_sections_created_at",
	}
	orderClause, _ := helper.Params{
		SortBy:    p.SortBy,
		SortOrder: p.SortOrder,
	}.SafeOrderClause(allowed, defaultSortBy)
	orderClause = strings.TrimPrefix(orderClause, "ORDER BY ")

	// ------------ Parse includes ------------
	includeStr := strings.ToLower(strings.TrimSpace(c.Query("include")))
	includeAll := includeStr == "all"
	includes := map[string]bool{}
	for _, part := range strings.Split(includeStr, ",") {
		if s := strings.TrimSpace(part); s != "" {
			includes[s] = true
		}
	}
	wantClass := includeAll || includes["class"] || includes["classes"]
	wantRoom := includeAll || includes["room"] || includes["rooms"]
	wantTeacher := includeAll || includes["teacher"] || includes["teachers"]

	// ------------ Filters ------------
	var (
		classID, teacherID, roomID *uuid.UUID
		activeOnly                 *bool
	)
	if s := strings.TrimSpace(c.Query("class_id")); s != "" {
		if id, e := uuid.Parse(s); e == nil {
			classID = &id
		} else {
			return fiber.NewError(fiber.StatusBadRequest, "class_id tidak valid")
		}
	}
	if s := strings.TrimSpace(c.Query("teacher_id")); s != "" {
		if id, e := uuid.Parse(s); e == nil {
			teacherID = &id
		} else {
			return fiber.NewError(fiber.StatusBadRequest, "teacher_id tidak valid")
		}
	}
	if s := strings.TrimSpace(c.Query("room_id")); s != "" {
		if id, e := uuid.Parse(s); e == nil {
			roomID = &id
		} else {
			return fiber.NewError(fiber.StatusBadRequest, "room_id tidak valid")
		}
	}
	if s := strings.TrimSpace(c.Query("active_only")); s != "" {
		b := c.QueryBool("active_only")
		activeOnly = &b
	}

	// ------------ Base query ------------
	tx := ctrl.DB.Model(&secModel.ClassSectionModel{}).
		Where("class_sections_deleted_at IS NULL").
		Where("class_sections_masjid_id IN ?", masjidIDs)

	if activeOnly != nil {
		tx = tx.Where("class_sections_is_active = ?", *activeOnly)
	}
	if classID != nil {
		tx = tx.Where("class_sections_class_id = ?", *classID)
	}
	if teacherID != nil {
		tx = tx.Where("class_sections_teacher_id = ?", *teacherID)
	}
	if roomID != nil {
		tx = tx.Where("class_sections_class_room_id = ?", *roomID)
	}
	// 🔎 unified search
	if searchTerm != "" {
		s := "%" + strings.ToLower(searchTerm) + "%"
		tx = tx.Where(`LOWER(class_sections_name) LIKE ?
		               OR LOWER(class_sections_code) LIKE ?
		               OR LOWER(class_sections_slug) LIKE ?`, s, s, s)
	}

	// ------------ Total count (sebelum limit/offset) ------------
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghitung total")
	}

	// ------------ Fetch rows ------------
	if orderClause != "" {
		tx = tx.Order(orderClause)
	}
	if !p.All {
		tx = tx.Limit(p.Limit()).Offset(p.Offset())
	}

	var rows []secModel.ClassSectionModel
	if err := tx.Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	/* ========= Prefetch TEACHER → users ========= */
	teacherToUser := make(map[uuid.UUID]uuid.UUID) // masjid_teacher_id -> users.id
	userMap := map[uuid.UUID]ucsDTO.UserLite{}     // users.id -> user lite
	if wantTeacher && len(rows) > 0 {
		tSet := make(map[uuid.UUID]struct{})
		for i := range rows {
			if rows[i].ClassSectionsTeacherID != nil {
				tSet[*rows[i].ClassSectionsTeacherID] = struct{}{}
			}
		}
		if len(tSet) > 0 {
			teacherIDs := make([]uuid.UUID, 0, len(tSet))
			for id := range tSet {
				teacherIDs = append(teacherIDs, id)
			}
			type mtRow struct {
				ID     uuid.UUID `gorm:"column:masjid_teacher_id"`
				UserID uuid.UUID `gorm:"column:masjid_teacher_user_id"`
			}
			var mts []mtRow
			if err := ctrl.DB.
				Table("masjid_teachers").
				Select("masjid_teacher_id, masjid_teacher_user_id").
				Where("masjid_teacher_id IN ?", teacherIDs).
				Find(&mts).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data relasi pengajar")
			}
			found := make(map[uuid.UUID]struct{}, len(mts))
			for _, r := range mts {
				teacherToUser[r.ID] = r.UserID
				found[r.ID] = struct{}{}
			}
			// fallback skema lama (jaga kompat)
			for _, tid := range teacherIDs {
				if _, ok := found[tid]; !ok {
					teacherToUser[tid] = tid
				}
			}
			// ambil users
			uSet := make(map[uuid.UUID]struct{}, len(teacherToUser))
			for _, uid := range teacherToUser {
				uSet[uid] = struct{}{}
			}
			if len(uSet) > 0 {
				userIDs := make([]uuid.UUID, 0, len(uSet))
				for uid := range uSet {
					userIDs = append(userIDs, uid)
				}
				type userRow struct {
					ID       uuid.UUID `gorm:"column:id"`
					UserName string    `gorm:"column:user_name"`
					FullName *string   `gorm:"column:full_name"`
					Email    string    `gorm:"column:email"`
					IsActive bool      `gorm:"column:is_active"`
				}
				var urs []userRow
				if err := ctrl.DB.
					Table("users").
					Select("id, user_name, full_name, email, is_active").
					Where("id IN ?", userIDs).
					Find(&urs).Error; err != nil {
					return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data guru")
				}
				for _, u := range urs {
					full := ""
					if u.FullName != nil {
						full = *u.FullName
					}
					userMap[u.ID] = ucsDTO.UserLite{
						ID:       u.ID,
						UserName: u.UserName,
						FullName: full,
						Email:    u.Email,
						IsActive: u.IsActive,
					}
				}
			}
		}
	}

	/* ========= Prefetch CLASSES (pakai class_parent untuk nama) ========= */
	type classLite struct {
		ID   uuid.UUID `json:"id"   gorm:"column:id"`
		Name string    `json:"name" gorm:"column:name"`
		Slug string    `json:"slug,omitempty" gorm:"column:slug"`
	}
	classMap := map[uuid.UUID]classLite{}
	if wantClass && len(rows) > 0 {
		cSet := map[uuid.UUID]struct{}{}
		for i := range rows {
			if rows[i].ClassSectionsClassID != uuid.Nil {
				cSet[rows[i].ClassSectionsClassID] = struct{}{}
			}
		}
		if len(cSet) > 0 {
			classIDs := make([]uuid.UUID, 0, len(cSet))
			for id := range cSet {
				classIDs = append(classIDs, id)
			}
			var cr []classLite
			if err := ctrl.DB.
				Table("classes AS c").
				Select(`
					c.class_id AS id,
					cp.class_parent_name AS name,
					c.class_slug AS slug
				`).
				Joins(`JOIN public.class_parents AS cp
						ON cp.class_parent_id = c.class_parent_id
						AND cp.class_parent_masjid_id = c.class_masjid_id
						AND cp.class_parent_deleted_at IS NULL`).
				Where("c.class_id IN ? AND c.class_deleted_at IS NULL", classIDs).
				Find(&cr).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data classes")
			}
			for _, r := range cr {
				classMap[r.ID] = r
			}
		}
	}

	/* ========= Prefetch ROOMS ========= */
	type roomLite struct {
		ID   uuid.UUID `json:"id"   gorm:"column:id"`
		Name string    `json:"name" gorm:"column:name"`
	}
	roomMap := map[uuid.UUID]roomLite{}
	if wantRoom && len(rows) > 0 {
		rSet := map[uuid.UUID]struct{}{}
		for i := range rows {
			if rows[i].ClassSectionsClassRoomID != nil && *rows[i].ClassSectionsClassRoomID != uuid.Nil {
				rSet[*rows[i].ClassSectionsClassRoomID] = struct{}{}
			}
		}
		if len(rSet) > 0 {
			roomIDs := make([]uuid.UUID, 0, len(rSet))
			for id := range rSet {
				roomIDs = append(roomIDs, id)
			}
			var rr []roomLite
			if err := ctrl.DB.
				Table("class_rooms").
				Select("class_room_id AS id, class_rooms_name AS name").
				Where("class_room_id IN ? AND class_rooms_deleted_at IS NULL", roomIDs).
				Find(&rr).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data rooms")
			}
			for _, r := range rr {
				roomMap[r.ID] = r
			}
		}
	}

	/* ========= Build response ========= */
	type sectionWithExpand struct {
		*ucsDTO.ClassSectionResponse `json:",inline"`
		Class   *classLite       `json:"class,omitempty"`
		Room    *roomLite        `json:"room,omitempty"`
		Teacher *ucsDTO.UserLite `json:"teacher,omitempty"`
	}

	out := make([]*sectionWithExpand, 0, len(rows))
	for i := range rows {
		// teacher name utk field bawaan DTO
		teacherName := ""
		var teacherPtr *ucsDTO.UserLite
		if wantTeacher && rows[i].ClassSectionsTeacherID != nil {
			if uid, ok := teacherToUser[*rows[i].ClassSectionsTeacherID]; ok {
				if ul, ok := userMap[uid]; ok {
					if ul.FullName != "" {
						teacherName = ul.FullName
					} else {
						teacherName = ul.UserName
					}
					uCopy := ul
					teacherPtr = &uCopy
				}
			}
		}
		base := ucsDTO.NewClassSectionResponse(&rows[i], teacherName)

		w := &sectionWithExpand{ClassSectionResponse: base}
		if wantTeacher {
			w.Teacher = teacherPtr
		}
		if wantClass {
			if cl, ok := classMap[rows[i].ClassSectionsClassID]; ok {
				cCopy := cl
				w.Class = &cCopy
			}
		}
		if wantRoom && rows[i].ClassSectionsClassRoomID != nil {
			if rl, ok := roomMap[*rows[i].ClassSectionsClassRoomID]; ok {
				rCopy := rl
				w.Room = &rCopy
			}
		}
		out = append(out, w)
	}

	meta := helper.BuildMeta(total, p)
	return helper.JsonOK(c, "OK", fiber.Map{
		"data": out,
		"meta": meta,
	})
}



/* ================= Get by Slug ================= */

// GET /admin/class-sections/slug/:slug
func (ctrl *ClassSectionController) GetClassSectionBySlug(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}
	slug := helper.GenerateSlug(c.Params("slug"))

	var m secModel.ClassSectionModel
	if err := ctrl.DB.
		Where("class_sections_masjid_id = ? AND lower(class_sections_slug) = lower(?) AND class_sections_deleted_at IS NULL",
			masjidID, slug).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Section tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	return helper.JsonOK(c, "OK", ucsDTO.NewClassSectionResponse(&m, ""))
}
