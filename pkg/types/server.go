package types

import (
	"time"
)

// ServerStatus는 서버의 상태를 나타냅니다
type ServerStatus string

const (
	// 서버 상태 정의
	StatusHealthy   ServerStatus = "healthy"   // 정상 상태
	StatusWarning   ServerStatus = "warning"   // 경고 상태
	StatusUnhealthy ServerStatus = "unhealthy" // 비정상 상태
	StatusUnknown   ServerStatus = "unknown"   // 알 수 없음
)

// OptimalServer는 최적 서버 정보를 나타내는 구조체입니다
type OptimalServer struct {
	ServerId string  `json:"serverId"`
	Score    float64 `json:"score"`
}

// Server는 서버 정보를 나타내는 구조체입니다
type Server struct {
	ServerId      string    `json:"serverId"`
	ServerUrl     string    `json:"serverUrl"`
	CurrentStatus string    `json:"status"`
	LastUpdated   time.Time `json:"lastUpdated"`
	Metrics       *Metrics  `json:"metrics,omitempty"`
}

// Metrics represents server metrics
type Metrics struct {
	CPUUsage    float64   `json:"cpuUsage"`     // CPU 사용량 (0-1)
	MemoryUsage float64   `json:"memoryUsage"`  // 메모리 사용량 (bytes)
	RequestRate float64   `json:"requestRate"`  // 초당 요청 수
	ErrorRate   float64   `json:"errorRate"`    // 에러율 (0-1)
	Latency     float64   `json:"responseTime"` // 응답 시간 (초)
	Score       float64   `json:"score"`        // 서버 점수 (0-100)
	Timestamp   time.Time `json:"timestamp"`    // 메트릭 수집 시간
}

// OptimalServerRequest는 최적 서버 등록 요청 구조체입니다
type OptimalServerRequest struct {
	Servers []struct {
		ServerId  string `json:"serverId"`
		ServerUrl string `json:"serverUrl"`
		Metrics   struct {
			CpuUsage     float64 `json:"cpuUsage"`
			MemoryUsage  float64 `json:"memoryUsage"`
			ErrorRate    float64 `json:"errorRate"`
			ResponseTime float64 `json:"responseTime"`
			Score        float64 `json:"score"`
		} `json:"metrics"`
	} `json:"servers"`
}

// ServerGroup 서버들을 그룹별로 관리하는 구조체
type ServerGroup struct {
	ExcellentServers []*Server // 최상위 서버 목록
	GoodServers      []*Server // 양호 서버 목록
	ServerlessServer *Server   // 서버리스 서버
	ForceServerless  bool      // 서버리스 강제 사용 여부
}
