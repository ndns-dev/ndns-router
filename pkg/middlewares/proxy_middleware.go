package middlewares

import (
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/proxy"
	"github.com/sh5080/ndns-router/pkg/configs"
	"github.com/sh5080/ndns-router/pkg/interfaces"
	"github.com/sh5080/ndns-router/pkg/types"
	"github.com/sh5080/ndns-router/pkg/utils"
	"github.com/valyala/fasthttp"
)

// selectProxyServer는 요청 Limit 값과 서버 상태에 따라 프록시할 최적의 서버를 선택합니다.
// 적합한 서버를 찾으면 해당 서버 객체를 반환하고, 그렇지 않으면 nil을 반환합니다.
func selectProxyServer(c *fiber.Ctx, serverService interfaces.ServerService, requestId string) *types.Server {
	serverGroup := serverService.GetServerGroup()
	limit := c.QueryInt("limit", 0)

	// limit=2일 때는 서버리스 강제사용 건너뛰기
	if limit != 2 && serverGroup.ForceServerless {
		utils.Infof("[%s] 서버리스 강제 사용", requestId)
		return serverGroup.ServerlessServer
	}

	// limit 값에 따른 우선순위 정의
	var performanceOrder []string
	// limit ~2일 때는 서버리스 사용하지 않음
	if limit <= 2 {
		performanceOrder = []string{
			"ndns-api1",     // EC2
			"ndns-api2",     // Windows Desktop
			"ndns-external", // Mac M1
		}
		utils.Infof("[%s] Limit=2 우선순위 적용", requestId)
	} else if limit <= 10 {
		performanceOrder = []string{
			"ndns-external", // Mac M1
			"ndns-api1",     // EC2
			"ndns-api2",     // Windows Desktop
			"ndns-api3",     // Cloud Run
		}
		utils.Infof("[%s] Limit=10 우선순위 적용", requestId)
	} else {
		// 기본 우선순위
		performanceOrder = []string{
			"ndns-external", // Mac M1
			"ndns-api1",     // EC2
			"ndns-api3",     // Cloud Run
			"ndns-api2",     // Windows Desktop
		}
		utils.Infof("[%s] 기본 우선순위 적용", requestId)
	}

	// Excellent 서버가 있으면 Excellent 서버들 중에서만 선택
	if len(serverGroup.ExcellentServers) > 0 {
		for _, preferredId := range performanceOrder {
			for _, server := range serverGroup.ExcellentServers {
				if server.ServerId == preferredId {
					utils.Infof("[%s] Excellent 서버 중 성능 우선순위 선택: %s (점수: %.2f, limit: %d)",
						requestId, server.ServerId, server.Metrics.Score, limit)
					return server
				}
			}
		}
		return serverGroup.ExcellentServers[0]
	}

	// Good 서버들 중에서 선택
	if len(serverGroup.GoodServers) > 0 {
		for _, preferredId := range performanceOrder {
			for _, server := range serverGroup.GoodServers {
				if server.ServerId == preferredId {
					utils.Infof("[%s] Good 서버 중 성능 우선순위 선택: %s (점수: %.2f, limit: %d)",
						requestId, server.ServerId, server.Metrics.Score, limit)
					return server
				}
			}
		}
		return serverGroup.GoodServers[0]
	}

	return nil
}

func NewProxyMiddleware(serverService interfaces.ServerService) fiber.Handler {
	pathUtil := utils.NewPath(configs.InternalPaths)

	// 서버 요청 시도 (tryServer 함수는 그대로 유지)
	tryServer := func(c *fiber.Ctx, server *types.Server, requestId string) error {
		if server == nil {
			utils.Infof("[%s] 서버가 없어 서버리스로 전환", requestId)
			server = serverService.GetServerlessServer() // 폴백 서버 (서버리스)
		}

		// nil이 여전히 발생할 수 있는 시나리오 방지
		if server == nil {
			utils.Errorf("[%s] 프록시할 서버(서버리스 포함)를 찾을 수 없습니다.", requestId)
			return fmt.Errorf("no proxy server available")
		}

		utils.Infof("[%s] 서버 시도: %s (점수: %.2f)", requestId, server.ServerId, server.Metrics.Score)

		// [1] URL 정규화
		targetURL := server.ServerUrl
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

		// [4] TLS 검증 건너뛰기 설정 및 프록시 요청 실행
		err := proxy.Do(c, fullURL, &fasthttp.Client{
			TLSConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		})
		if err != nil {
			utils.Warnf("[%s] 서버 요청 실패, 서버 제거: %s (%v)", requestId, server.ServerId, err)
			serverService.RemoveServer(server.ServerId)
			return err
		}

		// [6] 응답 헤더에 서버 정보 추가
		c.Response().Header.Set("X-Served-By", server.ServerId)
		c.Response().Header.Set("X-Server-Score", fmt.Sprintf("%.2f", server.Metrics.Score))

		return nil
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

		// 단일 서버 선택 및 요청 시도
		selectedServer := selectProxyServer(c, serverService, requestId)
		err := tryServer(c, selectedServer, requestId)
		if err != nil {
			utils.Infof("[%s] 서버리스로 전환", requestId)
			return tryServer(c, nil, requestId)
		}

		return nil
	}
}
