package configs

import (
	"log"
	"sync"

	"github.com/caarlos0/env/v6"
	"github.com/joho/godotenv"
)

type EnvConfig struct {
	// 서버 설정
	Server struct {
		Port   int    `env:"PORT,required"`
		AppEnv string `env:"APP_ENV,required"`
	}
	App struct {
		TestUrl   string `env:"TEST_URL,required"`
		Url       string `env:"URL,required"`
		JwtSecret string `env:"JWT_SECRET,required"`
	}

	// 서버리스 설정
	Serverless struct {
		Servers []string `env:"SERVERLESS_SERVERS" envSeparator:","`
	}
	// 라우팅 설정
	Routing struct {
		// 라우팅 가중치 (퍼센트)
		WeightDistribution struct {
			OnPremise int `env:"WEIGHT_ONPREMISE" envDefault:"70"` // 온프레미스 라우팅 비율
			CloudRun  int `env:"WEIGHT_CLOUD_RUN" envDefault:"15"` // Cloud Run 라우팅 비율
			Lambda    int `env:"WEIGHT_LAMBDA" envDefault:"15"`    // Lambda 라우팅 비율
		}
	}
}

var (
	configInstance *EnvConfig
	once           sync.Once
)

// GetConfig는 EnvConfig의 싱글톤 인스턴스를 반환합니다.
func GetConfig() *EnvConfig {
	once.Do(func() {
		// .env 파일 로드 시도
		if err := godotenv.Load(); err != nil {
			log.Printf(".env 파일 로드 실패 (무시됨): %v", err)
		}

		config := &EnvConfig{}
		if err := env.Parse(config); err != nil {
			log.Fatalf("환경 변수 로드 실패: %v", err)
		}

		configInstance = config
		log.Printf("환경 변수 로드 완료")
	})
	return configInstance
}
