package configs

import (
	"strings"
	"sync"

	"github.com/sh5080/ndns-router/pkg/utils"
	"github.com/spf13/viper"
)

// RouterConfig는 라우터 관련 환경 설정을 저장하는 구조체입니다
type RouterConfig struct {
	Server struct {
		Port        int      `mapstructure:"PORT"`
		AppEnv      string   `mapstructure:"APP_ENV"`
		ServerList  []string `mapstructure:"SERVER_LIST"`
		MaxRequests int      `mapstructure:"MAX_REQUESTS"`
	}
	RedisURL string `mapstructure:"REDIS_URL"`
}

var (
	configInstance *RouterConfig
	once           sync.Once
)

// loadConfig는 환경 변수를 로드하고 검증하는 내부 함수
func loadConfig() *RouterConfig {
	// Viper 초기화
	v := viper.New()
	v.AutomaticEnv()
	v.SetConfigFile(".env")

	// .env 파일 로드 (있는 경우만)
	if err := v.ReadInConfig(); err != nil {
		utils.Warnf(".env 파일 로드 실패: %v", err)
	}

	// 필수 환경 변수 목록
	requiredEnvVars := []string{
		"SERVER_LIST",
	}

	// Redis 관련 필수 환경 변수 (URL 또는 주소와 비밀번호)
	redisEnvVars := []string{
		"REDIS_URL",
	}

	// 필수 환경 변수 확인
	missingVars := []string{}
	for _, envVar := range requiredEnvVars {
		if !v.IsSet(envVar) || v.GetString(envVar) == "" {
			missingVars = append(missingVars, envVar)
		}
	}

	// Redis 관련 환경 변수 중 최소 하나는 있어야 함
	redisConfigFound := false
	for _, envVar := range redisEnvVars {
		if v.IsSet(envVar) && v.GetString(envVar) != "" {
			redisConfigFound = true
			break
		}
	}

	if !redisConfigFound {
		missingVars = append(missingVars, "REDIS_URL 또는 REDIS_ADDR")
	}

	// 필수 환경 변수가 없으면 에러 로깅 후 종료
	if len(missingVars) > 0 {
		utils.Fatalf("필수 환경 변수가 설정되지 않았습니다: %s", strings.Join(missingVars, ", "))
	}

	// 구성 인스턴스 생성
	conf := &RouterConfig{}

	// 서버 설정 로드
	conf.Server.Port = v.GetInt("PORT")
	conf.Server.AppEnv = v.GetString("APP_ENV")
	conf.Server.MaxRequests = v.GetInt("MAX_REQUESTS")

	// 서버 목록 로드
	serverListStr := v.GetString("SERVER_LIST")
	if serverListStr != "" {
		// 쉼표로 구분된 목록 파싱
		serverList := strings.Split(serverListStr, ",")
		for i, url := range serverList {
			url = strings.TrimSpace(url)
			// http:// 접두사 확인 및 추가
			if url != "" && !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
				url = "http://" + url
			}
			if url != "" {
				serverList[i] = url
			}
		}
		// 빈 URL 필터링
		filteredList := []string{}
		for _, url := range serverList {
			if url != "" {
				filteredList = append(filteredList, url)
			}
		}
		conf.Server.ServerList = filteredList
	} else {
		conf.Server.ServerList = []string{}
	}

	// Redis 설정 로드
	conf.RedisURL = v.GetString("REDIS_URL")

	// 구성 정보 로깅
	utils.Infof("로드 완료 (환경: %s, 포트: %d, 최대 요청 수: %d)",
		conf.Server.AppEnv, conf.Server.Port, conf.Server.MaxRequests)

	if len(conf.Server.ServerList) > 0 {
		utils.Infof("서버 목록: %d개 서버", len(conf.Server.ServerList))
	} else {
		utils.Infof("서버 목록이 비어 있습니다.")
	}

	return conf
}

// GetConfig는 RouterConfig의 싱글톤 인스턴스를 반환합니다.
// 처음 호출 시에만 환경 변수를 로드하고 이후 호출에서는 캐시된 인스턴스를 반환합니다.
func GetConfig() *RouterConfig {
	once.Do(func() {
		configInstance = loadConfig()
		utils.Infof("환경 변수 로드 완료")
	})
	return configInstance
}
