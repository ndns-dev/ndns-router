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
	URL           string    `json:"url"`
	CurrentStatus string    `json:"status"`
	LastUpdated   time.Time `json:"lastUpdated"`
	Metrics       *Metrics  `json:"metrics,omitempty"`
}
