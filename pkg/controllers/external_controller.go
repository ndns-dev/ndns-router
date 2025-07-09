package controllers

import (
	"bufio"
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sh5080/ndns-router/pkg/interfaces"
	"github.com/sh5080/ndns-router/pkg/types"
	"github.com/sh5080/ndns-router/pkg/utils"
	"github.com/valyala/fasthttp"
)

type ExternalController struct {
	serverService interfaces.ServerService
	clients       map[string]*types.ConnectionInfo
	mu            sync.RWMutex
}

// NewExternalController는 새로운 ExternalController를 생성합니다
func NewExternalController(serverService interfaces.ServerService) *ExternalController {
	return &ExternalController{
		serverService: serverService,
		clients:       make(map[string]*types.ConnectionInfo),
	}
}

// GetActiveConnections는 현재 활성화된 SSE 연결들의 정보를 반환합니다
func (c *ExternalController) GetActiveConnections(ctx *fiber.Ctx) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 활성 연결 정보 수집
	activeConnections := make([]types.Connection, 0)
	for reqId, info := range c.clients {
		connectionInfo := types.Connection{
			ReqId:             reqId,
			ConnectedAt:       info.ConnectedAt,
			ConnectedDuration: time.Since(info.ConnectedAt),
		}
		activeConnections = append(activeConnections, connectionInfo)
	}

	utils.Infof("[SSE] 활성 연결 조회 - 총 %d개의 연결", len(activeConnections))

	return utils.SendSuccessData(ctx, fiber.Map{
		"totalConnections": len(activeConnections),
		"connections":      activeConnections,
	})
}

// RegisterMessageChannel은 특정 reqId에 대한 메시지 채널을 등록합니다
func (c *ExternalController) RegisterMessageChannel(reqId string, msgChan chan string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 기존 채널이 있다면 닫기
	if info, exists := c.clients[reqId]; exists {
		close(info.Channel)
		delete(c.clients, reqId)
	}

	c.clients[reqId] = &types.ConnectionInfo{
		Channel:     msgChan,
		ConnectedAt: time.Now(),
	}
	utils.Infof("[SSE] 메시지 채널 등록 완료 - reqId: %s", reqId)
}

// UnregisterMessageChannel은 특정 reqId의 메시지 채널을 제거합니다
func (c *ExternalController) UnregisterMessageChannel(reqId string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if info, exists := c.clients[reqId]; exists {
		close(info.Channel)
		delete(c.clients, reqId)
		utils.Infof("[SSE] 메시지 채널 제거 완료 - reqId: %s", reqId)
	}
}

// SseHandler는 외부 서버에서 데이터를 수신하는 핸들러입니다
func (c *ExternalController) SseHandler(ctx *fiber.Ctx) error {
	// reqId 파라미터 확인
	reqId := ctx.Query("reqId")
	if reqId == "" {
		return utils.SendError(ctx, fiber.StatusBadRequest, "reqId 파라미터가 필요합니다")
	}

	messageChan := make(chan string, 10)

	// SSE 헤더 설정
	ctx.Set("Content-Type", "text/event-stream")
	ctx.Set("Cache-Control", "no-cache")
	ctx.Set("Connection", "keep-alive")

	utils.Infof("[SSE] 새로운 연결 시작: %s", reqId)

	// 메시지 채널 등록
	c.RegisterMessageChannel(reqId, messageChan)

	// 연결이 끊어졌을 때 정리
	ctx.Context().SetConnectionClose()

	// 스트림 작성기 설정
	ctx.Context().Response.SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		defer func() {
			utils.Infof("[SSE] 스트림 종료, 채널 정리 시작 - reqId: %s", reqId)
			c.UnregisterMessageChannel(reqId)
			utils.Infof("[SSE] 스트림 종료, 채널 정리 완료 - reqId: %s", reqId)
		}()

		// 초기 연결 메시지 전송
		fmt.Fprintf(w, "event: connect\ndata: {\"type\":\"connected\",\"message\":\"SSE 연결됨\",\"reqId\":\"%s\"}\n\n", reqId)
		if err := w.Flush(); err != nil {
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
				fmt.Fprintf(w, "event: message\ndata: {\"type\":\"analysisComplete\",\"post\":%s,\"reqId\":\"%s\"}\n\n", msg, reqId)
				if err := w.Flush(); err != nil {
					utils.Infof("[SSE] 클라이언트 연결 종료 (reqId: %s): %v", reqId, err)
					return
				}
			case <-ticker.C:
				fmt.Fprintf(w, "event: heartbeat\ndata: {\"type\":\"heartbeat\",\"time\":\"%v\",\"reqId\":\"%s\"}\n\n", time.Now().Format(time.RFC3339), reqId)
				if err := w.Flush(); err != nil {
					utils.Infof("[SSE] 하트비트 전송 실패 (reqId: %s): %v", reqId, err)
					return
				}
			}
		}
	}))

	return nil
}

// SendMessage는 특정 클라이언트에게 메시지를 전송하는 핸들러입니다
func (c *ExternalController) SendMessage(ctx *fiber.Ctx) error {
	// POST body에서 데이터 파싱
	type MessageRequest struct {
		ReqId   string `json:"reqId"`
		Message string `json:"message"`
	}

	req := new(MessageRequest)
	if err := ctx.BodyParser(req); err != nil {
		return utils.SendError(ctx, fiber.StatusBadRequest, "잘못된 요청 형식")
	}

	if req.ReqId == "" || req.Message == "" {
		return utils.SendError(ctx, fiber.StatusBadRequest, "reqId와 message는 필수 값입니다")
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	utils.Infof("[SSE] Send 시도 - reqId: %s", req.ReqId)
	if info, exists := c.clients[req.ReqId]; exists {
		select {
		case info.Channel <- req.Message:
			utils.Infof("[SSE] 메시지 전송 성공 (reqId: %s)", req.ReqId)
			return utils.SendSuccessData(ctx, fiber.Map{
				"reqId":             req.ReqId,
				"connectedAt":       info.ConnectedAt.Format(time.RFC3339),
				"connectedDuration": time.Since(info.ConnectedAt).String(),
			})
		default:
			utils.Infof("[SSE] 메시지 전송 실패 (reqId: %s) - 채널이 가득 참", req.ReqId)
			return utils.SendError(ctx, fiber.StatusServiceUnavailable, "메시지 전송 실패 - 채널이 가득 참")
		}
	} else {
		utils.Infof("[WARN] [SSE] 채널을 찾을 수 없음 - reqId: %s", req.ReqId)
		return utils.SendError(ctx, fiber.StatusNotFound, "존재하지 않는 reqId입니다")
	}
}
