package middlewares

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/proxy"
	"github.com/sh5080/ndns-router/pkg/configs"
	"github.com/sh5080/ndns-router/pkg/interfaces"
	"github.com/sh5080/ndns-router/pkg/utils"
)

// NewProxyMiddleware는 프록시 미들웨어를 생성합니다
func NewProxyMiddleware(serverService interfaces.ServerService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// API 요청은 프록시하지 않음
		if strings.HasPrefix(c.Path(), "/api") {
			return c.Next()
		}

		// 모든 서버 조회
		servers, err := serverService.GetAllServers()
		if err != nil {
			utils.Errorf("서버 목록 조회 실패: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": "서버 목록 조회 실패",
			})
		}

		if len(servers) == 0 {
			utils.Info("온프레미스 서버가 없습니다. 서버리스 서버로 전환합니다.")

			// 서버리스 서버 목록 가져오기
			serverlessServers := configs.GetConfig().Serverless.Servers
			if len(serverlessServers) == 0 {
				utils.Error("사용 가능한 서버가 없습니다 (온프레미스 및 서버리스)")
				return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
					"success": false,
					"message": "사용 가능한 서버가 없습니다",
				})
			}

			// 라운드 로빈으로 서버리스 서버 선택
			selectedServer := utils.NewRoundRobin().Next(serverlessServers)
			utils.Infof("서버리스 서버로 요청을 전달합니다: %s", selectedServer)

			// URL 유효성 검사
			if _, err := url.Parse(selectedServer); err != nil {
				utils.Errorf("URL 파싱 실패: %v", err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"success": false,
					"message": "서버 URL이 잘못되었습니다",
				})
			}

			// 프록시 요청 전송
			return proxy.Do(c, selectedServer)
		}

		// 가장 건강한 서버 선택
		var bestServer = servers[0]
		for _, server := range servers[1:] {
			if server.Metrics.Score > bestServer.Metrics.Score {
				bestServer = server
			}
		}

		// URL 파싱
		targetURL, err := url.Parse(bestServer.URL)
		if err != nil {
			utils.Errorf("URL 파싱 실패 (%s): %v", bestServer.URL, err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": fmt.Sprintf("잘못된 서버 URL: %s", bestServer.URL),
			})
		}

		// 원본 요청 경로 유지
		targetURL.Path = c.Path()
		if c.Request().URI().QueryString() != nil {
			targetURL.RawQuery = string(c.Request().URI().QueryString())
		}

		// 요청 헤더 설정
		c.Request().Header.Set("X-Forwarded-Host", string(c.Request().Header.Host()))
		c.Request().Header.Set("X-Origin-Host", targetURL.Host)
		c.Request().Header.Set("X-App-Name", bestServer.ServerId)

		utils.Infof("프록시 요청: %s -> %s", c.Path(), targetURL.String())

		// 프록시 요청 전달
		if err := proxy.Do(c, targetURL.String()); err != nil {
			utils.Errorf("프록시 요청 실패: %v", err)
			return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
				"success": false,
				"message": "프록시 요청 실패",
			})
		}

		return nil
	}
}
