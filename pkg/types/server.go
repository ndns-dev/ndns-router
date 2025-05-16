package types

import (
	"sync"
	"time"
)

// ServerStatus 서버 상태 열거형
type ServerStatus string

const (
	// 서버 상태 정의
	StatusHealthy   ServerStatus = "healthy"   // 정상 상태
	StatusUnhealthy ServerStatus = "unhealthy" // 비정상 상태
	StatusUnknown   ServerStatus = "unknown"   // 알 수 없음
)

// Server는 NDNS API 서버 정보를 저장하는 구조체입니다
type Server struct {
	URL            string       // 서버 URL
	CurrentStatus  ServerStatus // 현재 서버 상태
	PreviousStatus ServerStatus // 이전 서버 상태
	LastUpdated    time.Time    // 마지막으로 상태가 업데이트된 시간

	// 부하 관련 필드
	CurrentLoad int // 현재 부하 (동시 요청 수)
	PrevLoad    int // 이전 부하 (동시 요청 수)
	MaxRequests int // 서버가 동시에 처리할 수 있는 최대 요청 수

	// 통계 필드
	RequestDuration int64 // 평균 요청 처리 시간 (밀리초)
	TotalRequests   int64 // 총 요청 수

	// 헬스 체크 필드
	IsHealthy    bool      // 서버 건강 상태 (빠른 접근용)
	LastResponse time.Time // 마지막 응답 시간 (헬스 체크용)

	mutex     sync.RWMutex // 데이터 접근 뮤텍스
	LoadMutex sync.Mutex   // 부하 업데이트용 뮤텍스
}

// ServerState 서버의 상태 정보를 표현합니다
type ServerState struct {
	URL         string    `json:"url,omitempty"` // URL은 레디스에서 필드 키로 사용되므로 JSON에서는 생략 가능
	Healthy     bool      `json:"healthy"`
	CurrentLoad int       `json:"current_load"`
	MaxRequests int       `json:"max_requests"`
	LastCheck   time.Time `json:"last_check"`
}

// UpdateServerStatus는 서버 상태를 업데이트합니다
func (s *Server) UpdateServerStatus(newStatus ServerStatus) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.PreviousStatus = s.CurrentStatus
	s.CurrentStatus = newStatus
	s.LastUpdated = time.Now()

	// 헬스 상태 업데이트
	s.IsHealthy = (newStatus == StatusHealthy)
}

// IncrementActiveRequests는 활성 요청 수를 증가시킵니다
func (s *Server) IncrementActiveRequests() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.CurrentLoad++
	s.TotalRequests++
}

// DecrementActiveRequests는 활성 요청 수를 감소시킵니다
func (s *Server) DecrementActiveRequests() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.CurrentLoad > 0 {
		s.CurrentLoad--
	}
}

// AddRequestDuration는 요청 처리 시간을 추가합니다
func (s *Server) AddRequestDuration(durationMs int64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 간단한 이동 평균 계산
	// 여기서는 기존 평균과 새 값의 중간값 사용
	if s.RequestDuration == 0 {
		s.RequestDuration = durationMs
	} else {
		s.RequestDuration = (s.RequestDuration + durationMs) / 2
	}
}

// IsAvailable은 서버가 새 요청을 처리할 수 있는지 확인합니다
func (s *Server) IsAvailable() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.IsHealthy && s.CurrentLoad < s.MaxRequests
}

// GetLoadRatio는 부하 비율을 계산합니다 (0.0 ~ 1.0)
func (s *Server) GetLoadRatio() float64 {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if s.MaxRequests <= 0 {
		return 1.0
	}
	return float64(s.CurrentLoad) / float64(s.MaxRequests)
}

// HasHighLoad는 높은 부하인지 확인합니다
func (s *Server) HasHighLoad() bool {
	return s.GetLoadRatio() >= 0.8 // 80% 이상의 부하를 높음으로 판단
}

// UpdateLoadSnapshot는 현재 부하를 이전 부하로 저장합니다
func (s *Server) UpdateLoadSnapshot() {
	s.LoadMutex.Lock()
	defer s.LoadMutex.Unlock()
	s.PrevLoad = s.CurrentLoad
}

// Copy는 서버 정보를 복사합니다
func (s *Server) Copy() *Server {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return &Server{
		URL:             s.URL,
		CurrentStatus:   s.CurrentStatus,
		PreviousStatus:  s.PreviousStatus,
		LastUpdated:     s.LastUpdated,
		CurrentLoad:     s.CurrentLoad,
		PrevLoad:        s.PrevLoad,
		MaxRequests:     s.MaxRequests,
		RequestDuration: s.RequestDuration,
		TotalRequests:   s.TotalRequests,
		IsHealthy:       s.IsHealthy,
		LastResponse:    s.LastResponse,
	}
}

// ToServerState 저장소용 상태 객체로 변환
func (s *Server) ToServerState() ServerState {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return ServerState{
		URL:         s.URL,
		Healthy:     s.IsHealthy,
		CurrentLoad: s.CurrentLoad,
		MaxRequests: s.MaxRequests,
		LastCheck:   s.LastResponse,
	}
}
