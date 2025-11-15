// file: internals/helpers/json.go
package helper

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

/* ===============================
   Pagination type & defaults
=================================*/

type Pagination struct {
	Page           int   `json:"page"`
	PerPage        int   `json:"per_page"`
	Total          int64 `json:"total"`
	TotalPages     int   `json:"total_pages"`
	HasNext        bool  `json:"has_next"`
	HasPrev        bool  `json:"has_prev"`
	Count          int   `json:"count"`                      // jumlah item di halaman ini
	PerPageOptions []int `json:"per_page_options,omitempty"` // opsi per_page yg disarankan
}

var defaultPerPageOptions = []int{10, 20, 30, 50, 100}

func SetDefaultPerPageOptions(opts []int) {
	if len(opts) > 0 {
		defaultPerPageOptions = opts
	}
}

/* ===============================
   Paging resolver (query → page/perPage/offset)
=================================*/

type Paging struct {
	Page    int
	PerPage int
	Offset  int
	Limit   int
}

// ResolvePaging membaca ?page= & ?per_page= (atau alias ?limit=) dan normalisasi.
// - defaultPerPage: fallback kalau tidak ada/invalid
// - maxPerPage: batasi per_page maksimum (0 = tanpa batas)
func ResolvePaging(c *fiber.Ctx, defaultPerPage, maxPerPage int) Paging {
	pageStr := strings.TrimSpace(c.Query("page", "1"))

	// dukung dua nama: per_page (utama) atau limit (alias lama)
	perPageStr := strings.TrimSpace(c.Query("per_page"))
	if perPageStr == "" {
		perPageStr = strings.TrimSpace(c.Query("limit", strconv.Itoa(defaultPerPage)))
	}

	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}

	perPage, _ := strconv.Atoi(perPageStr)
	if perPage <= 0 {
		perPage = defaultPerPage
	}
	if maxPerPage > 0 && perPage > maxPerPage {
		perPage = maxPerPage
	}

	offset := (page - 1) * perPage

	return Paging{
		Page:    page,
		PerPage: perPage,
		Offset:  offset,
		Limit:   perPage,
	}
}

/* ===============================
   Pagination builders
=================================*/

func BuildPaginationFromOffset(total int64, offset, limit int) Pagination {
	perPage := limit
	if perPage <= 0 {
		perPage = 20 // default aman
	}
	page := (offset / perPage) + 1
	if page <= 0 {
		page = 1
	}
	totalPages := int((total + int64(perPage) - 1) / int64(perPage)) // ceil
	if totalPages == 0 {
		totalPages = 1
	}
	return Pagination{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}
}

func BuildPaginationFromPage(total int64, page, perPage int) Pagination {
	if perPage <= 0 {
		perPage = 20
	}
	if page <= 0 {
		page = 1
	}
	totalPages := int((total + int64(perPage) - 1) / int64(perPage)) // ceil
	if totalPages == 0 {
		totalPages = 1
	}
	return Pagination{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}
}

/* ===============================
   Internal helpers
=================================*/

func lenOf(v any) int {
	if v == nil {
		return 0
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map, reflect.String:
		return rv.Len()
	default:
		return 0
	}
}

// coercePagination: terima Pagination / *Pagination / fiber.Map / map[string]any → Pagination
func coercePagination(p any) (Pagination, bool) {
	switch t := p.(type) {
	case nil:
		return Pagination{}, false
	case Pagination:
		return t, true
	case *Pagination:
		if t == nil {
			return Pagination{}, false
		}
		return *t, true
	case fiber.Map:
		out := Pagination{}
		if v, ok := t["page"].(int); ok {
			out.Page = v
		}
		if v, ok := t["per_page"].(int); ok {
			out.PerPage = v
		}
		if v, ok := t["total"].(int64); ok {
			out.Total = v
		} else if vInt, ok := t["total"].(int); ok {
			out.Total = int64(vInt)
		}
		if v, ok := t["total_pages"].(int); ok {
			out.TotalPages = v
		}
		if v, ok := t["has_next"].(bool); ok {
			out.HasNext = v
		}
		if v, ok := t["has_prev"].(bool); ok {
			out.HasPrev = v
		}
		if v, ok := t["count"].(int); ok {
			out.Count = v
		}
		if v, ok := t["per_page_options"].([]int); ok {
			out.PerPageOptions = v
		}
		// minimal valid
		if out.Page > 0 && out.PerPage > 0 && out.Total >= 0 {
			if out.TotalPages <= 0 {
				out = BuildPaginationFromPage(out.Total, out.Page, out.PerPage)
			}
			return out, true
		}
		return Pagination{}, false
	case map[string]any:
		return coercePagination(fiber.Map(t))
	default:
		return Pagination{}, false
	}
}

