package services

import (
	"sync"
	"time"

	"github.com/sh5080/ndns-router/pkg/types"
	"github.com/sh5080/ndns-router/pkg/utils"
)

// ServerService 서버 관리를 위한 서비스 인터페이스
type ServerService interface {
	// 서버 관리
	AddServer(url string, maxRequests int) error
	RemoveServer(url string) error
	GetAllServers() ([]*types.Server, error)
	GetHealthyServers() ([]*types.Server, error)
	GetServer(url string) (*types.Server, error)

	// 서버 상태 관리
	UpdateServerHealth(url string, isHealthy bool) error
	UpdateServerLoad(url string, load int) error
	IncrementServerLoad(url string) error
	DecrementServerLoad(url string) error

	// 동기화 관리
	StartSync(interval time.Duration) error
	StopSync() error
	LoadServersFromStorage() error
}

// serverServiceImpl 서버 서비스 구현체
type serverServiceImpl struct {
	storage types.Storage
	servers map[string]*types.Server
	mutex   *sync.RWMutex
	stopCh  chan struct{}
}

// NewServerService 새 서버 서비스 생성
func NewServerService(storage types.Storage) ServerService {
	return &serverServiceImpl{
		storage: storage,
		servers: make(map[string]*types.Server),
		mutex:   &sync.RWMutex{},
		stopCh:  make(chan struct{}),
	}
}

// AddServer 새 서버 추가
func (s *serverServiceImpl) AddServer(url string, maxRequests int) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 스토리지에 서버 추가
	if err := s.storage.AddServer(url); err != nil {
		return err
	}

	// 메모리에 서버 객체 생성
	server := &types.Server{
		URL:            url,
		CurrentStatus:  types.StatusUnknown,
		PreviousStatus: types.StatusUnknown,
		CurrentLoad:    0,
		MaxRequests:    maxRequests,
		IsHealthy:      true,
		LastUpdated:    time.Now(),
		LastResponse:   time.Now(),
	}

	s.servers[url] = server

	// 서버 상태 스토리지에 저장
	state := types.ServerState{
		URL:         url,
		Healthy:     true,
		CurrentLoad: 0,
		MaxRequests: maxRequests,
		LastCheck:   time.Now(),
	}

	if err := s.storage.UpdateServerStatus(state); err != nil {
		utils.Warnf("서버 상태 저장 실패: %v", err)
	}

	utils.Infof("서버 추가됨: %s", url)
	return nil
}

// RemoveServer 서버 제거
func (s *serverServiceImpl) RemoveServer(url string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 스토리지에서 서버 제거
	if err := s.storage.RemoveServer(url); err != nil {
		return err
	}

	// 메모리에서 서버 제거
	delete(s.servers, url)

	utils.Infof("서버 제거됨: %s", url)
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
		if server.IsHealthy && server.CurrentStatus == types.StatusHealthy {
			servers = append(servers, server)
		}
	}

	return servers, nil
}

// GetServer 특정 서버 조회
func (s *serverServiceImpl) GetServer(url string) (*types.Server, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	server, exists := s.servers[url]
	if !exists {
		return nil, nil
	}

	return server, nil
}

// UpdateServerHealth 서버 건강 상태 업데이트
func (s *serverServiceImpl) UpdateServerHealth(url string, isHealthy bool) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	server, exists := s.servers[url]
	if !exists {
		return nil
	}

	// 서버 상태 업데이트
	if isHealthy {
		server.UpdateServerStatus(types.StatusHealthy)
	} else {
		server.UpdateServerStatus(types.StatusUnhealthy)
		utils.Infof("서버가 비정상 상태로 표시됨: %s", url)
	}

	server.LastResponse = time.Now()

	// 스토리지에 상태 업데이트
	state := types.ServerState{
		URL:         url,
		Healthy:     isHealthy,
		CurrentLoad: server.CurrentLoad,
		MaxRequests: server.MaxRequests,
		LastCheck:   time.Now(),
	}

	if err := s.storage.UpdateServerStatus(state); err != nil {
		utils.Warnf("스토리지에 서버 상태 업데이트 실패: %v", err)
		return err
	}

	return nil
}

// UpdateServerLoad 서버 부하 정보 업데이트
func (s *serverServiceImpl) UpdateServerLoad(url string, load int) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	server, exists := s.servers[url]
	if !exists {
		return nil
	}

	server.CurrentLoad = load

	// 스토리지에 상태 업데이트
	state := types.ServerState{
		URL:         url,
		Healthy:     server.IsHealthy,
		CurrentLoad: load,
		MaxRequests: server.MaxRequests,
		LastCheck:   time.Now(),
	}

	if err := s.storage.UpdateServerStatus(state); err != nil {
		return err
	}

	return nil
}

// IncrementServerLoad 서버 부하 증가
func (s *serverServiceImpl) IncrementServerLoad(url string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	server, exists := s.servers[url]
	if !exists {
		return nil
	}

	server.IncrementActiveRequests()

	// 스토리지에 상태 업데이트
	state := types.ServerState{
		URL:         url,
		Healthy:     server.IsHealthy,
		CurrentLoad: server.CurrentLoad,
		MaxRequests: server.MaxRequests,
		LastCheck:   time.Now(),
	}

	if err := s.storage.UpdateServerStatus(state); err != nil {
		return err
	}

	return nil
}

