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

// startUsingServer marks a server as being used
func (s *serverServiceImpl) startUsingServer(serverId string) {
	s.mutex.RLock()
	state, exists := s.serverStates[serverId]
	s.mutex.RUnlock()

	if !exists {
		s.mutex.Lock()
		state = &ServerState{}
		s.serverStates[serverId] = state
		s.mutex.Unlock()
	}

	state.mutex.Lock()
	state.ActiveRequests++
	state.LastUsedTime = time.Now()
	state.mutex.Unlock()
}

// FinishUsingServer marks a server as no longer being used
func (s *serverServiceImpl) FinishUsingServer(serverId string) {
	s.mutex.RLock()
	state, exists := s.serverStates[serverId]
	s.mutex.RUnlock()

	if !exists {
		return
	}

	state.mutex.Lock()
	if state.ActiveRequests > 0 {
		state.ActiveRequests--
	}
	state.mutex.Unlock()
}

func (s *serverServiceImpl) SelectOptimalServer() *types.Server {
	// 서버리스 강제 사용 비율 체크
	randomValue := utils.NewCalculate().RandomFloat64()
	utils.Infof("서버리스 강제 사용 비율 체크: %.2f (기준: %.2f)", randomValue, configs.ServerlessForceRatio)

	if randomValue < configs.ServerlessForceRatio {
		utils.Info("서버리스로 강제 전환 (부하 분산)")
		return s.GetServerlessServer()
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	var excellentServers []*types.Server
	var goodServers []*types.Server

	for serverId, server := range s.servers {
		utils.Infof("서버 검사 중: %s", serverId)

		if !s.canUseServer(server.ServerId) {
			utils.Infof("사용 불가능한 서버: %s", serverId)
			continue
		}

		if server.Metrics.Score >= configs.ScoreExcellent {
			excellentServers = append(excellentServers, server)
		} else if server.Metrics.Score >= configs.ScoreGood {
			goodServers = append(goodServers, server)
		}
	}

	utils.Infof("분류 결과 - 최상위 서버: %d개, 양호 서버: %d개",
		len(excellentServers), len(goodServers))

	var selectedServer *types.Server
	if len(excellentServers) > 0 {
		selectedServer = s.selectRandomServer(excellentServers)
		utils.Infof("최상위 서버 선택: %s (점수: %.2f)",
			selectedServer.ServerId, selectedServer.Metrics.Score)
	} else if len(goodServers) > 0 {
		selectedServer = s.selectRandomServer(goodServers)
		utils.Infof("양호 서버 선택: %s (점수: %.2f)",
			selectedServer.ServerId, selectedServer.Metrics.Score)
	} else {
		utils.Info("적합한 서버가 없어 서버리스로 전환")
		return s.GetServerlessServer()
	}

	s.startUsingServer(selectedServer.ServerId)
	return selectedServer
}

func (s *serverServiceImpl) GetServerlessServer() *types.Server {
	serverIndex := utils.NewGenerate().NextRoundRobinIndex(len(configs.GetConfig().Serverless.Servers))
	selectedServer := configs.GetConfig().Serverless.Servers[serverIndex]

	return &types.Server{
		ServerId: selectedServer,
		URL:      selectedServer,
		Metrics: &types.Metrics{
			Score: 100,
		},
	}
}

func (s *serverServiceImpl) selectRandomServer(servers []*types.Server) *types.Server {
	if len(servers) == 0 {
		return nil
	}
	return servers[utils.NewCalculate().RandomInt(len(servers))]
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
