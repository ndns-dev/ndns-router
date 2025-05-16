package services

import (
	"net/http"
	"time"

	"github.com/sh5080/ndns-router/pkg/utils"
)

// HealthService 서버 헬스 체크를 위한 서비스 인터페이스
type HealthService interface {
	// 헬스 체크 관리
	Start() error
	Stop() error
	CheckServerHealth(url string) bool
	CheckAllServers() (int, int)
	SetHealthCheckInterval(duration time.Duration)
}

// healthServiceImpl 헬스 체크 서비스 구현체
type healthServiceImpl struct {
	serverService ServerService
	client        *http.Client
	checkInterval time.Duration
	stopCh        chan struct{}
}

// NewHealthService 새 헬스 체크 서비스 생성
func NewHealthService(serverService ServerService, checkInterval time.Duration) HealthService {
	if checkInterval == 0 {
		checkInterval = 30 * time.Second
	}

	return &healthServiceImpl{
		serverService: serverService,
		client: &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     30 * time.Second,
			},
		},
		checkInterval: checkInterval,
		stopCh:        make(chan struct{}),
	}
}

// Start 헬스 체크 시작
func (h *healthServiceImpl) Start() error {
	utils.Infof("헬스 체크 서비스 시작 (체크 간격: %s)", h.checkInterval)

	// 즉시 한 번 실행 후 주기적으로 실행
	go func() {
		h.CheckAllServers()

		ticker := time.NewTicker(h.checkInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				h.CheckAllServers()
			case <-h.stopCh:
				utils.Infof("헬스 체크 서비스 중지됨")
				return
			}
		}
	}()

	return nil
}

// Stop 헬스 체크 중지
func (h *healthServiceImpl) Stop() error {
	close(h.stopCh)
	return nil
}

// CheckServerHealth 특정 서버 헬스 체크 수행
func (h *healthServiceImpl) CheckServerHealth(url string) bool {
	// 헬스 체크 요청 생성
	req, err := http.NewRequest("GET", url+"/health", nil)
	if err != nil {
		utils.Warnf("서버 헬스 체크 요청 생성 실패 (%s): %v", url, err)
		h.serverService.UpdateServerHealth(url, false)
		return false
	}

	// 헬스 체크 요청 전송
	resp, err := h.client.Do(req)
	if err != nil {
		utils.Warnf("서버 헬스 체크 실패 (%s): %v", url, err)
		h.serverService.UpdateServerHealth(url, false)
		return false
	}
	defer resp.Body.Close()

	// 상태 코드 확인
	isHealthy := resp.StatusCode == http.StatusOK

	// 헬스 결과에 따라 서버 상태 업데이트
	if isHealthy {
		utils.Infof("서버 헬스 체크 성공: %s", url)
		h.serverService.UpdateServerHealth(url, true)
	} else {
		utils.Warnf("서버 헬스 체크 실패 (%s): 응답 코드 %d", url, resp.StatusCode)
		h.serverService.UpdateServerHealth(url, false)
	}

	return isHealthy
}

// CheckAllServers 모든 서버 헬스 체크 수행
func (h *healthServiceImpl) CheckAllServers() (int, int) {
	servers, err := h.serverService.GetAllServers()
	if err != nil {
		utils.Warnf("서버 목록 조회 실패: %v", err)
		return 0, 0
	}

	utils.Infof("전체 서버 헬스 체크 수행 중... (%d개 서버)", len(servers))

	var healthyCount, unhealthyCount int

	for _, server := range servers {
		isHealthy := h.CheckServerHealth(server.URL)
		if isHealthy {
			healthyCount++
		} else {
			unhealthyCount++
			utils.Warnf("서버 헬스 체크 실패: %s (비정상 상태로 표시됨, 라우팅에서 제외)", server.URL)
		}
	}

	utils.Infof("헬스 체크 완료: 전체 %d개 서버 중 정상 %d개, 비정상 %d개",
		len(servers), healthyCount, unhealthyCount)

	return healthyCount, unhealthyCount
}

// SetHealthCheckInterval 헬스 체크 간격 설정
func (h *healthServiceImpl) SetHealthCheckInterval(duration time.Duration) {
	if duration > 0 {
		h.checkInterval = duration
		utils.Infof("헬스 체크 간격 변경: %s", duration)
	}
}
