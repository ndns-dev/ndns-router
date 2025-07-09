package routers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sh5080/ndns-router/pkg/middlewares"
	"github.com/sh5080/ndns-router/pkg/services"
	"github.com/sh5080/ndns-router/pkg/utils"
)

// SetupRoutes는 애플리케이션의 모든 라우트를 설정합니다
func SetupRoutes(app *fiber.App) error {
	// 서비스 초기화
	serverService, err := services.NewServerService()
	if err != nil {
		return err
	}

	// 프록시 미들웨어를 먼저 설정 (모든 요청에 대해 먼저 검사)
	app.Use(middlewares.NewProxyMiddleware(serverService))

	// 내부 관리용 라우터 설정
	servers := app.Group("/servers")
	if err := SetupServerRoutes(servers, serverService); err != nil {
		return err
	}

	metrics := app.Group("/metrics")
	if err := SetupMetricsRoutes(metrics, serverService); err != nil {
		return err
	}

	internal := app.Group("/internal")
	if err := SetupInternalRoutes(internal, serverService); err != nil {
		return err
	}

	external := app.Group("/external")
	if err := SetupExternalRoutes(external, serverService); err != nil {
		return err
	}

	utils.Info("라우터 설정이 완료되었습니다")
	return nil
}
