package interfaces

import (
	"time"

	"github.com/sh5080/ndns-router/pkg/types"
)

// ServerService 서버 관리를 위한 서비스 인터페이스
type ServerService interface {
	// 서버 관리
	AddServer(serverId, url string) error
	RemoveServer(serverId string) error
	GetAllServers() ([]*types.Server, error)
	GetHealthyServers() ([]*types.Server, error)
	GetServer(serverId string) (*types.Server, error)
	UpdateServerMetrics(serverId string, metrics *types.Metrics) error
	GetOptimalServer() *types.OptimalServer

	// 메트릭 수집 관리
	StartMetricsCollection(interval time.Duration) error
	StopMetricsCollection() error
}

// PrometheusService는 프로메테우스 서비스 인터페이스입니다
type PrometheusService interface {
	CollectMetrics(appName string) (*types.Metrics, error)
	Close() error
}

// RouterService는 라우터 서비스 인터페이스입니다
type RouterService interface {
	Start() error
	Stop() error
	GetServerService() ServerService
}
