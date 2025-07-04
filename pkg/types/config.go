package types

// EnvConfig는 라우터의 설정을 정의합니다
type EnvConfig struct {
	// 서버 설정
	Server struct {
		Port   int    `env:"PORT,required"`
		AppEnv string `env:"APP_ENV,required"`
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
