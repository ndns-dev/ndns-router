package types

import (
	"sync"
)

// ServerPool은 서버 풀을 관리하는 구조체입니다
type ServerPool struct {
	servers    []*Server
	currentIdx int
	mutex      sync.RWMutex
}

// NewServerPool은 새로운 서버 풀을 생성합니다
func NewServerPool() *ServerPool {
	return &ServerPool{
		servers:    make([]*Server, 0),
		currentIdx: 0,
	}
}

// AddServer는 서버 풀에 서버를 추가합니다
func (sp *ServerPool) AddServer(server *Server) {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()

	sp.servers = append(sp.servers, server)
}

// RemoveServer는 서버 풀에서 서버를 제거합니다
func (sp *ServerPool) RemoveServer(url string) {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()

	for i, server := range sp.servers {
		if server.URL == url {
			// 현재 인덱스가 제거되는 서버 이후이면 인덱스 조정
			if sp.currentIdx >= i {
				if sp.currentIdx > 0 {
					sp.currentIdx--
				}
			}

			// 서버 제거
			sp.servers = append(sp.servers[:i], sp.servers[i+1:]...)
			break
		}
	}
}

// GetServerCount는 서버 풀의 서버 수를 반환합니다
func (sp *ServerPool) GetServerCount() int {
	sp.mutex.RLock()
	defer sp.mutex.RUnlock()

	return len(sp.servers)
}

// GetNextServer는 라운드로빈 방식으로 다음 서버를 반환합니다
func (sp *ServerPool) GetNextServer() *Server {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()

	if len(sp.servers) == 0 {
		return nil
	}

	// 인덱스 순환
	sp.currentIdx = (sp.currentIdx + 1) % len(sp.servers)
	return sp.servers[sp.currentIdx]
}

// GetAllServers는 모든 서버 목록을 반환합니다
func (sp *ServerPool) GetAllServers() []*Server {
	sp.mutex.RLock()
	defer sp.mutex.RUnlock()

	// 복사본 생성
	servers := make([]*Server, len(sp.servers))
	copy(servers, sp.servers)

	return servers
}

// GetHealthyServers는 건강한 서버 목록을 반환합니다
func (sp *ServerPool) GetHealthyServers() []*Server {
	sp.mutex.RLock()
	defer sp.mutex.RUnlock()

	healthyServers := make([]*Server, 0)
	for _, server := range sp.servers {
		if server.IsHealthy {
			healthyServers = append(healthyServers, server)
		}
	}
	return healthyServers
}
