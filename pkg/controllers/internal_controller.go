package controllers

import (
	"encoding/json"
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
	var request types.OptimalServerRequest
	if err := ctx.BodyParser(&request); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "잘못된 요청 형식입니다",
			"error":   err.Error(),
		})
	}

	utils.Infof("서버 정보 수신 (총 %d개):", len(request.Servers))

	for _, serverInfo := range request.Servers {
		server := &types.Server{
			ServerId:      serverInfo.ServerId,
			ServerUrl:     serverInfo.ServerUrl,
			ServerType:    serverInfo.ServerType,
			CurrentStatus: string(types.StatusExcellent),
			LastUpdated:   time.Now(),
			Metrics: &types.Metrics{
				CPUUsage:    serverInfo.Metrics.CpuUsage,
				MemoryUsage: serverInfo.Metrics.MemoryUsage,
				ErrorRate:   serverInfo.Metrics.ErrorRate,
				Latency:     serverInfo.Metrics.ResponseTime,
				Score:       serverInfo.Metrics.Score,
				Timestamp:   time.Now(),
			},
		}

		utils.Infof("[%s] 정보:", server.ServerId)
		utils.Infof("  - CPU 사용률: %.2f%%", server.Metrics.CPUUsage)
		utils.Infof("  - 메모리 사용률: %.2f%%", server.Metrics.MemoryUsage)
		utils.Infof("  - 에러율: %.2f%%", server.Metrics.ErrorRate)
		utils.Infof("  - 응답시간: %.2fms", server.Metrics.Latency)
		utils.Infof("  - 점수: %.2f", server.Metrics.Score)

		// 서버 정보 저장 (없으면 추가, 있으면 업데이트)
		if err := c.serverService.AddServer(server); err != nil {
			utils.Warnf("서버 정보 저장 실패: %v", err)
			continue
		}
	}

	return ctx.SendStatus(fiber.StatusOK)
}

func (c *InternalController) HandleAnalysis(ctx *fiber.Ctx) error {
	var result types.AnalysisResult
	if err := ctx.BodyParser(&result); err != nil {
		return ctx.Status(fiber.StatusBadRequest).SendString("Invalid payload")
	}
	utils.Infof("\n=== AnalyzeCycle 분석결과 api서버에서 수신 ===\n%+v\n", result)

	jsonMsg, _ := json.Marshal(result)
	utils.Global.Send(result.ReqId, string(jsonMsg))
	return ctx.SendStatus(fiber.StatusOK)
}
