package dtos

import "github.com/sh5080/ndns-router/pkg/types"

type MessageRequest struct {
	ReqId   string `json:"reqId"`
	Message string `json:"message"`
}

type ActiveConnections struct {
	TotalConnections int                `json:"totalConnections"`
	Connections      []types.Connection `json:"connections"`
}

type SsePayload struct {
	Type SseMessageType `json:"type"`
	Data interface{}    `json:"data,omitempty"`
}

type SseMessageType string

const (
	SseConnect   SseMessageType = "connect"
	SseMessage   SseMessageType = "message"
	SseHeartbeat SseMessageType = "heartbeat"
)
