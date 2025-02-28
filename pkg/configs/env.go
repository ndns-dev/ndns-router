package configs

import (
	"sync"

	"github.com/caarlos0/env/v6"
	"github.com/joho/godotenv"
	"github.com/sh5080/ndns-router/pkg/types"
	"github.com/sh5080/ndns-router/pkg/utils"
)

var (
	configInstance *types.RouterConfig
	once           sync.Once
)

// GetConfig는 싱글톤 패턴으로 설정을 반환합니다
func GetConfig() *types.RouterConfig {
	once.Do(func() {
		// .env 파일 로드
		if err := godotenv.Load(); err != nil {
			utils.Warnf(".env 파일 로드 실패: %v", err)
		}

		config := &types.RouterConfig{}
		if err := env.Parse(config); err != nil {
			utils.Fatalf("환경 변수 로드 실패: %v", err)
		}

		// 가중치 합이 100인지 검증
		weights := config.Routing.WeightDistribution
		totalWeight := weights.OnPremise + weights.CloudRun + weights.Lambda
		if totalWeight != 100 {
			utils.Fatalf("라우팅 가중치의 합이 100이 아닙니다: %d", totalWeight)
		}

		configInstance = config
	})
	return configInstance
}
