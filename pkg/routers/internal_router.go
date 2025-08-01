package routers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sh5080/ndns-router/pkg/controllers"
	"github.com/sh5080/ndns-router/pkg/interfaces"
)

// SetupInternalRoutes는 /internal 경로의 라우터를 설정합니다
func SetupInternalRoutes(router fiber.Router, serverService interfaces.ServerService) error {
	controller := controllers.NewInternalController(serverService)
	// 서버 관련 라우터
	server := router.Group("/server")
	{
		// 최적 서버 정보 업데이트
		server.Put("/optimal", controller.HandleOptimalServer)
	}
	// 분석 관련 라우터
	analysis := router.Group("/analysis")
	{
		// 분석결과 수신
		analysis.Post("/", controller.HandleAnalysis)
	}
	return nil
}
