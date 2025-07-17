package helper

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

// ✅ Success Response tanpa custom code (default 200)
func Success(c *fiber.Ctx, message string, data interface{}) error {
	return SuccessWithCode(c, fiber.StatusOK, message, data)
}

// ✅ Success Response dengan custom code (contoh 201 untuk created)
func SuccessWithCode(c *fiber.Ctx, code int, message string, data interface{}) error {
	return c.Status(code).JSON(fiber.Map{
		"code":    code,
		"status":  "success",
		"message": message,
		"data":    data,
	})
}

// ✅ Error Response sederhana
func Error(c *fiber.Ctx, code int, message string) error {
	return c.Status(code).JSON(fiber.Map{
		"code":    code,
		"status":  "error",
		"message": message,
	})
}

// ✅ Error Response advance (opsional), bisa kirim multiple field error
func ErrorWithDetails(c *fiber.Ctx, code int, message string, errors interface{}) error {
	return c.Status(code).JSON(fiber.Map{
		"code":    code,
		"status":  "error",
		"message": message,
		"errors":  errors,
	})
}


// ✅ Khusus error validasi (validator.v10)
func ValidationError(c *fiber.Ctx, err error) error {
	var ve validator.ValidationErrors
	if errors, ok := err.(validator.ValidationErrors); ok {
		ve = errors
	} else {
		return Error(c, fiber.StatusBadRequest, "Invalid input")
	}

	errorsMap := make(map[string]string)
	for _, fieldErr := range ve {
		errorsMap[fieldErr.Field()] = fieldErr.Tag() // bisa diganti jadi pesan kustom
	}

	return ErrorWithDetails(c, fiber.StatusBadRequest, "Validasi gagal", errorsMap)
}