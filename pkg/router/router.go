package router

import (
	"net/http"
	"strconv"
	"time"

	"github.com/sh5080/ndns-router/pkg/configs"
	"github.com/sh5080/ndns-router/pkg/services"
	"github.com/sh5080/ndns-router/pkg/utils"
)

// RouterService는 NDNS 라우터 서비스를 관리합니다
type RouterService struct {
	config          *configs.RouterConfig
	version         string
	storageService  services.StorageService
	serverService   services.ServerService
	healthService   services.HealthService
	balancerService services.BalancerService
	proxyService    services.ProxyService
	startTime       time.Time // 서버 시작 시간
}

// NewRouterService는 새로운 라우터 서비스 인스턴스를 생성합니다
func NewRouterService(config *configs.RouterConfig, version string) *RouterService {
	return &RouterService{
		config:  config,
		version: version,
	}
}

// Start는 라우터 서비스를 초기화하고 시작합니다
func (rs *RouterService) Start() {
	// 시작 시간 기록
	rs.startTime = time.Now()

	// 스토리지 서비스 초기화
	rs.initStorageService()

	// 스토리지 어댑터 생성
	storageAdapter := services.NewStorageAdapter(rs.storageService)

	// 서버 초기화
	initializer := NewServerInitializer(rs.config, storageAdapter)
	validCount, _ := initializer.InitializeServers()

	// 서버 서비스 초기화
	rs.serverService = services.NewServerService(storageAdapter)

	// 서버 목록 동기화 시작 (10초 간격)
	rs.serverService.StartSync(10 * time.Second)

	// 서버 서비스 재로드
	if err := rs.serverService.LoadServersFromStorage(); err != nil {
		utils.Warnf("서버 목록 로드 실패: %v", err)
	}

	// 서버 상태 확인
	utils.Infof("서버 서비스 초기화 완료: %d개 정상 서버", validCount)

	// 밸런서 서비스 초기화
	rs.balancerService = services.NewBalancerService(rs.serverService)

	// 헬스 서비스 초기화
	rs.healthService = services.NewHealthService(rs.serverService, 30*time.Second)

	// 프록시 서비스 초기화
	rs.proxyService = services.NewProxyService(rs.serverService, rs.balancerService)

	// 헬스 체커 시작
	rs.healthService.Start()

	// API 엔드포인트 설정
	rs.setupRoutes()

	// HTTP 서버 시작
	rs.startServer()
}

// Shutdown은 라우터 서비스를 종료합니다
func (rs *RouterService) Shutdown() {
	// 구현된 서비스들의 정리 작업 수행
	if rs.healthService != nil {
		rs.healthService.Stop()
	}

	if rs.serverService != nil {
		rs.serverService.StopSync()
	}

	if rs.storageService != nil {
		rs.storageService.Disconnect()
	}
}

// initStorageService는 스토리지 서비스를 초기화합니다
func (rs *RouterService) initStorageService() {
	// Redis 스토리지 서비스 생성
	rs.storageService = services.NewRedisStorageService()

	// Redis 연결
	if err := rs.storageService.Connect(rs.config.RedisURL); err != nil {
		utils.Fatalf("Redis 연결 실패: %v. NDNS Router는 Redis 연결이 필요합니다.", err)
	}

	utils.Info("Redis 스토리지 연결 성공")
}

// setupRoutes는 API 엔드포인트를 설정합니다
func (rs *RouterService) setupRoutes() {
	// 메인 프록시 엔드포인트
	http.HandleFunc("/", rs.proxyService.ServeHTTP)

	// 관리 엔드포인트
	http.HandleFunc("/router/status", rs.handleRouterStatus)
	http.HandleFunc("/router/servers", rs.handleServersStatus)
	http.HandleFunc("/router/health", rs.handleManualHealthCheck)
	http.HandleFunc("/router/metrics", rs.handleMetrics)
}

// startServer는 HTTP 서버를 시작합니다
func (rs *RouterService) startServer() {
	port := strconv.Itoa(rs.config.Server.Port)
	utils.Infof("NDNS Router 시작 (포트: %s, 환경: %s)", port, rs.config.Server.AppEnv)
	utils.Info("프록시 준비 완료. 요청을 수신합니다...")

	// 서버 시작
	go func() {
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			utils.Fatalf("서버 시작 실패: %v", err)
		}
	}()
}
