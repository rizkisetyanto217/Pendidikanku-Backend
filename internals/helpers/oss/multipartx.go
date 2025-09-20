// file: internals/helpers/multipartx/multipartx.go
package helper

import (
	"mime/multipart"
	"regexp"
	"strconv"
	"strings"
)

// ==============================
// File collector
// ==============================

type CollectOptions struct {
	// Urutan kandidat nama field multipart untuk file (boleh kosong -> pakai default)
	FileFieldCandidates []string
}

// Default kandidat nama field yg umum dipakai FE/Postman
var defaultFileFieldCandidates = []string{
	"files[]", "files", "file",
	"attachments[]", "attachments",
	"uploads[]", "uploads", "upload[]", "upload",
}

// CollectUploadFiles mengumpulkan semua *FileHeader dari form multipart,
// dengan urutan preferensi berdasarkan kandidat field yang diberikan.
// Mengembalikan: daftar file dan daftar key yang dipakai.
func CollectUploadFiles(form *multipart.Form, opt *CollectOptions) (out []*multipart.FileHeader, usedKeys []string) {
	if form == nil || form.File == nil {
		return nil, nil
	}
	candidates := defaultFileFieldCandidates
	if opt != nil && len(opt.FileFieldCandidates) > 0 {
		candidates = opt.FileFieldCandidates
	}

	seen := map[string]bool{}
	for _, key := range candidates {
		if fhs, ok := form.File[key]; ok && len(fhs) > 0 {
			usedKeys = append(usedKeys, key)
			for _, fh := range fhs {
				if fh != nil && fh.Filename != "" {
					out = append(out, fh)
				}
			}
			seen[key] = true
		}
	}
	// sweep semua key lain
	for key, fhs := range form.File {
		if seen[key] || len(fhs) == 0 {
			continue
		}
		hasFile := false
		for _, fh := range fhs {
			if fh != nil && fh.Filename != "" {
				out = append(out, fh)
				hasFile = true
			}
		}
		if hasFile {
			usedKeys = append(usedKeys, key)
		}
	}
	return out, usedKeys
}

// ==============================
// URL upserts parser (generic)
// ==============================

type URLUpsert struct {
	Kind      string // default "attachment"
	Label     *string
	Href      *string
	ObjectKey *string
	Order     int
	IsPrimary bool
}

// Normalisasi ringan (opsional)
func (u *URLUpsert) Normalize() {
	u.Kind = strings.TrimSpace(u.Kind)
	if u.Kind == "" {
		u.Kind = "attachment"
	}
	if u.Label != nil {
		v := strings.TrimSpace(*u.Label)
		if v == "" {
			u.Label = nil
		} else {
			u.Label = &v
		}
	}
	if u.Href != nil {
		v := strings.TrimSpace(*u.Href)
		if v == "" {
			u.Href = nil
		} else {
			u.Href = &v
		}
	}
	if u.ObjectKey != nil {
		v := strings.TrimSpace(*u.ObjectKey)
		if v == "" {
			u.ObjectKey = nil
		} else {
			u.ObjectKey = &v
		}
	}
}

// Konfigurasi parser
type URLParseOptions struct {
	// Prefix untuk bracket notation: default "urls" → urls[0][kind], dll
	BracketPrefix string
	// Default kind jika kosong
	DefaultKind string

	// Nama-nama field array style (boleh kosong → gunakan default konvensi)
	ArrayKindKey      string // default "url_kind[]"
	ArrayLabelKey     string // default "url_label[]"
	ArrayHrefKey      string // default "url_href[]"
	ArrayObjectKeyKey string // default "url_object_key[]"
	ArrayOrderKey     string // default "url_order[]"
	ArrayIsPrimaryKey string // default "url_is_primary[]"
}

func (o *URLParseOptions) withDefaults() *URLParseOptions {
	out := *o
	if out.BracketPrefix == "" {
		out.BracketPrefix = "urls"
	}
	if out.DefaultKind == "" {
		out.DefaultKind = "attachment"
	}
	if out.ArrayKindKey == "" {
		out.ArrayKindKey = "url_kind[]"
	}
	if out.ArrayLabelKey == "" {
		out.ArrayLabelKey = "url_label[]"
	}
	if out.ArrayHrefKey == "" {
		out.ArrayHrefKey = "url_href[]"
	}
	if out.ArrayObjectKeyKey == "" {
		out.ArrayObjectKeyKey = "url_object_key[]"
	}
	if out.ArrayOrderKey == "" {
		out.ArrayOrderKey = "url_order[]"
	}
	if out.ArrayIsPrimaryKey == "" {
		out.ArrayIsPrimaryKey = "url_is_primary[]"
	}
	return &out
}

