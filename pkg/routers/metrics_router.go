package routers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sh5080/ndns-router/pkg/controllers"
	"github.com/sh5080/ndns-router/pkg/interfaces"
)

// SetupMetricsRoutes는 /api/metrics 경로의 라우터를 설정합니다
func SetupMetricsRoutes(router fiber.Router, serverService interfaces.ServerService) error {
	controller := controllers.NewMetricsController(serverService)
	{
		// 메트릭 업데이트
		router.Post("/update", controller.HandleMetricsUpdate)
	}

	return nil
}
