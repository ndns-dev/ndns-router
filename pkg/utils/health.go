package utils

import "github.com/sh5080/ndns-router/pkg/types"

// evaluateServerHealth 메트릭을 기반으로 서버 상태 평가
func EvaluateServerHealth(metrics *types.Metrics) types.ServerStatus {
	if metrics == nil {
		return types.StatusUnknown
	}

	// CPU 사용량이 90% 이상이면 비정상
	if metrics.CPUUsage > 0.9 {
		return types.StatusUnhealthy
	}

	// 메모리 사용량이 90% 이상이면 비정상 (8GB 기준)
	if metrics.MemoryUsage > 8*1024*1024*1024*0.9 {
		return types.StatusUnhealthy
	}

	// 에러율이 50% 이상이면 비정상
	if metrics.ErrorRate > 0.5 {
		return types.StatusUnhealthy
	}

	// 응답 시간이 5초 이상이면 비정상
	if metrics.Latency > 5.0 {
		return types.StatusUnhealthy
	}

	return types.StatusHealthy
}
