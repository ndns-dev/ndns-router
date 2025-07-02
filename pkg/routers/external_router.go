package routers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sh5080/ndns-router/pkg/controllers"
	"github.com/sh5080/ndns-router/pkg/interfaces"
)

// SetupExternalRoutes는 /external 경로의 라우터를 설정합니다
func SetupExternalRoutes(router fiber.Router, serverService interfaces.ServerService) error {
	controller := controllers.NewExternalController(serverService)
	// Sse 연결 라우터
	stream := router.Group("/stream")
	{
		// Sse 연결
		stream.Get("/", controller.SseHandler)
	}

	return nil
}
