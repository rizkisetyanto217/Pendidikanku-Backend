package snapshot

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type CSSTSnapshot struct {
	Name      *string    `json:"name,omitempty"`
	TeacherID *uuid.UUID `json:"teacher_id,omitempty"`
	SectionID *uuid.UUID `json:"section_id,omitempty"`
}

func tableColumns(tx *gorm.DB, table string) (map[string]struct{}, error) {
	type colRow struct {
		ColumnName string `gorm:"column:column_name"`
	}
	var rows []colRow
	q := `
SELECT column_name
FROM information_schema.columns
WHERE table_name = ?
  AND table_schema = ANY (current_schemas(true))`
	if err := tx.Raw(q, table).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]struct{}, len(rows))
	for _, r := range rows {
		out[strings.ToLower(strings.TrimSpace(r.ColumnName))] = struct{}{}
	}
	return out, nil
}
func firstExisting(cols map[string]struct{}, cands ...string) string {
	for _, c := range cands {
		if _, ok := cols[strings.ToLower(c)]; ok {
			return c
		}
	}
	return ""
}

func ValidateAndSnapshotCSST(
	tx *gorm.DB,
	expectSchoolID uuid.UUID,
	csstID uuid.UUID,
) (*CSSTSnapshot, error) {
	cols, err := tableColumns(tx, "class_section_subject_teachers")
	if err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal membaca metadata skema CSST")
	}

	idCol := firstExisting(cols, "class_section_subject_teacher_id", "id")
	schoolCol := firstExisting(cols, "class_section_subject_teacher_school_id", "school_id")
	nameCol := firstExisting(cols, "class_section_subject_teacher_name", "name")
	teacherCol := firstExisting(cols,
		"class_section_subject_teacher_teacher_id",
		"school_teacher_id",
		"teacher_id",
	)
	sectionCol := firstExisting(cols,
		"class_section_subject_teacher_section_id",
		"class_section_id",
		"section_id",
	)
	deletedCol := firstExisting(cols, "class_section_subject_teacher_deleted_at", "deleted_at")

	if idCol == "" || schoolCol == "" || nameCol == "" {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "CSST snapshot: kolom minimal tidak ditemukan")
	}

	teachExpr := "NULL::uuid"
	if teacherCol != "" {
		teachExpr = fmt.Sprintf("csst.%s", teacherCol)
	}
	secExpr := "NULL::uuid"
	if sectionCol != "" {
		secExpr = fmt.Sprintf("csst.%s", sectionCol)
	}
	whereDeleted := ""
	if deletedCol != "" {
		whereDeleted = fmt.Sprintf(" AND csst.%s IS NULL", deletedCol)
	}

	q := fmt.Sprintf(`
SELECT
  csst.%s::text AS school_id,
  csst.%s       AS name,
  %s            AS teacher_id,
  %s            AS section_id
FROM class_section_subject_teachers csst
WHERE csst.%s = ?
%s
LIMIT 1`,
		schoolCol, nameCol, teachExpr, secExpr, idCol, whereDeleted,
	)

	var row struct {
		SchoolID  string     `gorm:"column:school_id"`
		Name      *string    `gorm:"column:name"`
		TeacherID *uuid.UUID `gorm:"column:teacher_id"`
		SectionID *uuid.UUID `gorm:"column:section_id"`
	}
	if err := tx.Raw(q, csstID).Scan(&row).Error; err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal memuat CSST")
	}
	if strings.TrimSpace(row.SchoolID) == "" {
		return nil, fiber.NewError(fiber.StatusBadRequest, "CSST tidak ditemukan")
	}

	rmz, perr := uuid.Parse(strings.TrimSpace(row.SchoolID))
	if perr != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Format school_id CSST tidak valid")
	}
	if expectSchoolID != uuid.Nil && rmz != uuid.Nil && rmz != expectSchoolID {
		return nil, fiber.NewError(fiber.StatusForbidden, "CSST bukan milik school Anda")
	}

	trimPtr := func(p *string) *string {
		if p == nil {
			return nil
		}
		v := strings.TrimSpace(*p)
		if v == "" {
			return nil
		}
		return &v
	}

	return &CSSTSnapshot{
		Name:      trimPtr(row.Name),
		TeacherID: row.TeacherID,
		SectionID: row.SectionID,
	}, nil
}

func ToJSON(cs *CSSTSnapshot) datatypes.JSON {
	if cs == nil {
		return datatypes.JSON([]byte("null"))
	}
	m := map[string]any{
		"captured_at": time.Now().UTC(),
		"source":      "generator_v2",
	}
	if cs.Name != nil && strings.TrimSpace(*cs.Name) != "" {
		m["name"] = *cs.Name
	}
	if cs.TeacherID != nil {
		m["teacher_id"] = *cs.TeacherID
	}
	if cs.SectionID != nil {
		m["section_id"] = *cs.SectionID
	}
	b, _ := json.Marshal(m)
	return datatypes.JSON(b)
}
