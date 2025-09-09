package helper

// import "github.com/gofiber/fiber/v2"

// FromFiberError mengubah error hasil Transaction (biasanya *fiber.Error)
// menjadi response JSON konsisten via helper.Error.
// Jika bukan *fiber.Error, fallback ke 500 dengan pesan asli.
