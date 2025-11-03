package helper

import "github.com/gofiber/fiber/v2"

// ===============================
// Response JSON standar
// ===============================

// Success: list dengan pagination
func JsonList(c *fiber.Ctx, data any, pagination any) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data":       data,
		"pagination": pagination,
	})
}

// Success: single resource (GET by ID, CREATE, UPDATE)
func JsonOK(c *fiber.Ctx, message string, data any) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data":    data,
		"message": message,
	})
}

func JsonCreated(c *fiber.Ctx, message string, data any) error {
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"data":    data,
		"message": message,
	})
}

func JsonUpdated(c *fiber.Ctx, message string, data any) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data":    data,
		"message": message,
	})
}

// Success: delete
// (200 agar bisa kirim body, bukan 204 No Content)
func JsonDeleted(c *fiber.Ctx, message string, data any) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data":    data,
		"message": message,
	})
}

// Error helper
func JsonError(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(fiber.Map{
		"error": fiber.Map{
			"code":    status,
			"message": message,
		},
	})
}

// Success: list + includes (opsional)
func JsonListEx(c *fiber.Ctx, data any, pagination any, includes any) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data":       data,
		"pagination": pagination,
		"includes":   includes,
	})
}
