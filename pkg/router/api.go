package router

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/sh5080/ndns-router/pkg/utils"
)

// APIResponse는 모든 API 응답에 대한 표준 구조입니다
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Time    string      `json:"time"`
}

// newAPIResponse는 새로운 API 응답을 생성합니다
func newAPIResponse(success bool, message string, data interface{}, err string) APIResponse {
	return APIResponse{
		Success: success,
		Message: message,
		Data:    data,
		Error:   err,
		Time:    time.Now().Format(time.RFC3339),
	}
}

// sendJSONResponse는 JSON 응답을 전송합니다
func sendJSONResponse(w http.ResponseWriter, status int, response APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)
}

// handleRouterStatus는 라우터 상태 정보를 반환합니다
func (rs *RouterService) handleRouterStatus(w http.ResponseWriter, r *http.Request) {
	// 라우터 상태 정보 구성
	statusData := map[string]interface{}{
		"version":       rs.version,
		"env":           rs.config.Server.AppEnv,
		"uptime":        time.Since(rs.startTime).String(),
		"server_count":  0,
		"healthy_count": 0,
	}

	// 서버 수 정보 추가
	servers, err := rs.serverService.GetAllServers()
	if err == nil {
		statusData["server_count"] = len(servers)
	}

	// 건강한 서버 수 정보 추가
	healthyServers, err := rs.serverService.GetHealthyServers()
	if err == nil {
		statusData["healthy_count"] = len(healthyServers)
	}

	// 응답 반환
	response := newAPIResponse(true, "라우터 상태 정보", statusData, "")
	sendJSONResponse(w, http.StatusOK, response)
}

// handleServersStatus는 등록된 서버 목록과 상태를 반환합니다
func (rs *RouterService) handleServersStatus(w http.ResponseWriter, r *http.Request) {
	// 서버 목록 조회
	servers, err := rs.serverService.GetAllServers()
	if err != nil {
		utils.Errorf("서버 목록 조회 실패: %v", err)
		response := newAPIResponse(false, "", nil, "서버 목록 조회 실패")
		sendJSONResponse(w, http.StatusInternalServerError, response)
		return
	}

	// 서버 상태 정보 구성
	serverInfos := make([]map[string]interface{}, 0, len(servers))
	for _, server := range servers {
		serverInfo := map[string]interface{}{
			"url":          server.URL,
			"healthy":      server.IsHealthy,
			"status":       server.CurrentStatus,
			"current_load": server.CurrentLoad,
			"max_requests": server.MaxRequests,
			"load_ratio":   server.GetLoadRatio(),
			"last_check":   server.LastResponse.Format(time.RFC3339),
		}
		serverInfos = append(serverInfos, serverInfo)
	}

	// 응답 반환
	response := newAPIResponse(true, "서버 상태 목록", serverInfos, "")
	sendJSONResponse(w, http.StatusOK, response)
}

// handleManualHealthCheck는 수동 헬스 체크를 처리합니다
func (rs *RouterService) handleManualHealthCheck(w http.ResponseWriter, r *http.Request) {
	// 전체 헬스체크 수행
	healthyCount, unhealthyCount := rs.healthService.CheckAllServers()

	// 결과 응답
	healthData := map[string]interface{}{
		"total":      healthyCount + unhealthyCount,
		"healthy":    healthyCount,
		"unhealthy":  unhealthyCount,
		"check_time": time.Now().Format(time.RFC3339),
	}

	// 응답 반환
	response := newAPIResponse(true, "전체 서버 헬스체크 완료", healthData, "")
	sendJSONResponse(w, http.StatusOK, response)
}

// handleMetrics는 메트릭 정보를 반환합니다
func (rs *RouterService) handleMetrics(w http.ResponseWriter, r *http.Request) {
	// 각 서비스에서 통계 수집
	proxyStats := rs.proxyService.GetStats()
	balancerStats := rs.balancerService.GetStats()

	// 합쳐진 메트릭 구성
	metrics := map[string]interface{}{
		"uptime":           time.Since(rs.startTime).String(),
		"proxy":            proxyStats,
		"balancer":         balancerStats,
		"current_strategy": rs.balancerService.GetCurrentStrategy(),
	}

	// 서버 수 정보 추가
	servers, err := rs.serverService.GetAllServers()
	if err == nil {
		metrics["server_count"] = len(servers)
	}

	// 건강한 서버 수 정보 추가
	healthyServers, err := rs.serverService.GetHealthyServers()
	if err == nil {
		metrics["healthy_server_count"] = len(healthyServers)
	}

	// 응답 반환
	response := newAPIResponse(true, "메트릭 정보", metrics, "")
	sendJSONResponse(w, http.StatusOK, response)
}
