package routers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sh5080/ndns-router/pkg/controllers"
	"github.com/sh5080/ndns-router/pkg/interfaces"
	"github.com/sh5080/ndns-router/pkg/middlewares"
)

// SetupExternalRoutes는 /external 경로의 라우터를 설정합니다
func SetupExternalRoutes(router fiber.Router, serverService interfaces.ServerService) error {
	controller := controllers.NewExternalController(serverService)
	// Sse 연결 라우터
	stream := router.Group("/stream")
	{
		// Sse 연결
		stream.Get("/", middlewares.JwtMiddleware(), controller.SseHandler)

		// Sse 전송
		stream.Post("/", controller.SendMessage)

		// Sse 연결 조회
		stream.Get("/connections", controller.GetActiveConnections)
	}

	return nil
}
