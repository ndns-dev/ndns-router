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
		if server == nil {
			return fiber.NewError(fiber.StatusServiceUnavailable, "서버가 없음")
		}

		utils.Infof("[%s] 서버 시도: %s (점수: %.2f)", requestId, server.ServerId, server.Metrics.Score)

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
		err := proxy.DoRedirects(c, fullURL, configs.MaxRetryAttempts)
		if err != nil {
			utils.Warnf("[%s] 서버 요청 실패, 서버 제거: %s (%v)", requestId, server.ServerId, err)
			serverService.RemoveServer(server.ServerId)
			return err
		}
		return nil
	}

	// 최적의 서버 선택
	selectBestServer := func(servers []*types.Server) *types.Server {
		if len(servers) == 0 {
			return nil
		}

		// 점수가 가장 높은 서버 선택
		bestServer := servers[0]
		for _, server := range servers[1:] {
			if server.Metrics.Score > bestServer.Metrics.Score {
				bestServer = server
			}
		}
		return bestServer
	}

	// 서버 목록에서 다음 최적 서버 선택
	selectNextBestServer := func(servers []*types.Server, excludeServerId string) *types.Server {
		if len(servers) == 0 {
			return nil
		}

		var bestServer *types.Server
		bestScore := float64(-1)

		for _, server := range servers {
			if server.ServerId != excludeServerId && server.Metrics.Score > bestScore {
				bestScore = server.Metrics.Score
				bestServer = server
			}
		}
		return bestServer
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

		// [4] 서버 그룹 가져오기
		serverGroup := serverService.SelectOptimalServers()

		// [5] 서버리스 강제 사용 체크
		if serverGroup.ForceServerless {
			utils.Infof("[%s] 서버리스 강제 사용", requestId)
			return tryServer(c, serverGroup.ServerlessServer, requestId)
		}

		// [6] 최상위 서버 시도
		if len(serverGroup.ExcellentServers) > 0 {
			// 첫 번째 최상위 서버 시도
			bestServer := selectBestServer(serverGroup.ExcellentServers)
			err := tryServer(c, bestServer, requestId)
			if err == nil {
				return nil
			}

			// 다른 최상위 서버 시도
			nextServer := selectNextBestServer(serverGroup.ExcellentServers, bestServer.ServerId)
			if nextServer != nil {
				err = tryServer(c, nextServer, requestId)
				if err == nil {
					return nil
				}
			}
		}

		// [7] 양호 서버 시도
		if len(serverGroup.GoodServers) > 0 {
			// 첫 번째 양호 서버 시도
			bestServer := selectBestServer(serverGroup.GoodServers)
			err := tryServer(c, bestServer, requestId)
			if err == nil {
				return nil
			}

			// 다른 양호 서버 시도
			nextServer := selectNextBestServer(serverGroup.GoodServers, bestServer.ServerId)
			if nextServer != nil {
				err = tryServer(c, nextServer, requestId)
				if err == nil {
					return nil
				}
			}
		}

		// [8] 모든 서버 실패 시 서버리스로 최종 시도
		utils.Infof("[%s] 모든 서버 실패, 서버리스로 최종 시도", requestId)
		err := tryServer(c, serverGroup.ServerlessServer, requestId)
		if err != nil {
			utils.Errorf("[%s] 서버리스 최종 시도 실패: %v", requestId, err)
			return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
				"success": false,
				"message": "모든 서버 요청 실패",
			})
		}

		return nil
	}
}
