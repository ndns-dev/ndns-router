package router

import (
	"net/http"
	"time"

	"github.com/sh5080/ndns-router/pkg/configs"
	"github.com/sh5080/ndns-router/pkg/types"
	"github.com/sh5080/ndns-router/pkg/utils"
)

// ServerInitializer는 NDNS API 서버 초기화를 담당합니다
type ServerInitializer struct {
	config       *configs.RouterConfig
	storage      types.Storage
	healthClient *http.Client
}

// NewServerInitializer는 새로운 서버 초기화 객체를 생성합니다
func NewServerInitializer(config *configs.RouterConfig, storage types.Storage) *ServerInitializer {
	return &ServerInitializer{
		config:  config,
		storage: storage,
		healthClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// InitializeServers는 서버 목록을 초기화합니다
func (si *ServerInitializer) InitializeServers() (int, int) {
	// 서버 목록 확인
	if len(si.config.Server.ServerList) == 0 {
		utils.Fatal("유효한 NDNS API 서버가 없습니다. SERVER_LIST 환경 변수를 확인하세요.")
	}

	utils.Infof("서버 목록 확인: %d개 서버", len(si.config.Server.ServerList))

	// 유효한 서버와 비정상 서버 리스트
	validServers := make([]string, 0)
	invalidServers := make([]string, 0)

	// 각 서버 헬스체크 및 초기화
	for _, url := range si.config.Server.ServerList {
		utils.Infof("서버 헬스체크 중: %s", url)

		// 헬스체크 요청 생성
		req, err := http.NewRequest("GET", url+"/health", nil)
		if err != nil {
			utils.Warnf("서버 헬스체크 요청 생성 실패 (%s): %v", url, err)
			invalidServers = append(invalidServers, url)
			isHealthy := false

			// 비정상 서버도 목록에는 추가하되 상태를 비정상으로 표시
			if err := si.storage.ValidateAndInitializeServer(url, si.config.Server.MaxRequests, isHealthy); err != nil {
				utils.Warnf("서버 초기화/제거 실패 (%s): %v", url, err)
			}
			continue
		}

		// 헬스체크 요청 전송
		resp, err := si.healthClient.Do(req)
		isHealthy := true // 기본값

		if err != nil {
			utils.Warnf("서버 헬스체크 실패 (%s): %v", url, err)
			isHealthy = false
			invalidServers = append(invalidServers, url)
		} else {
			defer resp.Body.Close()

			// 응답 상태 확인
			if resp.StatusCode != http.StatusOK {
				utils.Warnf("서버 헬스체크 실패 (%s): 상태 코드 %d", url, resp.StatusCode)
				isHealthy = false
				invalidServers = append(invalidServers, url)
			} else {
				utils.Infof("서버 헬스체크 성공: %s", url)
				validServers = append(validServers, url)
			}
		}

		// 스토리지에 헬스체크 결과 기반으로 서버 초기화 또는 제거
		if err := si.storage.ValidateAndInitializeServer(url, si.config.Server.MaxRequests, isHealthy); err != nil {
			utils.Warnf("서버 초기화/제거 실패 (%s): %v", url, err)
		}
	}

	// 헬스체크 결과 요약 출력
	utils.Infof("헬스체크 결과: 총 %d개 서버 중 정상 %d개, 비정상 %d개",
		len(si.config.Server.ServerList), len(validServers), len(invalidServers))

	if len(invalidServers) > 0 {
		utils.Warnf("비정상 서버 목록: %v (이 서버들은 라우팅에서 제외됩니다)", invalidServers)
	}

	// 유효한 서버가 없는 경우 오류 발생
	if len(validServers) == 0 {
		utils.Fatal("헬스체크 통과한 유효한 API 서버가 없습니다. SERVER_LIST 환경 변수를 확인하세요.")
	}

	return len(validServers), len(invalidServers)
}
