package middlewares

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/proxy"
	"github.com/sh5080/ndns-router/pkg/configs"
	"github.com/sh5080/ndns-router/pkg/interfaces"
	"github.com/sh5080/ndns-router/pkg/types"
	"github.com/sh5080/ndns-router/pkg/utils"
)

// NewProxyMiddleware creates a new proxy middleware
func NewProxyMiddleware(serverService interfaces.ServerService) fiber.Handler {
	pathUtil := utils.NewPath(configs.InternalPaths)

	// 서버 요청 시도
	tryServer := func(c *fiber.Ctx, server *types.Server, requestId string) error {
		// [2] 선택된 서버로 요청 준비
		targetURL := server.URL
		if idx := strings.Index(targetURL, ":"); idx != -1 {
			targetURL = targetURL[:idx]
		}
		if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
			targetURL = "http://" + targetURL
		}

		fullURL := targetURL + c.Path()
		if c.Request().URI().QueryString() != nil {
			fullURL += "?" + string(c.Request().URI().QueryString())
		}

		// [3] 요청 헤더 설정
		c.Request().Header.Set("X-Forwarded-Host", string(c.Request().Header.Host()))
		c.Request().Header.Set("X-Origin-Host", server.ServerId)
		c.Request().Header.Set("X-App-Name", server.ServerId)
		c.Request().Header.Set("X-Request-ID", requestId)

		utils.Infof("[%s] 프록시 요청 시도: %s -> %s", requestId, c.Path(), fullURL)

		// [4] 타임아웃 컨텍스트 설정
		ctx, cancel := context.WithTimeout(context.Background(), configs.ProxyTimeout)
		defer cancel()

		// 프록시 요청 전달
		errCh := make(chan error, 1)
		go func() {
			errCh <- proxy.Do(c, fullURL)
		}()

		// [5] 타임아웃 또는 응답 대기
		select {
		case err := <-errCh:
			if err != nil {
				utils.Errorf("[%s] 프록시 요청 실패 (%s): %v", requestId, server.ServerId, err)
				return err
			}
			utils.Infof("[%s] 프록시 요청 성공 (%s)", requestId, server.ServerId)
			return nil
		case <-ctx.Done():
			utils.Errorf("[%s] 프록시 요청 타임아웃 (%s)", requestId, server.ServerId)
			return fmt.Errorf("request timeout")
		}
	}

	return func(c *fiber.Ctx) error {
		requestId := utils.NewGenerate().GenerateRequestId()
		utils.Infof("[%s] 새로운 프록시 요청 시작: %s %s", requestId, c.Method(), c.Path())

		path := c.Path()
		if pathUtil.IsInternalPath(path) {
			utils.Infof("[%s] 내부 관리 경로 감지 (%s), 직접 처리", requestId, path)
			return c.Next()
		}

		// [1] 최적의 서버 선택
		selectedServer := serverService.SelectOptimalServer()
		utils.Infof("[%s] 선택된 서버: %s (점수: %.2f)", requestId, selectedServer.ServerId, selectedServer.Metrics.Score)

		// [2-5] 선택된 서버로 요청 시도
		err := tryServer(c, selectedServer, requestId)
		if err != nil {
			utils.Warnf("[%s] 기본 서버 실패, 서버리스로 장애 조치", requestId)

			// [6] 서버리스로 fallback
			fallbackServer := serverService.GetServerlessServer()
			err = tryServer(c, fallbackServer, requestId)
			if err != nil {
				utils.Errorf("[%s] 서버리스 장애 조치 실패", requestId)
				return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
					"success": false,
					"message": "모든 서버 요청 실패",
				})
			}
		}

		// [7] 요청 완료 처리
		serverService.FinishUsingServer(selectedServer.ServerId)
		utils.Infof("[%s] 프록시 요청 완료", requestId)
		return nil
	}
}
