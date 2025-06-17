package interfaces

import (
	"github.com/sh5080/ndns-router/pkg/types"
)

// ServerService 서버 관리를 위한 서비스 인터페이스
type ServerService interface {
	// 서버 관리
	AddServer(server *types.Server) error
	RemoveServer(serverId string) error
	GetAllServers() ([]*types.Server, error)
	GetHealthyServers() ([]*types.Server, error)
	GetServer(serverId string) (*types.Server, error)
	GetServerGroup() *types.ServerGroup
	GetServerlessServer() *types.Server
}

// RouterService는 라우터 서비스 인터페이스입니다
type RouterService interface {
	Start() error
	Stop() error
	GetServerService() ServerService
}
