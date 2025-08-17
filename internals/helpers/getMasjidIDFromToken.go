package helper

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

//
// ========== Utilities ==========
//

// normalizeLocalsToStrings mencoba mengekstrak daftar string dari berbagai tipe common Locals:
// []string, []interface{}(string), string, uuid.UUID, []uuid.UUID.
// Menghasilkan slice string yang sudah TrimSpace & tidak kosong.
func normalizeLocalsToStrings(v any) []string {
	out := make([]string, 0)
	switch t := v.(type) {
	case []string:
		for _, s := range t {
			s = strings.TrimSpace(s)
			if s != "" {
				out = append(out, s)
			}
		}
	case []interface{}:
		for _, it := range t {
			if s, ok := it.(string); ok {
				s = strings.TrimSpace(s)
				if s != "" {
					out = append(out, s)
				}
			}
		}
	case string:
		if s := strings.TrimSpace(t); s != "" {
			out = append(out, s)
		}
	case uuid.UUID:
		if t != uuid.Nil {
			out = append(out, t.String())
		}
	case []uuid.UUID:
		for _, id := range t {
			if id != uuid.Nil {
				out = append(out, id.String())
			}
		}
	}
	return out
}

// parseFirstUUIDFromLocals mengembalikan UUID pertama dari Locals[key].
func parseFirstUUIDFromLocals(c *fiber.Ctx, key string) (uuid.UUID, error) {
	v := c.Locals(key)
	if v == nil {
		return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, key+" tidak ditemukan di token")
	}
	items := normalizeLocalsToStrings(v)
	if len(items) == 0 {
		return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, key+" kosong di token")
	}
	id, err := uuid.Parse(items[0])
	if err != nil {
		return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "Format "+key+" tidak valid di token")
	}
	return id, nil
}

// parseUUIDSliceFromLocals mengembalikan slice UUID dari Locals[key].
// Error 400 jika ada item yang bukan UUID, 401 jika key ada tapi kosong.
func parseUUIDSliceFromLocals(c *fiber.Ctx, key string) ([]uuid.UUID, error) {
	v := c.Locals(key)
	if v == nil {
		return nil, fiber.NewError(fiber.StatusUnauthorized, key+" tidak ditemukan di token")
	}
	raw := normalizeLocalsToStrings(v)
	if len(raw) == 0 {
		return nil, fiber.NewError(fiber.StatusUnauthorized, key+" kosong di token")
	}
	out := make([]uuid.UUID, 0, len(raw))
	for _, s := range raw {
		id, err := uuid.Parse(s)
		if err != nil {
			return nil, fiber.NewError(fiber.StatusBadRequest, key+" berisi UUID tidak valid")
		}
		out = append(out, id)
	}
	return out, nil
}

//
// ========== Single-tenant getters (dipakai untuk Create/Update/Delete) ==========
//

// Admin-only (existing)
func GetMasjidIDFromToken(c *fiber.Ctx) (uuid.UUID, error) {
	return parseFirstUUIDFromLocals(c, "masjid_admin_ids")
}

// Teacher-only (existing)
func GetTeacherMasjidIDFromToken(c *fiber.Ctx) (uuid.UUID, error) {
	return parseFirstUUIDFromLocals(c, "masjid_teacher_ids")
}

// Prefer TEACHER -> UNION masjid_ids -> ADMIN (existing behavior, dipakai C/U/D)
func GetMasjidIDFromTokenPreferTeacher(c *fiber.Ctx) (uuid.UUID, error) {
	if id, err := parseFirstUUIDFromLocals(c, "masjid_teacher_ids"); err == nil {
		return id, nil
	}
	if id, err := parseFirstUUIDFromLocals(c, "masjid_ids"); err == nil {
		return id, nil
	}
	return parseFirstUUIDFromLocals(c, "masjid_admin_ids")
}

//
// ========== Multi-tenant getter (dipakai untuk List/Read) ==========
//

// Ambil semua masjid yang berhak diakses user dari token.
// Prioritas:
// 1) "masjid_ids" (union semua peran; yang kamu tunjukkan ada di JWT)
// 2) fallback gabungan teacher/admin/student jika "masjid_ids" tidak tersedia.
func GetMasjidIDsFromToken(c *fiber.Ctx) ([]uuid.UUID, error) {
	// 1) union langsung
	if ids, err := parseUUIDSliceFromLocals(c, "masjid_ids"); err == nil {
		return ids, nil
	}

	// 2) fallback gabungan role-role
	groups := []string{"masjid_teacher_ids", "masjid_admin_ids", "masjid_student_ids"}
	seen := map[uuid.UUID]struct{}{}
	out := make([]uuid.UUID, 0, 4)

	var anyFound bool
	for _, key := range groups {
		v := c.Locals(key)
		if v == nil {
			continue
		}
		anyFound = true
		raw := normalizeLocalsToStrings(v)
		for _, s := range raw {
			id, err := uuid.Parse(strings.TrimSpace(s))
			if err != nil {
				return nil, fiber.NewError(fiber.StatusBadRequest, key+" berisi UUID tidak valid")
			}
			if id == uuid.Nil {
				continue
			}
			if _, ok := seen[id]; !ok {
				seen[id] = struct{}{}
				out = append(out, id)
			}
		}
	}

	if !anyFound || len(out) == 0 {
		// terakhir: fallback ke single preferTeacher (biar tetap backward-compatible)
		if id, err := GetMasjidIDFromTokenPreferTeacher(c); err == nil && id != uuid.Nil {
			return []uuid.UUID{id}, nil
		}
		return nil, fiber.NewError(fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}
	return out, nil
}

//
// ========== Role helpers kecil (opsional) ==========
//

func IsAdmin(c *fiber.Ctx) bool   { v := c.Locals("role"); s, _ := v.(string); return strings.EqualFold(s, "admin") }
func IsTeacher(c *fiber.Ctx) bool { v := c.Locals("role"); s, _ := v.(string); return strings.EqualFold(s, "teacher") }
func IsStudent(c *fiber.Ctx) bool { v := c.Locals("role"); s, _ := v.(string); return strings.EqualFold(s, "student") }