// DecrementServerLoad 서버 부하 감소
func (s *serverServiceImpl) DecrementServerLoad(url string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	server, exists := s.servers[url]
	if !exists {
		return nil
	}

	server.DecrementActiveRequests()

	// 스토리지에 상태 업데이트
	state := types.ServerState{
		URL:         url,
		Healthy:     server.IsHealthy,
		CurrentLoad: server.CurrentLoad,
		MaxRequests: server.MaxRequests,
		LastCheck:   time.Now(),
	}

	if err := s.storage.UpdateServerStatus(state); err != nil {
		return err
	}

	return nil
}

// StartSync 서버 목록 동기화 시작
func (s *serverServiceImpl) StartSync(interval time.Duration) error {
	if interval == 0 {
		interval = 30 * time.Second
	}

	utils.Infof("서버 동기화 시작 (간격: %s)", interval)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := s.syncServersFromStorage(); err != nil {
					utils.Warnf("서버 목록 동기화 실패: %v", err)
				}
			case <-s.stopCh:
				utils.Info("서버 동기화 중지됨")
				return
			}
		}
	}()

	return nil
}

// StopSync 서버 목록 동기화 중지
func (s *serverServiceImpl) StopSync() error {
	close(s.stopCh)
	return nil
}

// syncServersFromStorage는 스토리지에서 서버 정보를 동기화합니다
func (s *serverServiceImpl) syncServersFromStorage() error {
	// 스토리지에서 모든 서버 URL 조회
	urls, err := s.storage.GetAllServers()
	if err != nil {
		return err
	}

	// 현재 메모리에 있는 서버 URL 맵 생성 (O(1) 검색을 위해)
	s.mutex.RLock()
	existingServers := make(map[string]bool)
	for url := range s.servers {
		existingServers[url] = true
	}
	s.mutex.RUnlock()

	// 새로운 서버를 메모리에 추가
	for _, url := range urls {
		if !existingServers[url] {
			// 새 서버 발견, 상세 정보 로드
			state, err := s.storage.GetServerStatus(url)
			if err != nil {
				utils.Warnf("새 서버(%s) 상태 조회 실패: %v", url, err)
				continue
			}

			// URL이 비어있는 경우, 키로 사용된 URL을 설정
			if state.URL == "" {
				state.URL = url
			}

			// 메모리에 서버 객체 생성
			status := types.StatusUnknown
			if state.Healthy {
				status = types.StatusHealthy
			} else {
				status = types.StatusUnhealthy
			}

			server := &types.Server{
				URL:            url,
				CurrentStatus:  status,
				PreviousStatus: types.StatusUnknown,
				CurrentLoad:    state.CurrentLoad,
				MaxRequests:    state.MaxRequests,
				IsHealthy:      state.Healthy,
				LastUpdated:    time.Now(),
				LastResponse:   state.LastCheck,
			}

			// 메모리에 서버 추가
			s.mutex.Lock()
			s.servers[url] = server
			s.mutex.Unlock()

			utils.Infof("새 서버 추가됨: %s (건강: %v, 부하: %d/%d)",
				url, state.Healthy, state.CurrentLoad, state.MaxRequests)
		}
	}

	// 메모리에서 삭제된 서버 제거
	s.mutex.Lock()
	for url := range s.servers {
		found := false
		for _, storageURL := range urls {
			if url == storageURL {
				found = true
				break
			}
		}

		if !found {
			// 서버가 스토리지에서 제거됨, 메모리에서도 제거
			delete(s.servers, url)
			utils.Infof("서버 제거됨: %s (더 이상 스토리지에 존재하지 않음)", url)
		}
	}
	s.mutex.Unlock()

	return nil
}

// LoadServersFromStorage 스토리지에서 서버 목록을 불러와 메모리에 로드
func (s *serverServiceImpl) LoadServersFromStorage() error {
	// 스토리지에서 모든 서버 URL 조회
	urls, err := s.storage.GetAllServers()
	if err != nil {
		return err
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 기존 서버 맵 초기화
	s.servers = make(map[string]*types.Server)

	// 새로운 서버 목록 로드
	for _, url := range urls {
		// 서버 상태 조회
		state, err := s.storage.GetServerStatus(url)
		if err != nil {
			utils.Warnf("서버 상태 조회 실패 (%s): %v", url, err)
			continue
		}

		// URL 필드가 비어있는 경우, 키로 사용된 URL을 설정
		if state.URL == "" {
			state.URL = url
		}

		// 메모리에 서버 객체 생성
		status := types.StatusUnknown
		if state.Healthy {
			status = types.StatusHealthy
		} else {
			status = types.StatusUnhealthy
		}

		server := &types.Server{
			URL:            url,
			CurrentStatus:  status,
			PreviousStatus: types.StatusUnknown,
			CurrentLoad:    state.CurrentLoad,
			MaxRequests:    state.MaxRequests,
			IsHealthy:      state.Healthy,
			LastUpdated:    time.Now(),
			LastResponse:   state.LastCheck,
		}

		s.servers[url] = server
	}

	utils.Infof("스토리지에서 %d개 서버 로드 완료", len(s.servers))
	return nil
}
