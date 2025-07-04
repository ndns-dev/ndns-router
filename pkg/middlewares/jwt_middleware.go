package middlewares

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/sh5080/ndns-router/pkg/utils"
)

func JwtMiddleware() fiber.Handler {
	return func(ctx *fiber.Ctx) error {

		auth := ctx.Get("Authorization")
		if auth == "" {
			return utils.SendError(ctx, fiber.StatusUnauthorized, "Missing token")
		}

		parts := strings.Split(auth, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return utils.SendError(ctx, fiber.StatusUnauthorized, "Invalid token format")
		}

		_, err := utils.ParseAndValidateSseToken(parts[1])
		if err != nil {
			return utils.SendError(ctx, fiber.StatusUnauthorized, "Invalid or expired token")
		}
		ctx.Locals("reqId", ctx.Query("reqId"))
		return ctx.Next()
	}
}
