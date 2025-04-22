package routers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sh5080/ndns-router/pkg/middlewares"
	"github.com/sh5080/ndns-router/pkg/services"
	"github.com/sh5080/ndns-router/pkg/types"
	"github.com/sh5080/ndns-router/pkg/utils"
)

// SetupRoutes는 애플리케이션의 모든 라우트를 설정합니다
func SetupRoutes(app *fiber.App, config *types.RouterConfig) error {
	// 서비스 초기화
	serverService, err := services.NewServerService()
	if err != nil {
		return err
	}

	// API 라우터 그룹
	api := app.Group("/api")

	// 서버 관련 라우터 (/api/servers/*)
	servers := api.Group("/servers")
	if err := SetupServerRoutes(servers, serverService); err != nil {
		return err
	}

	// 메트릭 관련 라우터 (/api/metrics/*)
	metrics := api.Group("/metrics")
	if err := SetupMetricsRoutes(metrics, serverService); err != nil {
		return err
	}

	// Internal API 라우터 (/api/internal/*)
	internal := api.Group("/internal")
	if err := SetupInternalRoutes(internal, serverService); err != nil {
		return err
	}

	// 프록시 미들웨어 설정 (/)
	app.Use(middlewares.NewProxyMiddleware(serverService))

	utils.Info("라우터 설정이 완료되었습니다")
	return nil
}
