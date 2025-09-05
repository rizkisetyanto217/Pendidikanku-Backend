// pkg/pagination/pagination.go
package helper

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

const (
	DefaultPage = 1
)

type Options struct {
	DefaultPerPage int
	MaxPerPage     int
	AllowAll       bool // izinkan per_page=all
	AllHardCap     int  // batas saat all
}

// ===== Preset =====
var (
	DefaultOpts = Options{DefaultPerPage: 25, MaxPerPage: 200}
	AdminOpts   = Options{DefaultPerPage: 50, MaxPerPage: 500}
	ExportOpts  = Options{DefaultPerPage: 100, MaxPerPage: 1000, AllowAll: true, AllHardCap: 10_000}
)

type Params struct {
	Page      int
	PerPage   int
	SortBy    string
	SortOrder string // asc|desc
	All       bool   // true jika per_page=all dipakai
}

// Parse default (global)
func Parse(r *http.Request, defaultSortBy, defaultSortOrder string) Params {
	return ParseWith(r, defaultSortBy, defaultSortOrder, DefaultOpts)
}

// Parse dengan preset
func ParseWith(r *http.Request, defaultSortBy, defaultSortOrder string, opt Options) Params {
	q := r.URL.Query()

	page := atoiDefault(q.Get("page"), DefaultPage)
	if page < 1 {
		page = DefaultPage
	}

	perRaw := strings.TrimSpace(firstNonEmpty(q.Get("per_page"), q.Get("limit")))
	all := false
	per := opt.DefaultPerPage

	if opt.AllowAll && strings.EqualFold(perRaw, "all") {
		all = true
		page = 1
		if opt.AllHardCap > 0 {
			per = opt.AllHardCap
		} else {
			per = opt.MaxPerPage
		}
	} else {
		if n, err := strconv.Atoi(perRaw); err == nil && n > 0 {
			per = n
		}
		if per > opt.MaxPerPage {
			per = opt.MaxPerPage
		}
		if per < 1 {
			per = opt.DefaultPerPage
		}
	}

	sortBy := strings.TrimSpace(q.Get("sort_by"))
	if sortBy == "" {
		sortBy = defaultSortBy
	}
	order := strings.ToLower(strings.TrimSpace(firstNonEmpty(q.Get("order"), q.Get("sort"))))
	if order != "asc" && order != "desc" {
		order = strings.ToLower(defaultSortOrder)
		if order != "asc" && order != "desc" {
			order = "desc"
		}
	}

	return Params{
		Page:      page,
		PerPage:   per,
		SortBy:    sortBy,
		SortOrder: order,
		All:       all,
	}
}

// Helpers sebelumnya (Limit, Offset, SafeOrderClause, BuildMeta, AddLinkHeaders, dll.)
// tetap sama — cukup ganti Parse() dipanggil sesuai preset.

func atoiDefault(s string, def int) int { n, err := strconv.Atoi(s); if err != nil { return def }; return n }
func firstNonEmpty(a, b string) string  { if strings.TrimSpace(a) != "" { return a }; return b }


// Limit & Offset
func (p Params) Limit() int  { return p.PerPage }
func (p Params) Offset() int { return (p.Page - 1) * p.PerPage }

// ORDER BY aman (kolom dari whitelist)
func (p Params) SafeOrderClause(allowed map[string]string, defaultKey string) (string, error) {
	key := p.SortBy
	if key == "" {
		key = defaultKey
	}
	col, ok := allowed[key]
	if !ok {
		col, ok = allowed[defaultKey]
		if !ok {
			return "", fmt.Errorf("no valid default sort key")
		}
	}
	dir := "DESC"
	if strings.ToLower(p.SortOrder) == "asc" {
		dir = "ASC"
	}
	return "ORDER BY " + col + " " + dir, nil
}

// Meta untuk response
type Meta struct {
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
	HasNext    bool  `json:"has_next"`
	HasPrev    bool  `json:"has_prev"`
	NextPage   *int  `json:"next_page,omitempty"`
	PrevPage   *int  `json:"prev_page,omitempty"`
}

func BuildMeta(total int64, p Params) Meta {
	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(p.PerPage)))
	}
	meta := Meta{
		Page:       p.Page,
		PerPage:    p.PerPage,
		Total:      total,
		TotalPages: totalPages,
		HasPrev:    p.Page > 1,
		HasNext:    totalPages > 0 && p.Page < totalPages,
	}
	if meta.HasPrev {
		prev := p.Page - 1
		meta.PrevPage = &prev
	}
	if meta.HasNext {
		next := p.Page + 1
		meta.NextPage = &next
	}
	return meta
}


// ParseFiber: parse pagination/sorting langsung dari Fiber ctx.
// Menggunakan Options/Params/DefaultPage/atoiDefault/firstNonEmpty yang sudah ada di package helper.
func ParseFiber(c *fiber.Ctx, defaultSortBy, defaultSortOrder string, opt Options) Params {
	q := c.Queries()

	page := atoiDefault(q["page"], DefaultPage)
	if page < 1 {
		page = DefaultPage
	}

	perRaw := strings.TrimSpace(firstNonEmpty(q["per_page"], q["limit"]))
	all := false
	per := opt.DefaultPerPage

	if opt.AllowAll && strings.EqualFold(perRaw, "all") {
		all = true
		page = 1
		if opt.AllHardCap > 0 {
			per = opt.AllHardCap
		} else {
			per = opt.MaxPerPage
		}
	} else {
		if n, err := strconv.Atoi(perRaw); err == nil && n > 0 {
			per = n
		}
		if per > opt.MaxPerPage {
			per = opt.MaxPerPage
		}
		if per < 1 {
			per = opt.DefaultPerPage
		}
	}

	sortBy := strings.TrimSpace(q["sort_by"])
	if sortBy == "" {
		sortBy = defaultSortBy
	}

	order := strings.ToLower(strings.TrimSpace(firstNonEmpty(q["order"], q["sort"])))
	if order != "asc" && order != "desc" {
		order = strings.ToLower(defaultSortOrder)
		if order != "asc" && order != "desc" {
			order = "desc"
		}
	}

	return Params{
		Page:      page,
		PerPage:   per,
		SortBy:    sortBy,
		SortOrder: order,
		All:       all,
	}
}