func enrichPaginationWithCountAndOpts(p *Pagination, data any) {
	// isi count jika belum ada
	if p.Count == 0 {
		if n := lenOf(data); n > 0 {
			p.Count = n
		}
	}
	// isi per_page_options kalau kosong
	if len(p.PerPageOptions) == 0 && len(defaultPerPageOptions) > 0 {
		p.PerPageOptions = append([]int(nil), defaultPerPageOptions...)
	}
}

/* ===============================
   Error helpers (standard shape)
=================================*/

type ErrorResponse struct {
	Success   bool                `json:"success"`
	Message   string              `json:"message"`
	ErrorCode string              `json:"error_code,omitempty"`
	Errors    map[string][]string `json:"errors,omitempty"`
}

func statusToErrorCode(status int) string {
	switch status {
	case fiber.StatusBadRequest:
		return "BAD_REQUEST"
	case fiber.StatusUnauthorized:
		return "UNAUTHORIZED"
	case fiber.StatusForbidden:
		return "FORBIDDEN"
	case fiber.StatusNotFound:
		return "NOT_FOUND"
	case fiber.StatusUnprocessableEntity:
		return "VALIDATION_ERROR"
	case fiber.StatusConflict:
		return "CONFLICT"
	default:
		if status >= 500 {
			return "INTERNAL_ERROR"
		}
		return "ERROR"
	}
}

// JsonError: error generic (bukan validasi)
func JsonError(c *fiber.Ctx, status int, message string) error {
	if strings.TrimSpace(message) == "" {
		// fallback message default dari Fiber kalau kosong
		if fe := fiber.ErrInternalServerError; status == 0 || status >= 500 {
			message = fe.Message
		}
	}
	if status == 0 {
		status = fiber.StatusInternalServerError
	}

	resp := ErrorResponse{
		Success:   false,
		Message:   message,
		ErrorCode: statusToErrorCode(status),
	}
	return c.Status(status).JSON(resp)
}

// JsonValidationError: khusus error validasi (422)
func JsonValidationError(c *fiber.Ctx, fieldErrors map[string][]string) error {
	if fieldErrors == nil {
		fieldErrors = map[string][]string{}
	}
	resp := ErrorResponse{
		Success:   false,
		Message:   "validation failed",
		ErrorCode: "VALIDATION_ERROR",
		Errors:    fieldErrors,
	}
	return c.Status(fiber.StatusUnprocessableEntity).JSON(resp)
}

/* ===============================
   JSON responses (standard success)
=================================*/

// JsonList: list dengan pagination (GET /list dsb)
func JsonList(c *fiber.Ctx, message string, data any, pagination any) error {
	if strings.TrimSpace(message) == "" {
		message = "ok"
	}
	body := fiber.Map{
		"success": true,
		"message": message,
		"data":    data,
	}
	if p, ok := coercePagination(pagination); ok {
		enrichPaginationWithCountAndOpts(&p, data)
		body["pagination"] = p
	}
	return c.Status(fiber.StatusOK).JSON(body)
}

// JsonListEx: list + includes (misal dropdowns, metadata)
func JsonListEx(c *fiber.Ctx, message string, data any, pagination any, includes any) error {
	if strings.TrimSpace(message) == "" {
		message = "ok"
	}
	body := fiber.Map{
		"success": true,
		"message": message,
		"data":    data,
	}
	if p, ok := coercePagination(pagination); ok {
		enrichPaginationWithCountAndOpts(&p, data)
		body["pagination"] = p
	}
	if includes != nil {
		body["includes"] = includes
	}
	return c.Status(fiber.StatusOK).JSON(body)
}

// JsonOK: response sukses generic (GET detail, dsb)
func JsonOK(c *fiber.Ctx, message string, data any) error {
	if strings.TrimSpace(message) == "" {
		message = "ok"
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": message,
		"data":    data,
	})
}

// JsonCreated: response sukses create (POST)
func JsonCreated(c *fiber.Ctx, message string, data any) error {
	if strings.TrimSpace(message) == "" {
		message = "created"
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": message,
		"data":    data,
	})
}

// JsonUpdated: response sukses update (PATCH/PUT)
func JsonUpdated(c *fiber.Ctx, message string, data any) error {
	if strings.TrimSpace(message) == "" {
		message = "updated"
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": message,
		"data":    data,
	})
}

// JsonDeleted: response sukses delete (DELETE)
func JsonDeleted(c *fiber.Ctx, message string, data any) error {
	if strings.TrimSpace(message) == "" {
		message = "deleted"
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": message,
		"data":    data,
	})
}
