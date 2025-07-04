package utils

import (
	"sync"
)

type SseManager struct {
	clients sync.Map // map[string]chan string
}

var Global = &SseManager{}

func (s *SseManager) Register(reqId string, ch chan string) {
	if old, ok := s.clients.Load(reqId); ok {
		close(old.(chan string))
	}
	s.clients.Store(reqId, ch)
}

func (s *SseManager) Deregister(reqId string) {
	if ch, ok := s.clients.Load(reqId); ok {
		close(ch.(chan string))
		s.clients.Delete(reqId)
	}
}

func (s *SseManager) Send(reqId string, msg string) {
	if chRaw, ok := s.clients.Load(reqId); ok {
		ch := chRaw.(chan string)
		select {
		case ch <- msg:
			// 메시지 전송 성공
		default:
			// 수신 지연 또는 연결 없음
			s.Deregister(reqId)
		}
	}
}
