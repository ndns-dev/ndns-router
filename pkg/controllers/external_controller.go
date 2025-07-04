package controllers

import (
	"bufio"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/sh5080/ndns-router/pkg/interfaces"
	"github.com/sh5080/ndns-router/pkg/utils"
)

type ExternalController struct {
	serverService interfaces.ServerService
}

// NewExternalController는 새로운 ExternalController를 생성합니다
func NewExternalController(serverService interfaces.ServerService) *ExternalController {
	return &ExternalController{
		serverService: serverService,
	}
}

// SseHandler는 외부 서버에서 데이터를 수신하는 핸들러입니다.
func (c *ExternalController) SseHandler(ctx *fiber.Ctx) error {
	reqId := ctx.Locals("reqId").(string)

	msgChan := make(chan string)
	utils.Global.Register(reqId, msgChan)
	defer utils.Global.Deregister(reqId)

	// Header 설정
	ctx.Set("Content-Type", "text/event-stream")
	ctx.Set("Cache-Control", "no-cache")
	ctx.Set("Connection", "keep-alive")

	// 스트림 작성
	ctx.Context().Response.SetBodyStreamWriter(func(w *bufio.Writer) {
		// 초기 연결 확인 메시지 전송
		utils.Infof("[SSE] 새로운 연결 시작: %s", reqId)
		fmt.Fprintf(w, "data: {\"type\":\"connected\",\"message\":\"SSE 연결됨\"}\n\n")
		w.Flush()

		for msg := range msgChan {
			utils.Infof("[SSE] 클라이언트에게 결과 전송: %s", msg)
			fmt.Fprintf(w, "data: {\"type\":\"analysisComplete\",\"post\":\"%s\"}\n\n", msg)
			w.Flush()
		}
	})

	return nil
}
