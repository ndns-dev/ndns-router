package controllers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sh5080/ndns-router/pkg/interfaces"
	"github.com/sh5080/ndns-router/pkg/types"
	"github.com/sh5080/ndns-router/pkg/utils"
)

// InternalController는 /api/internal 경로의 요청을 처리하는 컨트롤러입니다
type InternalController struct {
	serverService interfaces.ServerService
}

// NewInternalController는 새로운 InternalController를 생성합니다
func NewInternalController(serverService interfaces.ServerService) *InternalController {
	return &InternalController{
		serverService: serverService,
	}
}

// OptimalServerRequest는 최적 서버 등록 요청 구조체입니다
type OptimalServerRequest struct {
	Servers []struct {
		ServerId string `json:"serverId"`
		Metrics  struct {
			CpuUsage     float64 `json:"cpuUsage"`
			MemoryUsage  float64 `json:"memoryUsage"`
			ErrorRate    float64 `json:"errorRate"`
			ResponseTime float64 `json:"responseTime"`
			Score        float64 `json:"score"`
		} `json:"metrics"`
	} `json:"servers"`
}

// HandleOptimalServer는 최적 서버 등록 요청을 처리합니다
func (c *InternalController) HandleOptimalServer(ctx *fiber.Ctx) error {
	// 원본 요청 바디 로깅
	body := ctx.Body()
	utils.Infof("수신된 원본 요청 바디 (길이: %d): %s", len(body), string(body))

	var req OptimalServerRequest
	if err := ctx.BodyParser(&req); err != nil {
		utils.Errorf("요청 파싱 실패: %v", err)
		utils.Infof("기대하는 요청 형식: %s", `{
			"servers": [
				{
					"serverId": "서버ID",
					"metrics": {
						"cpuUsage": 0.0,
						"memoryUsage": 0.0,
						"errorRate": 0.0,
						"responseTime": 0.0,
						"score": 0.0
					}
				}
			]
		}`)
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success":  false,
			"message":  "잘못된 요청 형식입니다",
			"error":    err.Error(),
			"received": string(body),
		})
	}

	if len(req.Servers) == 0 {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "서버 목록이 비어있습니다",
		})
	}

	// 최적의 서버 선택
	var optimalServer struct {
		ServerId string
		Score    float64
	}
	optimalServer.Score = -1 // 초기값 설정

	// 수신된 서버 정보 로깅 및 최적 서버 선택
	utils.Infof("서버 정보 수신 (총 %d개):", len(req.Servers))
	for _, server := range req.Servers {
		utils.Infof("[%s] 정보:", server.ServerId)
		utils.Infof("  - CPU 사용률: %.2f%%", server.Metrics.CpuUsage)
		utils.Infof("  - 메모리 사용률: %.2f%%", server.Metrics.MemoryUsage)
		utils.Infof("  - 에러율: %.2f%%", server.Metrics.ErrorRate)
		utils.Infof("  - 응답시간: %.2fms", server.Metrics.ResponseTime)
		utils.Infof("  - 점수: %.2f", server.Metrics.Score)

		// 최적의 서버 선택 (가장 높은 점수)
		if server.Metrics.Score > optimalServer.Score {
			optimalServer.ServerId = server.ServerId
			optimalServer.Score = server.Metrics.Score
		}

		// 서버 정보 업데이트
		serverExists, err := c.serverService.GetServer(server.ServerId)
		if err != nil || serverExists == nil {
			if err := c.serverService.AddServer(server.ServerId, server.ServerId); err != nil {
				utils.Errorf("서버 자동 등록 실패 (%s): %v", server.ServerId, err)
				continue
			}
			utils.Infof("새 서버가 자동 등록됨: %s", server.ServerId)
		}

		// 메트릭 데이터 업데이트
		metrics := &types.Metrics{
			CPUUsage:    server.Metrics.CpuUsage,
			MemoryUsage: server.Metrics.MemoryUsage,
			ErrorRate:   server.Metrics.ErrorRate,
			Latency:     server.Metrics.ResponseTime,
			Timestamp:   time.Now(),
		}

		if err := c.serverService.UpdateServerMetrics(server.ServerId, metrics); err != nil {
			utils.Errorf("메트릭 업데이트 실패 (%s): %v", server.ServerId, err)
		}
	}

	utils.Infof("선택된 최적 서버: %s (점수: %.2f)", optimalServer.ServerId, optimalServer.Score)

	return ctx.JSON(fiber.Map{
		"success": true,
		"message": "서버 정보가 성공적으로 업데이트되었습니다",
		"data": fiber.Map{
			"optimal_server": optimalServer.ServerId,
			"score":          optimalServer.Score,
		},
	})
}