// ParseURLUpsertsFromMultipart membaca metadata URL dari multipart.Form.
// Mendukung dua gaya:
// 1) Bracket notation:  <prefix>[0][kind], [label], [href], [object_key], [order], [is_primary]
// 2) Array style:       url_kind[], url_label[], url_href[], url_object_key[], url_order[], url_is_primary[]
func ParseURLUpsertsFromMultipart(form *multipart.Form, opt *URLParseOptions) []URLUpsert {
	if form == nil {
		return nil
	}
	o := (&URLParseOptions{}).withDefaults()
	if opt != nil {
		o = opt.withDefaults()
	}

	// ---------- 1) Bracket notation ----------
	re := regexp.MustCompile(`^` + regexp.QuoteMeta(o.BracketPrefix) + `\[(\d+)\]\[(kind|label|href|object_key|order|is_primary)\]$`)
	indexed := map[int]*URLUpsert{}
	for key, vals := range form.Value {
		m := re.FindStringSubmatch(key)
		if m == nil || len(vals) == 0 {
			continue
		}
		idx, _ := strconv.Atoi(m[1])
		field := m[2]
		if _, ok := indexed[idx]; !ok {
			indexed[idx] = &URLUpsert{Kind: o.DefaultKind}
		}
		v := strings.TrimSpace(vals[0])
		switch field {
		case "kind":
			if v != "" {
				indexed[idx].Kind = v
			}
		case "label":
			if v != "" {
				indexed[idx].Label = &v
			}
		case "href":
			if v != "" {
				indexed[idx].Href = &v
			}
		case "object_key":
			if v != "" {
				indexed[idx].ObjectKey = &v
			}
		case "order":
			if n, err := strconv.Atoi(v); err == nil {
				indexed[idx].Order = n
			}
		case "is_primary":
			if b, err := strconv.ParseBool(v); err == nil {
				indexed[idx].IsPrimary = b
			}
		}
	}
	out := make([]URLUpsert, 0, len(indexed))
	if len(indexed) > 0 {
		// Pastikan urut dari 0..N-1 jika ingin stabil
		for i := 0; i < len(indexed); i++ {
			if u, ok := indexed[i]; ok && u != nil {
				u.Normalize()
				out = append(out, *u)
			}
		}
		return out
	}

	// ---------- 2) Array style ----------
	kinds := form.Value[o.ArrayKindKey]
	labels := form.Value[o.ArrayLabelKey]
	hrefs := form.Value[o.ArrayHrefKey]
	objs := form.Value[o.ArrayObjectKeyKey]
	orders := form.Value[o.ArrayOrderKey]
	primaries := form.Value[o.ArrayIsPrimaryKey]

	maxN := 0
	for _, arr := range [][]string{kinds, labels, hrefs, objs, orders, primaries} {
		if len(arr) > maxN {
			maxN = len(arr)
		}
	}
	for i := 0; i < maxN; i++ {
		u := URLUpsert{Kind: o.DefaultKind}
		if i < len(kinds) && strings.TrimSpace(kinds[i]) != "" {
			u.Kind = strings.TrimSpace(kinds[i])
		}
		if i < len(labels) {
			if v := strings.TrimSpace(labels[i]); v != "" {
				u.Label = &v
			}
		}
		if i < len(hrefs) {
			if v := strings.TrimSpace(hrefs[i]); v != "" {
				u.Href = &v
			}
		}
		if i < len(objs) {
			if v := strings.TrimSpace(objs[i]); v != "" {
				u.ObjectKey = &v
			}
		}
		if i < len(orders) {
			if n, err := strconv.Atoi(strings.TrimSpace(orders[i])); err == nil {
				u.Order = n
			}
		}
		if i < len(primaries) {
			if b, err := strconv.ParseBool(strings.TrimSpace(primaries[i])); err == nil {
				u.IsPrimary = b
			}
		}
		u.Normalize()
		out = append(out, u)
	}
	return out
}
