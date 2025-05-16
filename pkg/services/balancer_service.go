package services

import (
	"sync"
	"time"

	"github.com/sh5080/ndns-router/pkg/types"
	"github.com/sh5080/ndns-router/pkg/utils"
)

// BalancerService 부하 분산을 위한 서비스 인터페이스
type BalancerService interface {
	// 서버 선택
	NextServer() (*types.Server, error)
	SetStrategy(strategy string) error
	GetCurrentStrategy() string

	// 통계
	GetStats() map[string]interface{}
	ResetStats()
}

// BalancerStrategy 부하 분산 전략 타입
type BalancerStrategy string

const (
	// 부하 분산 전략
	StrategyRoundRobin BalancerStrategy = "round_robin" // 라운드 로빈
	StrategyMinLoad    BalancerStrategy = "min_load"    // 최소 부하
	StrategySmart      BalancerStrategy = "smart"       // 스마트 (부하 기반 + 라운드 로빈)
)

// balancerServiceImpl 부하 분산 서비스 구현체
type balancerServiceImpl struct {
	serverService ServerService
	strategy      BalancerStrategy
	lastIndex     int
	mutex         *sync.RWMutex
	stats         map[string]int64
	lastReset     time.Time
}

// NewBalancerService 새 부하 분산 서비스 생성
func NewBalancerService(serverService ServerService) BalancerService {
	return &balancerServiceImpl{
		serverService: serverService,
		strategy:      StrategySmart,
		lastIndex:     -1,
		mutex:         &sync.RWMutex{},
		stats: map[string]int64{
			"total_requests": 0,
			"errors":         0,
		},
		lastReset: time.Now(),
	}
}

// NextServer 다음 서버 선택
func (b *balancerServiceImpl) NextServer() (*types.Server, error) {
	// 통계 업데이트
	b.mutex.Lock()
	b.stats["total_requests"]++
	b.mutex.Unlock()

	// 현재 전략에 따라 서버 선택
	switch b.strategy {
	case StrategyRoundRobin:
		return b.roundRobin()
	case StrategyMinLoad:
		return b.minLoad()
	case StrategySmart:
		return b.smart()
	default:
		return b.smart() // 기본값은 스마트 전략
	}
}

// roundRobin 라운드 로빈 방식으로 서버 선택
func (b *balancerServiceImpl) roundRobin() (*types.Server, error) {
	servers, err := b.serverService.GetHealthyServers()
	if err != nil {
		b.mutex.Lock()
		b.stats["errors"]++
		b.mutex.Unlock()
		return nil, err
	}

	if len(servers) == 0 {
		utils.Warnf("사용 가능한 건강한 서버가 없습니다")
		b.mutex.Lock()
		b.stats["errors"]++
		b.mutex.Unlock()
		return nil, nil
	}

	b.mutex.Lock()
	b.lastIndex = (b.lastIndex + 1) % len(servers)
	idx := b.lastIndex
	b.mutex.Unlock()

	return servers[idx], nil
}

// minLoad 최소 부하 서버 선택
func (b *balancerServiceImpl) minLoad() (*types.Server, error) {
	servers, err := b.serverService.GetHealthyServers()
	if err != nil {
		b.mutex.Lock()
		b.stats["errors"]++
		b.mutex.Unlock()
		return nil, err
	}

	if len(servers) == 0 {
		utils.Warnf("사용 가능한 건강한 서버가 없습니다")
		b.mutex.Lock()
		b.stats["errors"]++
		b.mutex.Unlock()
		return nil, nil
	}

	var minServer *types.Server
	var minRatio float64 = 1.0

	for _, server := range servers {
		// 비정상 서버는 제외
		if !server.IsHealthy || server.CurrentStatus != types.StatusHealthy {
			continue
		}

		// 부하 비율 계산
		ratio := server.GetLoadRatio()

		// 부하가 가장 적은 서버 선택
		if minServer == nil || ratio < minRatio {
			minServer = server
			minRatio = ratio
		}
	}

	// 선택한 서버가 없으면 라운드 로빈으로 선택
	if minServer == nil {
		utils.Warnf("부하 기반 서버 선택 실패, 라운드 로빈으로 대체")
		return b.roundRobin()
	}

	return minServer, nil
}

// smart 스마트 전략 (최소 부하 + 라운드 로빈)
func (b *balancerServiceImpl) smart() (*types.Server, error) {
	servers, err := b.serverService.GetHealthyServers()
	if err != nil {
		b.mutex.Lock()
		b.stats["errors"]++
		b.mutex.Unlock()
		return nil, err
	}

	if len(servers) == 0 {
		utils.Warnf("사용 가능한 건강한 서버가 없습니다")
		b.mutex.Lock()
		b.stats["errors"]++
		b.mutex.Unlock()
		return nil, nil
	}

	// 로그 레벨이 디버그일 때만 상세 정보 출력
	for _, server := range servers {
		utils.Infof("서버 상태: %s (상태: %s, 건강함: %v, 부하: %d/%d, 비율: %.2f)",
			server.URL, server.CurrentStatus, server.IsHealthy,
			server.CurrentLoad, server.MaxRequests, server.GetLoadRatio())
	}

	// 부하가 적은 서버 찾기
	var candidates []*types.Server
	maxLoad := 0.7 // 부하 기준 (70% 미만 서버만 고려)

	for _, server := range servers {
		if server.IsHealthy && server.CurrentStatus == types.StatusHealthy {
			if server.GetLoadRatio() < maxLoad {
				candidates = append(candidates, server)
			}
		}
	}

	// 부하가 적은 서버가 있으면 그 중에서 라운드 로빈으로 선택
	if len(candidates) > 0 {
		b.mutex.Lock()
		idx := (b.lastIndex + 1) % len(candidates)
		b.lastIndex = idx
		b.mutex.Unlock()
		utils.Infof("선택된 서버: %s (부하 비율: %.2f)",
			candidates[idx].URL, candidates[idx].GetLoadRatio())
		return candidates[idx], nil
	}

	// 모든 서버가 기준 이상의 부하이면 최소 부하 서버 선택
	return b.minLoad()
}

// SetStrategy 부하 분산 전략 설정
func (b *balancerServiceImpl) SetStrategy(strategy string) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	switch strategy {
	case string(StrategyRoundRobin):
		b.strategy = StrategyRoundRobin
	case string(StrategyMinLoad):
		b.strategy = StrategyMinLoad
	case string(StrategySmart):
		b.strategy = StrategySmart
	default:
		b.strategy = StrategySmart
	}

	utils.Infof("부하 분산 전략 변경: %s", b.strategy)
	return nil
}

// GetCurrentStrategy 현재 부하 분산 전략 조회
func (b *balancerServiceImpl) GetCurrentStrategy() string {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	return string(b.strategy)
}

// GetStats 통계 정보 조회
func (b *balancerServiceImpl) GetStats() map[string]interface{} {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	result := make(map[string]interface{})

	// 기본 통계 정보
	for k, v := range b.stats {
		result[k] = v
	}

	// 추가 정보
	result["uptime_seconds"] = int64(time.Since(b.lastReset).Seconds())
	result["current_strategy"] = string(b.strategy)

	return result
}

// ResetStats 통계 초기화
func (b *balancerServiceImpl) ResetStats() {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.stats = map[string]int64{
		"total_requests": 0,
		"errors":         0,
	}

	b.lastReset = time.Now()
}
