package utils

import (
	"sync"
)

type SseManager struct {
	clients sync.Map // map[string]chan string
}

var Global = &SseManager{}

func (s *SseManager) Register(reqId string, ch chan string) {
	Infof("[SSE] Register 시도 - reqId: %s", reqId)

	if old, ok := s.clients.Load(reqId); ok {
		Infof("[SSE] 기존 채널 발견, 종료 중 - reqId: %s", reqId)
		close(old.(chan string))
	}

	s.clients.Store(reqId, ch)
	Infof("[SSE] 새 채널 등록 완료 - reqId: %s", reqId)
}

func (s *SseManager) Deregister(reqId string) {
	Infof("[SSE] Deregister 시도 - reqId: %s", reqId)

	if ch, ok := s.clients.Load(reqId); ok {
		Infof("[SSE] 채널 종료 및 삭제 - reqId: %s", reqId)
		close(ch.(chan string))
		s.clients.Delete(reqId)
	}
}

func (s *SseManager) Send(reqId string, msg string) {
	Infof("[SSE] Send 시도 - reqId: %s", reqId)

	if chRaw, ok := s.clients.Load(reqId); ok {
		Infof("[SSE] 채널 찾음 - reqId: %s", reqId)
		ch := chRaw.(chan string)

		select {
		case ch <- msg:
			Infof("[SSE] 메시지 전송 성공 - reqId: %s, message: %s", reqId, msg)
		default:
			Warnf("[SSE] 메시지 전송 실패 (채널 막힘) - reqId: %s", reqId)
			s.Deregister(reqId)
		}
	} else {
		Warnf("[SSE] 채널을 찾을 수 없음 - reqId: %s", reqId)
	}
}
