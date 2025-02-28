package routers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sh5080/ndns-router/pkg/controllers"
	"github.com/sh5080/ndns-router/pkg/interfaces"
)

// SetupServerRoutes는 /api/servers 경로의 라우터를 설정합니다
func SetupServerRoutes(router fiber.Router, serverService interfaces.ServerService) error {
	controller := controllers.NewServerController(serverService)

	{
		// 서버 상태 목록 조회
		router.Get("/", controller.HandleServersStatus)
		// 서버 추가
		router.Post("/add", controller.HandleAddServer)
		// 서버 제거
		router.Delete("/remove", controller.HandleRemoveServer)
	}

	return nil
}
