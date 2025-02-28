package types

import "time"

// RouterConfig는 라우터의 설정을 정의합니다
type RouterConfig struct {
	// 서버 설정
	Server struct {
		Port   int    `env:"PORT,required"`
		AppEnv string `env:"APP_ENV,required"`
	}

	// 프로메테우스 설정
	PrometheusURL string `env:"PROMETHEUS_URL" envDefault:"http://localhost:9090"`

	// 서버리스 설정
	Serverless struct {
		Servers []string `env:"SERVERLESS_SERVERS" envSeparator:","`
		Weight  int      `env:"SERVERLESS_WEIGHT" envDefault:"30"` // 서버리스 라우팅 가중치 (%)
	}

	// 온프레미스 설정
	OnPremise struct {
		Weight int `env:"ONPREM_WEIGHT" envDefault:"70"` // 온프레미스 라우팅 가중치 (%)
	}

	// 헬스체크 설정
	HealthCheck struct {
		Interval    int `env:"HEALTH_CHECK_INTERVAL" envDefault:"30"`   // 헬스체크 간격 (초)
		MaxRetries  int `env:"HEALTH_CHECK_MAX_RETRIES" envDefault:"3"` // 최대 재시도 횟수
		TimeoutSecs int `env:"HEALTH_CHECK_TIMEOUT" envDefault:"5"`     // 타임아웃 (초)
	}

	// 라우팅 설정
	Routing struct {
		// 온프레미스 API 설정
		OnPremise struct {
			Servers             []string      `env:"ONPREM_SERVERS" envSeparator:","`
			HealthCheckInterval time.Duration `env:"ONPREM_HEALTH_CHECK_INTERVAL" envDefault:"30s"`
			HealthCheckTimeout  time.Duration `env:"ONPREM_HEALTH_CHECK_TIMEOUT" envDefault:"5s"`
			RetryAttempts       int           `env:"ONPREM_RETRY_ATTEMPTS" envDefault:"3"`
			RetryDelay          time.Duration `env:"ONPREM_RETRY_DELAY" envDefault:"1s"`
		}

		// 서버리스 API 설정
		Serverless struct {
			CloudRunURL string `env:"CLOUD_RUN_URL,required"`
			LambdaURL   string `env:"LAMBDA_URL,required"`
			// 서버리스로 전환하는 조건
			FailoverThreshold struct {
				ErrorRate    float64 `env:"FAILOVER_ERROR_RATE" envDefault:"50"`      // 50% 이상 에러율
				ResponseTime float64 `env:"FAILOVER_RESPONSE_TIME" envDefault:"5000"` // 5000ms 이상 응답시간
				CPUUsage     float64 `env:"FAILOVER_CPU_USAGE" envDefault:"90"`       // 90% 이상 CPU 사용률
				MemoryUsage  float64 `env:"FAILOVER_MEMORY_USAGE" envDefault:"90"`    // 90% 이상 메모리 사용률
				HealthScore  float64 `env:"FAILOVER_HEALTH_SCORE" envDefault:"30"`    // 30점 이하 헬스스코어
			}
		}

		// 라우팅 가중치 (퍼센트)
		WeightDistribution struct {
			OnPremise int `env:"WEIGHT_ONPREMISE" envDefault:"70"` // 온프레미스 라우팅 비율
			CloudRun  int `env:"WEIGHT_CLOUD_RUN" envDefault:"15"` // Cloud Run 라우팅 비율
			Lambda    int `env:"WEIGHT_LAMBDA" envDefault:"15"`    // Lambda 라우팅 비율
		}
	}
}
