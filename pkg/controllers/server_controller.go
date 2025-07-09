package controllers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sh5080/ndns-router/pkg/interfaces"
	"github.com/sh5080/ndns-router/pkg/types"
	"github.com/sh5080/ndns-router/pkg/utils"
)

// ServerController는 /api/servers 경로의 요청을 처리하는 컨트롤러입니다
type ServerController struct {
	serverService interfaces.ServerService
}

// NewServerController는 새로운 ServerController를 생성합니다
func NewServerController(serverService interfaces.ServerService) *ServerController {
	return &ServerController{
		serverService: serverService,
	}
}

// HandleServersStatus는 등록된 서버 목록과 상태를 반환합니다
func (c *ServerController) HandleServersStatus(ctx *fiber.Ctx) error {
	// 서버 목록 조회
	servers, err := c.serverService.GetAllServers()
	if err != nil {
		utils.Errorf("서버 목록 조회 실패: %v", err)
		return utils.SendError(ctx, fiber.StatusInternalServerError, "서버 목록 조회 실패")
	}

	// 서버 상태 정보 구성
	serverInfos := make([]fiber.Map, 0, len(servers))
	for _, server := range servers {
		serverInfo := fiber.Map{
			"serverId":    server.ServerId,
			"serverUrl":   server.ServerUrl,
			"serverType":  server.ServerType,
			"status":      server.CurrentStatus,
			"lastUpdated": server.LastUpdated.Format(time.RFC3339),
		}

		if server.Metrics != nil {
			serverInfo["metrics"] = server.Metrics
		}

		serverInfos = append(serverInfos, serverInfo)
	}

	return ctx.JSON(fiber.Map{
		"success": true,
		"message": "서버 상태 목록",
		"data":    serverInfos,
	})
}

// HandleAddServer는 새로운 서버를 등록합니다
func (c *ServerController) HandleAddServer(ctx *fiber.Ctx) error {
	var req struct {
		ServerId   string `json:"serverId"`
		URL        string `json:"url"`
		ServerType string `json:"serverType"`
	}

	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "잘못된 요청 형식",
		})
	}

	if req.ServerId == "" || req.URL == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "serverId와 url은 필수 값입니다",
		})
	}

	if err := c.serverService.AddServer(&types.Server{
		ServerId:      req.ServerId,
		ServerUrl:     req.URL,
		ServerType:    req.ServerType,
		CurrentStatus: string(types.StatusUnknown),
		LastUpdated:   time.Now(),
	}); err != nil {
		return utils.SendError(ctx, fiber.StatusInternalServerError, "서버 등록 실패: "+err.Error())
	}

	return utils.SendSuccessMessage(ctx, "서버가 등록되었습니다")
}

// HandleRemoveServer는 등록된 서버를 제거합니다
func (c *ServerController) HandleRemoveServer(ctx *fiber.Ctx) error {
	serverId := ctx.Query("serverId")
	if serverId == "" {
		return utils.SendError(ctx, fiber.StatusBadRequest, "serverId는 필수 값입니다")
	}

	if err := c.serverService.RemoveServer(serverId); err != nil {
		return utils.SendError(ctx, fiber.StatusInternalServerError, "서버 제거 실패: "+err.Error())
	}

	return utils.SendSuccessMessage(ctx, "서버가 제거되었습니다")
}
