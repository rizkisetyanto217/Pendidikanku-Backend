// file: internals/shared/dbtypes/tod.go
package dbtime

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Tod struct{ time.Time }

// From: bikin Tod dari time.Time (ambil HH:mm:ss, buang tanggal & zona)
func From(t time.Time) Tod {
	return Tod{
		Time: time.Date(0, 1, 1, t.Hour(), t.Minute(), t.Second(), 0, time.UTC),
	}
}

// Parse: bikin Tod dari string "HH:mm[:ss]"
func Parse(s string) (Tod, error) {
	var tt Tod
	return tt, tt.parse(s)
}

// Scan: terima time.Time atau string ("HH:MM[:SS]")
func (t *Tod) Scan(v any) error {
	switch x := v.(type) {
	case time.Time:
		t.Time = x
		return nil
	case []byte:
		return t.parse(string(x))
	case string:
		return t.parse(x)
	case nil:
		t.Time = time.Time{}
		return nil
	default:
		return fmt.Errorf("tod: unsupported Scan type %T", v)
	}
}

func (t *Tod) parse(s string) error {
	s = strings.TrimSpace(s)
	if len(s) == 5 { // "HH:MM"
		s += ":00"
	}
	tt, err := time.Parse("15:04:05", s)
	if err != nil {
		return err
	}
	t.Time = tt
	return nil
}

// Value: kirim "HH:MM:SS" agar Postgres TIME paham
func (t Tod) Value() (driver.Value, error) {
	if t.Time.IsZero() {
		return "00:00:00", nil
	}
	return t.Format("15:04:05"), nil
}

// (opsional) JSON codec biar konsisten
func (t Tod) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.Format("15:04:05"))
}
func (t *Tod) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	return t.parse(s)
}
