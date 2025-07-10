package utils

import (
	"bufio"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/sh5080/ndns-router/pkg/types"
	"github.com/sh5080/ndns-router/pkg/types/dtos"
)

type SseManager struct {
	clients     sync.Map
	validPeriod time.Duration // 연결 유효 기간
}

var Global = &SseManager{
	validPeriod: 5 * time.Minute, // 5분으로 설정
}

func (s *SseManager) Register(reqId string, ch chan string) {
	Infof("[SSE] Register 시도 - reqId: %s", reqId)

	now := time.Now()
	if old, ok := s.clients.Load(reqId); ok {
		oldInfo := old.(*types.ConnectionInfo)

		// 만료되지 않은 채널이 있다면 재활용
		if now.Before(oldInfo.ExpiresAt) {
			Infof("[SSE] 유효한 기존 채널 발견, 기간 연장 - reqId: %s", reqId)

			// 기존 채널 유지하고 유효기간만 연장
			oldInfo.ExpiresAt = now.Add(s.validPeriod)
			oldInfo.LastActive = now
			return
		}

		// 만료된 채널은 정리
		Infof("[SSE] 만료된 채널 발견, 정리 중 - reqId: %s", reqId)
		close(oldInfo.Channel)
		close(oldInfo.Done)
	}

	// 새 연결 등록
	s.clients.Store(reqId, &types.ConnectionInfo{
		Channel:     ch,
		ConnectedAt: now,
		LastActive:  now,
		Done:        make(chan struct{}),
		ExpiresAt:   now.Add(s.validPeriod),
	})
}

func (s *SseManager) GetActiveConnections() []types.Connection {
	connections := make([]types.Connection, 0)
	seen := make(map[string]bool)
	now := time.Now()

	s.clients.Range(func(key, value interface{}) bool {
		reqId := key.(string)
		if seen[reqId] {
			Warnf("[SSE] 중복된 reqId 발견: %s", reqId)
			return true
		}

		info := value.(*types.ConnectionInfo)
		// 만료까지 남은 시간 계산
		expiresIn := info.ExpiresAt.Sub(now)

		connections = append(connections, types.Connection{
			ReqId:             reqId,
			ConnectedAt:       info.ConnectedAt,
			ConnectedDuration: now.Sub(info.ConnectedAt),
			ExpiresIn:         expiresIn,
		})
		seen[reqId] = true
		return true
	})

	return connections
}

func (s *SseManager) Send(reqId string, msg string) {
	Infof("[SSE] Send 시도 - reqId: %s", reqId)

	if value, ok := s.clients.Load(reqId); ok {
		info := value.(*types.ConnectionInfo)
		Infof("[SSE] 채널 찾음 - reqId: %s", reqId)

		select {
		case info.Channel <- msg:
			info.LastActive = time.Now() // 메시지 전송 성공 시 LastActive 업데이트
			Infof("[SSE] 메시지 전송 성공 - reqId: %s, message: %s", reqId, msg)
		default:
			Warnf("[SSE] 메시지 전송 실패 (채널 막힘) - reqId: %s", reqId)
			s.Deregister(reqId)
		}
	} else {
		Warnf("[SSE] 채널을 찾을 수 없음 - reqId: %s", reqId)
	}
}

func (s *SseManager) Deregister(reqId string) {
	Infof("[SSE] Deregister 시도 - reqId: %s", reqId)

	if value, ok := s.clients.Load(reqId); ok {
		info := value.(*types.ConnectionInfo)
		close(info.Done)
		close(info.Channel)
		s.clients.Delete(reqId)
		Infof("[SSE] 채널 종료 및 삭제 완료 - reqId: %s", reqId)
	}
}

func SendSseEvent(w *bufio.Writer, payload *dtos.SsePayload) error {
	data, err := json.Marshal(payload.Data)
	if err != nil {
		return fmt.Errorf("JSON 직렬화 실패: %v", err)
	}

	_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", payload.Type, string(data))
	if err != nil {
		return fmt.Errorf("SSE 메시지 전송 실패: %v", err)
	}

	return w.Flush()
}
