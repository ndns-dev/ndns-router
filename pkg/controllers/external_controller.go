package controllers

import (
	"bufio"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sh5080/ndns-router/pkg/interfaces"
	"github.com/sh5080/ndns-router/pkg/types/dtos"
	"github.com/sh5080/ndns-router/pkg/utils"
	"github.com/valyala/fasthttp"
)

type ExternalController struct {
	serverService interfaces.ServerService
	sseManager    *utils.SseManager
}

func NewExternalController(serverService interfaces.ServerService) *ExternalController {
	return &ExternalController{
		serverService: serverService,
		sseManager:    utils.Global,
	}
}

func (c *ExternalController) GetActiveConnections(ctx *fiber.Ctx) error {
	activeConnections := c.sseManager.GetActiveConnections()
	utils.Infof("[SSE] 활성 연결 조회 - 총 %d개의 연결", len(activeConnections))

	return utils.SendSuccessData(ctx, dtos.ActiveConnections{
		TotalConnections: len(activeConnections),
		Connections:      activeConnections,
	})
}

func (c *ExternalController) SseHandler(ctx *fiber.Ctx) error {
	reqId := ctx.Query("reqId")
	if reqId == "" {
		return utils.SendError(ctx, fiber.StatusBadRequest, "reqId 파라미터가 필요합니다")
	}

	messageChan := make(chan string, 10)

	ctx.Set("Content-Type", "text/event-stream")
	ctx.Set("Cache-Control", "no-cache")
	ctx.Set("Connection", "keep-alive")

	utils.Infof("[SSE] 새로운 연결 시작: %s", reqId)

	c.sseManager.Register(reqId, messageChan)

	ctx.Context().SetConnectionClose()

	ctx.Context().Response.SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		defer func() {
			utils.Infof("[SSE] 스트림 종료, 채널 정리 시작 - reqId: %s", reqId)
			c.sseManager.Deregister(reqId)
			utils.Infof("[SSE] 스트림 종료, 채널 정리 완료 - reqId: %s", reqId)
		}()

		ssePayload := dtos.SsePayload{
			Type: dtos.SseConnect,
			Data: map[string]interface{}{
				"message": "SSE 연결됨",
			},
		}

		if err := utils.SendSseEvent(w, &ssePayload); err != nil {
			utils.Infof("[SSE] 초기 메시지 전송 실패 - reqId: %s, error: %v", reqId, err)
			return
		}

		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case msg, ok := <-messageChan:
				if !ok {
					utils.Infof("[SSE] 메시지 채널이 닫힘 - reqId: %s", reqId)
					return
				}
				utils.Infof("[SSE] 클라이언트에게 결과 전송: %s", msg)
				ssePayload := dtos.SsePayload{
					Type: dtos.SseMessage,
					Data: map[string]interface{}{
						"result": msg,
					},
				}
				if err := utils.SendSseEvent(w, &ssePayload); err != nil {
					utils.Infof("[SSE] 클라이언트 연결 종료 (reqId: %s): %v", reqId, err)
					return
				}
			case <-ticker.C:
				utils.Infof("[SSE] 하트비트 전송 시도 - reqId: %s", reqId)
				ssePayload := dtos.SsePayload{
					Type: dtos.SseHeartbeat,
					Data: map[string]interface{}{
						"heartbeat": time.Now().Format(time.RFC3339),
					},
				}
				if err := utils.SendSseEvent(w, &ssePayload); err != nil {
					utils.Infof("[SSE] 하트비트 전송 실패 (reqId: %s): %v", reqId, err)
					return
				}
			}
		}
	}))

	return nil
}

func (c *ExternalController) SendMessage(ctx *fiber.Ctx) error {
	req := new(dtos.MessageRequest)
	if err := ctx.BodyParser(req); err != nil {
		return utils.SendError(ctx, fiber.StatusBadRequest, "잘못된 요청 형식")
	}

	if req.ReqId == "" || req.Message == "" {
		return utils.SendError(ctx, fiber.StatusBadRequest, "reqId와 message는 필수 값입니다")
	}

	utils.Infof("[SSE] Send 시도 - reqId: %s", req.ReqId)
	c.sseManager.Send(req.ReqId, req.Message)

	return utils.SendSuccessMessage(ctx, "메시지가 성공적으로 전송되었습니다")
}
