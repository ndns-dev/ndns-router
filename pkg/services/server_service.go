package services

import (
	"fmt"
	"sync"
	"time"

	"github.com/sh5080/ndns-router/pkg/interfaces"
	"github.com/sh5080/ndns-router/pkg/types"
	"github.com/sh5080/ndns-router/pkg/utils"
)

// serverServiceImpl 서버 서비스 구현체
type serverServiceImpl struct {
	servers          map[string]*types.Server // serverId -> Server 매핑
	mutex            *sync.RWMutex
	stopCollection   chan struct{}
	prometheusClient interfaces.PrometheusService
	optimalServer    *types.OptimalServer // 최적 서버 정보 저장
}

// NewServerService 새 서버 서비스 생성
func NewServerService(prometheusURL string) (interfaces.ServerService, error) {
	prometheusClient, err := NewPrometheusService(prometheusURL)
	if err != nil {
		return nil, fmt.Errorf("프로메테우스 클라이언트 생성 실패: %w", err)
	}

	return &serverServiceImpl{
		servers:          make(map[string]*types.Server),
		mutex:            &sync.RWMutex{},
		stopCollection:   make(chan struct{}),
		prometheusClient: prometheusClient,
		optimalServer:    nil,
	}, nil
}

// AddServer 새 서버 추가
func (s *serverServiceImpl) AddServer(serverId, url string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	server := &types.Server{
		ServerId:      serverId,
		URL:           url,
		CurrentStatus: string(types.StatusUnknown),
		LastUpdated:   time.Now(),
	}

	s.servers[serverId] = server
	utils.Infof("서버 추가됨: %s (%s)", serverId, url)
	return nil
}

// RemoveServer 서버 제거
func (s *serverServiceImpl) RemoveServer(serverId string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.servers, serverId)
	utils.Infof("서버 제거됨: %s", serverId)
	return nil
}

// GetAllServers 모든 서버 조회
func (s *serverServiceImpl) GetAllServers() ([]*types.Server, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	servers := make([]*types.Server, 0, len(s.servers))
	for _, server := range s.servers {
		servers = append(servers, server)
	}

	return servers, nil
}

// GetHealthyServers 건강한 서버만 조회
func (s *serverServiceImpl) GetHealthyServers() ([]*types.Server, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	servers := make([]*types.Server, 0)
	for _, server := range s.servers {
		if server.CurrentStatus == string(types.StatusHealthy) {
			servers = append(servers, server)
		}
	}

	return servers, nil
}

// GetServer 특정 서버 조회
func (s *serverServiceImpl) GetServer(serverId string) (*types.Server, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	server, exists := s.servers[serverId]
	if !exists {
		return nil, nil
	}

	return server, nil
}

// StartMetricsCollection 메트릭 수집 시작
func (s *serverServiceImpl) StartMetricsCollection(interval time.Duration) error {
	if interval == 0 {
		interval = 30 * time.Second
	}

	utils.Infof("메트릭 수집 시작 (간격: %s)", interval)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := s.collectMetrics(); err != nil {
					utils.Warnf("메트릭 수집 실패: %v", err)
				}
			case <-s.stopCollection:
				utils.Info("메트릭 수집 중지됨")
				return
			}
		}
	}()

	return nil
}

// StopMetricsCollection 메트릭 수집 중지
func (s *serverServiceImpl) StopMetricsCollection() error {
	close(s.stopCollection)
	return nil
}

// collectMetrics 프로메테우스에서 메트릭 수집
func (s *serverServiceImpl) collectMetrics() error {
	s.mutex.RLock()
	servers := make([]*types.Server, 0, len(s.servers))
	for _, server := range s.servers {
		servers = append(servers, server)
	}
	s.mutex.RUnlock()

	for _, server := range servers {
		metrics, err := s.prometheusClient.CollectMetrics(server.ServerId)
		if err != nil {
			utils.Errorf("메트릭 수집 실패 (%s): %v", server.ServerId, err)
			continue
		}

		s.mutex.Lock()
		server.Metrics = metrics
		server.CurrentStatus = string(utils.EvaluateServerHealth(metrics))
		server.LastUpdated = time.Now()

		// 최적 서버 업데이트 (받은 점수 기준)
		if s.optimalServer == nil || metrics.Score > s.optimalServer.Score {
			s.optimalServer = &types.OptimalServer{
				ServerId: server.ServerId,
				Score:    metrics.Score,
			}
			utils.Infof("새로운 최적 서버가 선택되었습니다: %s (점수: %.2f)", server.ServerId, metrics.Score)
		}
		s.mutex.Unlock()
	}

	return nil
}

// UpdateServerMetrics 서버 메트릭 업데이트
func (s *serverServiceImpl) UpdateServerMetrics(serverId string, metrics *types.Metrics) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	server, exists := s.servers[serverId]
	if !exists {
		return fmt.Errorf("서버가 존재하지 않습니다: %s", serverId)
	}

	server.Metrics = metrics
	server.LastUpdated = time.Now()

	// 최적 서버 업데이트 (받은 점수 기준)
	if s.optimalServer == nil || metrics.Score > s.optimalServer.Score {
		s.optimalServer = &types.OptimalServer{
			ServerId: serverId,
			Score:    metrics.Score,
		}
		utils.Infof("새로운 최적 서버가 선택되었습니다: %s (점수: %.2f)", serverId, metrics.Score)
	}

	return nil
}

// GetOptimalServer 현재 최적 서버 정보 반환
func (s *serverServiceImpl) GetOptimalServer() *types.OptimalServer {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.optimalServer
}
