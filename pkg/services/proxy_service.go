package services

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"github.com/sh5080/ndns-router/pkg/utils"
)

// ProxyService 프록시 기능을 위한 서비스 인터페이스
type ProxyService interface {
	// 요청 처리
	ServeHTTP(w http.ResponseWriter, r *http.Request)

	// 통계
	GetStats() map[string]interface{}
	ResetStats()
}

// proxyServiceImpl 프록시 서비스 구현체
type proxyServiceImpl struct {
	serverService   ServerService
	balancerService BalancerService
	stats           map[string]int64
	mutex           *sync.RWMutex
	startTime       time.Time
}

// NewProxyService 새 프록시 서비스 생성
func NewProxyService(serverService ServerService, balancerService BalancerService) ProxyService {
	return &proxyServiceImpl{
		serverService:   serverService,
		balancerService: balancerService,
		stats: map[string]int64{
			"total_requests":   0,
			"success_requests": 0,
			"failed_requests":  0,
			"avg_response_ms":  0,
		},
		mutex:     &sync.RWMutex{},
		startTime: time.Now(),
	}
}

// ServeHTTP HTTP 요청 처리
func (p *proxyServiceImpl) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// 통계 업데이트
	p.mutex.Lock()
	p.stats["total_requests"]++
	p.mutex.Unlock()

	// 다음 서버 선택
	server, err := p.balancerService.NextServer()
	if err != nil || server == nil {
		utils.Warnf("서버 선택 실패: %v", err)
		p.mutex.Lock()
		p.stats["failed_requests"]++
		p.mutex.Unlock()
		http.Error(w, "모든 서버가 사용 불가능합니다", http.StatusServiceUnavailable)
		return
	}

	// 서버 URL 파싱
	targetURL, err := url.Parse(server.URL)
	if err != nil {
		utils.Warnf("서버 URL 파싱 실패: %v", err)
		p.mutex.Lock()
		p.stats["failed_requests"]++
		p.mutex.Unlock()
		http.Error(w, "잘못된 서버 URL", http.StatusInternalServerError)
		return
	}

	// 프록시 생성
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// 원래의 핸들러 저장
	originalDirector := proxy.Director

	// 요청 수정
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = targetURL.Host
		req.URL.Scheme = targetURL.Scheme
		req.URL.Host = targetURL.Host

		// X-Forwarded 헤더 추가
		req.Header.Set("X-Forwarded-Host", req.Host)
		req.Header.Set("X-Forwarded-Proto", req.URL.Scheme)
		req.Header.Set("X-Forwarded-For", req.RemoteAddr)

		// 라우터 헤더 추가
		req.Header.Set("X-NDNS-Router", "true")
		req.Header.Set("X-NDNS-Server", server.URL)
	}

	// 요청 전송 전 서버 부하 증가
	if err := p.serverService.IncrementServerLoad(server.URL); err != nil {
		utils.Warnf("서버 부하 증가 실패: %v", err)
	}

	// 응답 수정을 위한 래퍼
	proxy.ModifyResponse = func(resp *http.Response) error {
		// 응답 헤더에 라우터 정보 추가
		resp.Header.Set("X-NDNS-Routed-By", "NDNS-Router")
		resp.Header.Set("X-NDNS-Server", server.URL)
		return nil
	}

	// 에러 핸들러
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		utils.Warnf("프록시 요청 실패 (%s): %v", server.URL, err)
		p.mutex.Lock()
		p.stats["failed_requests"]++
		p.mutex.Unlock()

		// 서버를 비정상 상태로 표시
		if err := p.serverService.UpdateServerHealth(server.URL, false); err != nil {
			utils.Warnf("서버 상태 업데이트 실패: %v", err)
		}

		http.Error(w, "서버 응답 오류", http.StatusBadGateway)
	}

	// 프록시 처리
	proxy.ServeHTTP(w, r)

	// 요청 완료 후 서버 부하 감소
	if err := p.serverService.DecrementServerLoad(server.URL); err != nil {
		utils.Warnf("서버 부하 감소 실패: %v", err)
	}

	// 요청 처리 시간 계산 및 통계 업데이트
	elapsedMs := time.Since(startTime).Milliseconds()

	p.mutex.Lock()
	p.stats["success_requests"]++

	// 평균 응답 시간 업데이트 (이동 평균)
	prevAvg := p.stats["avg_response_ms"]
	p.stats["avg_response_ms"] = (prevAvg*9 + elapsedMs) / 10 // 90% 이전 평균 + 10% 새 값
	p.mutex.Unlock()

	utils.Infof("요청 처리 완료: %s -> %s (%d ms)", r.URL.Path, server.URL, elapsedMs)
}

// GetStats 통계 정보 조회
func (p *proxyServiceImpl) GetStats() map[string]interface{} {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	result := make(map[string]interface{})

	// 기본 통계 정보
	for k, v := range p.stats {
		result[k] = v
	}

	// 추가 정보
	result["uptime_seconds"] = int64(time.Since(p.startTime).Seconds())

	// 초당 요청 수 계산
	uptime := time.Since(p.startTime).Seconds()
	if uptime > 0 {
		result["requests_per_second"] = float64(p.stats["total_requests"]) / uptime
	} else {
		result["requests_per_second"] = 0.0
	}

	return result
}

// ResetStats 통계 초기화
func (p *proxyServiceImpl) ResetStats() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.stats = map[string]int64{
		"total_requests":   0,
		"success_requests": 0,
		"failed_requests":  0,
		"avg_response_ms":  0,
	}

	p.startTime = time.Now()
}
