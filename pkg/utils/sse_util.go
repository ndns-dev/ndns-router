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

type ConnectionInfo struct {
	Channel     chan string
	ConnectedAt time.Time
	LastActive  time.Time
	Done        chan struct{}
}

type SseManager struct {
	clients     sync.Map
	validPeriod time.Duration // 연결 유효 기간
}

var Global = &SseManager{
	validPeriod: 30 * time.Minute, // 30분으로 설정
}

func (s *SseManager) Register(reqId string, ch chan string) {
	Infof("[SSE] Register 시도 - reqId: %s", reqId)

	// 기존 연결 정리
	if old, ok := s.clients.Load(reqId); ok {
		Infof("[SSE] 기존 채널 발견, 종료 중 - reqId: %s", reqId)
		oldInfo := old.(*ConnectionInfo)
		close(oldInfo.Channel)
		s.clients.Delete(reqId)
	}

	// 새 연결 등록
	s.clients.Store(reqId, &ConnectionInfo{
		Channel:     ch,
		ConnectedAt: time.Now(),
		LastActive:  time.Now(),
		Done:        make(chan struct{}),
	})

	// 등록 후 확인을 위한 모든 활성 연결 출력
	s.logActiveConnections("[Register 후 활성 연결]")
	Infof("[SSE] 새 채널 등록 완료 - reqId: %s", reqId)
}

func (s *SseManager) Deregister(reqId string) {
	Infof("[SSE] Deregister 시도 - reqId: %s", reqId)

	if value, ok := s.clients.Load(reqId); ok {
		info := value.(*ConnectionInfo)
		close(info.Channel)
		s.clients.Delete(reqId)
		Infof("[SSE] 채널 종료 및 삭제 완료 - reqId: %s", reqId)
	}

	// 제거 후 확인을 위한 모든 활성 연결 출력
	s.logActiveConnections("[Deregister 후 활성 연결]")
}

func (s *SseManager) Send(reqId string, msg string) {
	Infof("[SSE] Send 시도 - reqId: %s", reqId)

	// 전송 전 활성 연결 확인
	s.logActiveConnections("[Send 시도 시 활성 연결]")

	if value, ok := s.clients.Load(reqId); ok {
		info := value.(*ConnectionInfo)
		Infof("[SSE] 채널 찾음 - reqId: %s", reqId)

		select {
		case info.Channel <- msg:
			Infof("[SSE] 메시지 전송 성공 - reqId: %s, message: %s", reqId, msg)
		default:
			Warnf("[SSE] 메시지 전송 실패 (채널 막힘) - reqId: %s", reqId)
			s.Deregister(reqId)
		}
	} else {
		Warnf("[SSE] 채널을 찾을 수 없음 - reqId: %s", reqId)
	}
}

func (s *SseManager) GetActiveConnections() []types.Connection {
	connections := make([]types.Connection, 0)
	seen := make(map[string]bool)

	s.clients.Range(func(key, value interface{}) bool {
		reqId := key.(string)
		if seen[reqId] {
			Warnf("[SSE] 중복된 reqId 발견: %s", reqId)
			return true
		}

		info := value.(*ConnectionInfo)
		// 남은 유효 시간 계산
		expiresIn := s.validPeriod - time.Since(info.LastActive)

		connections = append(connections, types.Connection{
			ReqId:             reqId,
			ConnectedAt:       info.ConnectedAt,
			ConnectedDuration: time.Since(info.ConnectedAt),
			ExpiresIn:         expiresIn,
		})
		seen[reqId] = true
		return true
	})

	return connections
}

// 활성 연결 로깅을 위한 헬퍼 함수
func (s *SseManager) logActiveConnections(context string) {
	Infof("%s =========", context)
	count := 0
	s.clients.Range(func(key, value interface{}) bool {
		reqId := key.(string)
		info := value.(*ConnectionInfo)
		expiresIn := s.validPeriod - time.Since(info.LastActive)
		Infof("- reqId: %s, 연결시간: %s, 남은 유효시간: %s",
			reqId,
			info.ConnectedAt.Format(time.RFC3339),
			expiresIn.Round(time.Second))
		count++
		return true
	})
	Infof("총 %d개의 활성 연결 =========", count)
}

func (s *SseManager) monitorConnection(reqId string, info *ConnectionInfo) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-info.Done:
			return
		case <-ticker.C:
			timeLeft := s.validPeriod - time.Since(info.LastActive)

			// 만료 5분 전 경고
			if timeLeft > 0 && timeLeft <= 5*time.Minute {
				warningMsg := fmt.Sprintf("연결이 %.0f분 후 만료됩니다", timeLeft.Minutes())
				select {
				case info.Channel <- warningMsg:
					Infof("[SSE] 만료 경고 전송 - reqId: %s, 남은시간: %s", reqId, timeLeft)
				default:
					// 채널이 막혔다면 연결 종료
					s.Deregister(reqId)
					return
				}
			}

			// 만료된 경우
			if timeLeft <= 0 {
				Infof("[SSE] 연결 만료 - reqId: %s", reqId)
				s.Deregister(reqId)
				return
			}
		}
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
