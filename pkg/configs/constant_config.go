package configs

import "time"

// 내부 관리 경로
var InternalPaths = map[string]bool{
	"/servers":  true,
	"/metrics":  true,
	"/internal": true,
}

// 타임아웃 설정
const (
	// 프록시 요청 타임아웃
	ProxyTimeout = 3 * time.Second
	// 서버 상태 체크 타임아웃
	HealthCheckTimeout = 2 * time.Second
)

// 재시도 설정
const (
	// 최대 재시도 횟수
	MaxRetryAttempts = 3
	// 재시도 대기 시간
	RetryBackoff = 100 * time.Millisecond
)

// 서버 상태 임계값
const (
	// 서버 점수 기준
	ScoreExcellent = 80.0 // 최상 기준 점수
	ScoreGood      = 60.0 // 중간 기준 점수

	// 동시성 제어
	MaxConcurrentRequests = 10                     // 서버당 최대 동시 요청 수
	CooldownPeriod        = 100 * time.Millisecond // 서버 재사용 대기 시간

	// 서버리스 설정
	ServerlessForceRatio = 0.7 // 강제로 서버리스로 보낼 비율 (70%)
)
