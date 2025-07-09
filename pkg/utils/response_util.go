package utils

import "github.com/gofiber/fiber/v2"

func SendError(ctx *fiber.Ctx, status int, message string) error {
	return ctx.Status(status).JSON(fiber.Map{
		"message": message,
		"success": false,
	})
}

func SendSuccessMessage(ctx *fiber.Ctx, message string) error {
	return ctx.JSON(fiber.Map{
		"success": true,
		"message": message,
	})
}

func SendSuccessData(ctx *fiber.Ctx, data interface{}) error {
	return ctx.JSON(fiber.Map{
		"data":    data,
		"success": true,
	})
}
