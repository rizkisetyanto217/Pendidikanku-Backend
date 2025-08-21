package helper

import "github.com/gofiber/fiber/v2"

// FromFiberError mengubah error hasil Transaction (biasanya *fiber.Error)
// menjadi response JSON konsisten via helper.Error.
// Jika bukan *fiber.Error, fallback ke 500 dengan pesan asli.
func FromFiberError(c *fiber.Ctx, err error) error {
	if fe, ok := err.(*fiber.Error); ok {
		return Error(c, fe.Code, fe.Message)
	}
	return Error(c, fiber.StatusInternalServerError, err.Error())
}
