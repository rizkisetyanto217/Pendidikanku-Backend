package helper

import "github.com/gofiber/fiber/v2"

// helper JSON response (urut: data lalu message)
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

func JsonDeleted(c *fiber.Ctx, message string, data any) error {
	// gunakan 200 agar bisa kirim body (bukan 204 No Content)
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data":    data,
		"message": message,
	})
}
