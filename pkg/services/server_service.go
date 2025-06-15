package services

import (
	"fmt"
	"sync"
	"time"

	"github.com/sh5080/ndns-router/pkg/configs"
	"github.com/sh5080/ndns-router/pkg/interfaces"
	"github.com/sh5080/ndns-router/pkg/types"
	"github.com/sh5080/ndns-router/pkg/utils"
)

// ServerState는 서버의 현재 상태를 관리합니다
type ServerState struct {
	ActiveRequests int       // 현재 활성 요청 수
	LastUsedTime   time.Time // 마지막 사용 시간
	mutex          sync.Mutex
}

// serverServiceImpl implements the ServerService interface
type serverServiceImpl struct {
	servers          map[string]*types.Server
	serverStates     map[string]*ServerState
	mutex            sync.RWMutex
	stopCollection   chan struct{}
	optimalServer    *types.OptimalServer // 최적 서버 정보 저장
	serverlessServer *types.Server
}

// NewServerService creates a new instance of ServerService
func NewServerService() (interfaces.ServerService, error) {
	return &serverServiceImpl{
		servers:          make(map[string]*types.Server),
		serverStates:     make(map[string]*ServerState),
		stopCollection:   make(chan struct{}),
		optimalServer:    nil,
		serverlessServer: nil,
	}, nil
}

// AddServer 새 서버 추가
func (s *serverServiceImpl) AddServer(serverId, serverUrl string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	server := &types.Server{
		ServerId:      serverId,
		ServerUrl:     serverUrl,
		CurrentStatus: string(types.StatusUnknown),
		LastUpdated:   time.Now(),
	}

	s.servers[serverId] = server
	utils.Infof("서버 추가됨: %s (%s)", serverId, serverUrl)
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

	for id, server := range s.servers {
		utils.Infof("서버 정보 [%s]: %+v", id, *server)
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

// canUseServer checks if a server can be used based on concurrent requests and cooldown
func (s *serverServiceImpl) canUseServer(serverId string) bool {
	state, exists := s.serverStates[serverId]
	if !exists {
		return true
	}

	state.mutex.Lock()
	defer state.mutex.Unlock()

	if state.ActiveRequests >= configs.MaxConcurrentRequests {
		return false
	}

	if time.Since(state.LastUsedTime) < configs.CooldownPeriod {
		return false
	}

	return true
}

func (s *serverServiceImpl) SelectOptimalServers() *types.ServerGroup {
	// 서버리스 강제 사용 비율 체크
	randomValue := utils.NewCalculate().RandomFloat64()
	utils.Infof("서버리스 강제 사용 비율 체크: %.2f (기준: %.2f)", randomValue, configs.ServerlessForceRatio)

	serverGroup := &types.ServerGroup{
		ServerlessServer: s.GetServerlessServer(),
		ForceServerless:  randomValue < configs.ServerlessForceRatio,
	}

	if serverGroup.ForceServerless {
		utils.Info("서버리스로 강제 전환 (부하 분산)")
		return serverGroup
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 서버들을 점수에 따라 분류
	for serverId, server := range s.servers {
		utils.Infof("서버 검사 중: %s", serverId)

		if !s.canUseServer(server.ServerId) {
			utils.Infof("사용 불가능한 서버: %s", serverId)
			continue
		}

		if server.Metrics.Score >= configs.ScoreExcellent {
			serverGroup.ExcellentServers = append(serverGroup.ExcellentServers, server)
		} else if server.Metrics.Score >= configs.ScoreGood {
			serverGroup.GoodServers = append(serverGroup.GoodServers, server)
		}
	}

	utils.Infof("분류 결과 - 최상위 서버: %d개, 양호 서버: %d개",
		len(serverGroup.ExcellentServers), len(serverGroup.GoodServers))

	return serverGroup
}

func (s *serverServiceImpl) GetServerlessServer() *types.Server {
	serverIndex := utils.NewGenerate().NextRoundRobinIndex(len(configs.GetConfig().Serverless.Servers))
	selectedServer := configs.GetConfig().Serverless.Servers[serverIndex]

	return &types.Server{
		ServerId:  selectedServer,
		ServerUrl: selectedServer,
		Metrics: &types.Metrics{
			Score: 100,
		},
	}
}

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

// UpdateServerInfo 서버의 URL과 메트릭스를 함께 업데이트합니다
func (s *serverServiceImpl) UpdateServerInfo(serverId string, serverUrl string, metrics *types.Metrics) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	server, exists := s.servers[serverId]
	if !exists {
		return fmt.Errorf("서버가 존재하지 않습니다: %s", serverId)
	}

	// URL과 메트릭스 업데이트
	server.ServerUrl = serverUrl
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

	utils.Infof("서버 정보 업데이트됨 - ServerId: %s, URL: %s, Score: %.2f", serverId, serverUrl, metrics.Score)
	return nil
}
