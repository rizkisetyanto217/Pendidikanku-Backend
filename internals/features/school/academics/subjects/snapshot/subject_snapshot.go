// file: internals/services/snapsvc/snapsvc.go
package snapsvc

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrMasjidMismatch = errors.New("masjid mismatch")

// Struktur snapshot untuk Subject (sinkron dengan field snapshot di ClassSubject)
type SubjectSnapshot struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Code string    `json:"code"`
	Slug string    `json:"slug"`
	URL  *string   `json:"url,omitempty"` // dipetakan dari subjects.subject_image_url (atau ganti sesuai kolom sumber kamu)
}

func BuildSubjectSnapshot(
	ctx context.Context,
	tx *gorm.DB,
	masjidID uuid.UUID,
	subjectID uuid.UUID,
) (*SubjectSnapshot, error) {
	var row struct {
		MasjidID uuid.UUID
		ID       uuid.UUID
		Name     string
		Code     string
		Slug     string
		URL      *string
	}

	if err := tx.WithContext(ctx).Raw(`
		SELECT
			s.subject_masjid_id AS masjid_id,
			s.subject_id        AS id,
			s.subject_name      AS name,
			s.subject_code      AS code,
			s.subject_slug      AS slug,
			s.subject_image_url AS url
		FROM subjects s
		WHERE s.subject_id = ?
		  AND s.subject_deleted_at IS NULL
	`, subjectID).Scan(&row).Error; err != nil {
		return nil, err
	}

	// not found
	if row.MasjidID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}

	// ✅ tenant check (kembalikan ErrMasjidMismatch)
	if row.MasjidID != masjidID {
		return nil, ErrMasjidMismatch
	}

	nz := func(p *string) *string {
		if p == nil {
			return nil
		}
		v := strings.TrimSpace(*p)
		if v == "" {
			return nil
		}
		return &v
	}

	return &SubjectSnapshot{
		ID:   row.ID,
		Name: strings.TrimSpace(row.Name),
		Code: strings.TrimSpace(row.Code),
		Slug: strings.TrimSpace(row.Slug),
		URL:  nz(row.URL),
	}, nil
}
