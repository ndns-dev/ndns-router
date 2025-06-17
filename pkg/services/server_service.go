package services

import (
	"strings"
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
	serverGroup      *types.ServerGroup // 추가
	serverGroupMutex sync.RWMutex       // 서버 그룹용 별도 뮤텍스
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
func (s *serverServiceImpl) AddServer(server *types.Server) error {
	s.mutex.Lock()
	s.servers[server.ServerId] = server
	s.mutex.Unlock()

	utils.Infof("서버 추가됨: %s (%s)", server.ServerId, server.ServerUrl)

	// 서버 추가 후 바로 분류 실행
	s.classifyServers()
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
		if server.CurrentStatus == string(types.StatusGood) || server.CurrentStatus == string(types.StatusExcellent) {
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

// 서버 분류 함수 분리
func (s *serverServiceImpl) classifyServers() {
	s.serverGroupMutex.Lock()
	defer s.serverGroupMutex.Unlock()

	newGroup := &types.ServerGroup{
		ExcellentServers: make([]*types.Server, 0),
		GoodServers:      make([]*types.Server, 0),
	}

	s.mutex.RLock()
	for serverId, server := range s.servers {
		if !s.canUseServer(server.ServerId) {
			utils.Infof("사용 불가능한 서버: %s", serverId)
			continue
		}

		if server.ServerType == "wsl" {
			server.CurrentStatus = string(types.StatusGood)
			newGroup.GoodServers = append(newGroup.GoodServers, server)
			continue
		}

		if server.Metrics.Score >= configs.ScoreExcellent {
			newGroup.ExcellentServers = append(newGroup.ExcellentServers, server)
		} else if server.Metrics.Score >= configs.ScoreGood {
			newGroup.GoodServers = append(newGroup.GoodServers, server)
		}
	}
	s.mutex.RUnlock()

	s.serverGroup = newGroup
	utils.Infof("서버 분류 완료 - 최상위 서버: %d개, 양호 서버: %d개",
		len(newGroup.ExcellentServers), len(newGroup.GoodServers))
}

func (s *serverServiceImpl) GetServerGroup() *types.ServerGroup {
	return s.serverGroup
}

func (s *serverServiceImpl) GetServerlessServer() *types.Server {
	serverIndex := utils.NewGenerate().NextRoundRobinIndex(len(configs.GetConfig().Serverless.Servers))
	selectedServer := configs.GetConfig().Serverless.Servers[serverIndex]

	// URL에서 서브도메인만 추출
	serverDomain := strings.Split(strings.Replace(selectedServer, "https://", "", 1), ".")[0] // api3.ndns.site -> api3
	serverId := "ndns-" + serverDomain                                                        // ndns-api3

	return &types.Server{
		ServerId:  serverId,
		ServerUrl: selectedServer,
		Metrics: &types.Metrics{
			Score: 100,
		},
	}
}
