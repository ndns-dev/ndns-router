package main

import (
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/sh5080/ndns-router/pkg/configs"
	"github.com/sh5080/ndns-router/pkg/router"
	"github.com/sh5080/ndns-router/pkg/utils"
)

const (
	// 버전 정보
	Version = "dev"
)

func main() {
	// 앱 설정 로드
	config := configs.GetConfig()
	utils.Infof("설정 로드 완료 (환경: %s, 포트: %d, 최대 요청 수: %d)", config.Server.AppEnv, config.Server.Port, config.Server.MaxRequests)
	utils.Infof("서버 목록: %d개 서버", len(config.Server.ServerList))

	// 시스템 정보 출력
	printSystemInfo()
	utils.Info("환경 변수 로드 완료")

	// 라우터 서비스 생성 및 시작
	utils.Infof("NDNS Router 시작 중 (버전: %s, 환경: %s)", Version, config.Server.AppEnv)
	routerService := router.NewRouterService(config, Version)
	routerService.Start()

	// 우아한 종료 처리
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	// 종료 대기
	<-signalCh
	utils.Info("종료 신호 감지. 서버를 종료합니다...")

	// 마무리 작업
	routerService.Shutdown()
	utils.Info("서버가 안전하게 종료되었습니다.")
}

// 시스템 정보 출력
func printSystemInfo() {
	utils.Infof("시스템 정보: %s/%s", runtime.GOOS, runtime.GOARCH)
	utils.Infof("Go 버전: %s, 코어 수: %d", runtime.Version(), runtime.NumCPU())

	// 현재 작업 디렉토리 출력
	cwd, err := os.Getwd()
	if err == nil {
		utils.Infof("작업 디렉토리: %s", cwd)
	}

	// 환경 변수 출력
	utils.Debugf("GOMAXPROCS: %d", runtime.GOMAXPROCS(0))
}
