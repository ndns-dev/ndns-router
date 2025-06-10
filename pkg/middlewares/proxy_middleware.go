package middlewares

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/proxy"
	"github.com/sh5080/ndns-router/pkg/configs"
	"github.com/sh5080/ndns-router/pkg/interfaces"
	"github.com/sh5080/ndns-router/pkg/types"
	"github.com/sh5080/ndns-router/pkg/utils"
)

func NewProxyMiddleware(serverService interfaces.ServerService) fiber.Handler {
	pathUtil := utils.NewPath(configs.InternalPaths)

	// 서버 요청 시도
	tryServer := func(c *fiber.Ctx, server *types.Server, requestId string) error {
		// [1] URL 정규화
		targetURL := server.URL
		if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
			targetURL = "https://" + targetURL
		}
		targetURL = strings.TrimSuffix(targetURL, "/")

		// [2] 전체 URL 구성
		fullURL := targetURL + c.Path()
		if c.Request().URI().QueryString() != nil {
			fullURL += "?" + string(c.Request().URI().QueryString())
		}

		// [3] 요청 헤더 설정
		c.Request().Header.Set("X-Forwarded-Host", string(c.Request().Header.Host()))
		c.Request().Header.Set("X-Origin-Host", server.ServerId)
		c.Request().Header.Set("X-App-Name", server.ServerId)
		c.Request().Header.Set("X-Request-ID", requestId)

		// [4] 프록시 요청 실행
		return proxy.DoRedirects(c, fullURL, configs.MaxRetryAttempts)
	}

	return func(c *fiber.Ctx) error {
		// [1] 요청 시작 및 초기화
		requestId := utils.NewGenerate().GenerateRequestId()
		path := c.Path()

		utils.Infof("[%s] 새로운 프록시 요청 시작: %s %s", requestId, c.Method(), path)

		// [2] 내부 경로 체크
		if pathUtil.IsInternalPath(path) {
			utils.Infof("[%s] 내부 관리 경로 감지 (%s), 직접 처리", requestId, path)
			return c.Next()
		}

		// [3] API 요청 검증
		if !strings.HasPrefix(path, "/api") {
			utils.Warnf("[%s] 비정상 요청 차단: %s %s from %s",
				requestId, c.Method(), path, c.IP())
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"message": "Forbidden",
			})
		}

		utils.Infof("[%s] 내부 경로 아님, 프록시 처리 시작", requestId)

		// [4] 최적의 서버 선택
		selectedServer := serverService.SelectOptimalServer()
		if selectedServer == nil {
			utils.Errorf("[%s] 선택된 서버가 없음", requestId)
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"success": false,
				"message": "사용 가능한 서버가 없습니다",
			})
		}
		utils.Infof("[%s] 선택된 서버: %s", requestId, selectedServer.ServerId)

		// [5] 선택된 서버로 요청 시도
		err := tryServer(c, selectedServer, requestId)

		// [6] 실패 시 서버리스로 장애 조치
		if err != nil {
			utils.Warnf("[%s] 기본 서버 실패, 서버리스로 장애 조치: %v", requestId, err)
			fallbackServer := serverService.GetServerlessServer()
			utils.Infof("[%s] 서버리스 서버 가져오기 완료: %+v", requestId, fallbackServer)

			// [7] 서버리스로 재시도
			err = tryServer(c, fallbackServer, requestId)
			if err != nil {
				utils.Errorf("[%s] 서버리스 장애 조치 실패: %v", requestId, err)
				return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
					"success": false,
					"message": "모든 서버 요청 실패",
				})
			}
		}

		// [8] 요청 완료 처리
		serverService.FinishUsingServer(selectedServer.ServerId)
		utils.Infof("[%s] 프록시 요청 완료", requestId)
		return nil
	}
}
