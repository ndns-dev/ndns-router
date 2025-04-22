package controllers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sh5080/ndns-router/pkg/interfaces"
	"github.com/sh5080/ndns-router/pkg/types"
	"github.com/sh5080/ndns-router/pkg/utils"
)

// MetricsController는 /api/metrics 경로의 요청을 처리하는 컨트롤러입니다
type MetricsController struct {
	serverService interfaces.ServerService
}

// NewMetricsController는 새로운 MetricsController를 생성합니다
func NewMetricsController(serverService interfaces.ServerService) *MetricsController {
	return &MetricsController{
		serverService: serverService,
	}
}

// MetricsUpdateRequest는 메트릭 업데이트 요청 구조체입니다
type MetricsUpdateRequest struct {
	AppName       string    `json:"app_name"`
	ServerURL     string    `json:"server_url"`
	CPUUsage      float64   `json:"cpu_usage"`
	MemoryUsage   float64   `json:"memory_usage"`
	ErrorRate     float64   `json:"error_rate"`
	ResponseTime  float64   `json:"response_time"`
	TotalRequests int64     `json:"total_requests"`
	ErrorRequests int64     `json:"error_requests"`
	Timestamp     time.Time `json:"timestamp"`
}

// HandleMetricsUpdate는 서버 메트릭 업데이트를 처리합니다
func (c *MetricsController) HandleMetricsUpdate(ctx *fiber.Ctx) error {
	var req MetricsUpdateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "잘못된 요청 형식입니다",
		})
	}

	// 수신된 메트릭 데이터 로깅
	utils.Infof("메트릭 수신 [%s]:", req.AppName)
	utils.Infof("  - 서버 URL: %s", req.ServerURL)
	utils.Infof("  - CPU 사용률: %.2f%%", req.CPUUsage)
	utils.Infof("  - 메모리 사용률: %.2f%%", req.MemoryUsage)
	utils.Infof("  - 에러율: %.2f%%", req.ErrorRate)
	utils.Infof("  - 응답시간: %.2fms", req.ResponseTime)
	utils.Infof("  - 총 요청수: %d", req.TotalRequests)
	utils.Infof("  - 에러 요청수: %d", req.ErrorRequests)
	utils.Infof("  - 타임스탬프: %s", req.Timestamp)

	// 서버가 없으면 자동으로 등록
	server, err := c.serverService.GetServer(req.AppName)
	if err != nil || server == nil {
		if err := c.serverService.AddServer(req.AppName, req.ServerURL); err != nil {
			utils.Errorf("서버 자동 등록 실패 (%s): %v", req.AppName, err)
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": "서버 등록 실패",
			})
		}
		utils.Infof("새 서버가 자동 등록됨: %s (%s)", req.AppName, req.ServerURL)
	}

	// 메트릭 데이터 업데이트
	metrics := &types.Metrics{
		CPUUsage:    req.CPUUsage,
		MemoryUsage: req.MemoryUsage,
		ErrorRate:   req.ErrorRate,
		Latency:     req.ResponseTime,
		Timestamp:   req.Timestamp,
	}

	if err := c.serverService.UpdateServerMetrics(req.AppName, metrics); err != nil {
		utils.Errorf("메트릭 업데이트 실패 (%s): %v", req.AppName, err)
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "메트릭 업데이트 실패",
		})
	}

	return ctx.JSON(fiber.Map{
		"success": true,
		"message": "메트릭이 성공적으로 업데이트되었습니다",
	})
}
