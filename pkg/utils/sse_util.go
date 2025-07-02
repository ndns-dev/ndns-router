package utils

import (
	"sync"
)

type SseManager struct {
	clients sync.Map // map[string]chan string
}

var Global = &SseManager{}

func (s *SseManager) Register(reqId string, ch chan string) {
	s.clients.Store(reqId, ch)
}

func (s *SseManager) Deregister(reqId string) {
	if ch, ok := s.clients.Load(reqId); ok {
		close(ch.(chan string))
		s.clients.Delete(reqId)
	}
}

func (s *SseManager) Send(reqId string, msg string) {
	if ch, ok := s.clients.Load(reqId); ok {
		ch.(chan string) <- msg
	}
}
