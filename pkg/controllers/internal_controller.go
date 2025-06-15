package controllers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sh5080/ndns-router/pkg/interfaces"
	"github.com/sh5080/ndns-router/pkg/types"
	"github.com/sh5080/ndns-router/pkg/utils"
)

type InternalController struct {
	serverService interfaces.ServerService
}

// NewInternalController는 새로운 InternalController를 생성합니다
func NewInternalController(serverService interfaces.ServerService) *InternalController {
	return &InternalController{
		serverService: serverService,
	}
}

// HandleOptimalServer는 최적 서버 등록 요청을 처리합니다
func (c *InternalController) HandleOptimalServer(ctx *fiber.Ctx) error {
	// 원본 요청 바디 로깅
	body := ctx.Body()
	utils.Infof("수신된 원본 요청 바디 (길이: %d): %s", len(body), string(body))

	var req types.OptimalServerRequest
	if err := ctx.BodyParser(&req); err != nil {
		utils.Errorf("요청 파싱 실패: %v", err)

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

	// 수신된 서버 정보 로깅 및 최적 서버 선택
	utils.Infof("서버 정보 수신 (총 %d개):", len(req.Servers))
	for _, server := range req.Servers {
		utils.Infof("[%s] 상세 정보:", server.ServerId)
		utils.Infof("  - URL: %s", server.ServerUrl)
		utils.Infof("  - CPU 사용률: %.2f%%", server.Metrics.CpuUsage)
		utils.Infof("  - 메모리 사용률: %.2f%%", server.Metrics.MemoryUsage)
		utils.Infof("  - 에러율: %.2f%%", server.Metrics.ErrorRate)
		utils.Infof("  - 응답시간: %.2fms", server.Metrics.ResponseTime)
		utils.Infof("  - 점수: %.2f", server.Metrics.Score)

		// 서버 정보 업데이트
		serverExists, err := c.serverService.GetServer(server.ServerId)

		if err != nil || serverExists == nil {
			// 새 서버 등록 - request에서 받은 정보로 등록
			if err := c.serverService.AddServer(server.ServerId, server.ServerUrl); err != nil {
				utils.Errorf("서버 자동 등록 실패 (%s): %v", server.ServerId, err)
				continue
			}
			utils.Infof("새 서버가 자동 등록됨: %s (URL: %s)", server.ServerId, server.ServerUrl)
		}

		// 서버 정보 업데이트 (메트릭스와 URL)
		metrics := &types.Metrics{
			Score:       server.Metrics.Score,
			CPUUsage:    server.Metrics.CpuUsage,
			MemoryUsage: server.Metrics.MemoryUsage,
			ErrorRate:   server.Metrics.ErrorRate,
			Latency:     server.Metrics.ResponseTime,
			Timestamp:   time.Now(),
		}

		if err := c.serverService.UpdateServerInfo(server.ServerId, server.ServerUrl, metrics); err != nil {
			utils.Errorf("서버 정보 업데이트 실패: %v", err)
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": "서버 정보 업데이트 실패",
				"error":   err.Error(),
			})
		}
	}

	return ctx.JSON(fiber.Map{
		"success": true,
		"message": "서버 정보가 성공적으로 업데이트되었습니다",
	})
}
