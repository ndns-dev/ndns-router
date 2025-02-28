package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/sh5080/ndns-router/pkg/configs"
	"github.com/sh5080/ndns-router/pkg/routers"
	"github.com/sh5080/ndns-router/pkg/utils"
)

const (
	// 버전 정보
	Version = "dev"
)

func main() {
	// 앱 설정 로드
	config := configs.GetConfig()
	utils.Infof("설정 로드 완료 (환경: %s, 포트: %d)", config.Server.AppEnv, config.Server.Port)

	// 시스템 정보 출력
	printSystemInfo()

	// Fiber 앱 설정
	app := fiber.New(fiber.Config{
		AppName:        "NDNS Router",
		ServerHeader:   "NDNS Router",
		ProxyHeader:    "X-Forwarded-For",
		BodyLimit:      10 * 1024 * 1024, // 10MB
		ReadBufferSize: 16384,            // 16KB
		JSONDecoder:    json.Unmarshal,
	})
	app.Use(logger.New())

	// 라우터 설정
	if err := routers.SetupRoutes(app, config); err != nil {
		utils.Fatalf("라우터 설정 실패: %v", err)
	}

	// 컨텍스트 생성
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 서버 시작
	go func() {
		port := config.Server.Port
		if err := app.Listen(fmt.Sprintf(":%d", port)); err != nil {
			utils.Fatalf("서버 시작 실패: %v", err)
		}
	}()

	// 종료 시그널 처리
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigCh:
		utils.Info("종료 시그널을 받았습니다. 서버를 종료합니다...")
	case <-ctx.Done():
		utils.Info("컨텍스트가 취소되었습니다. 서버를 종료합니다...")
	}

	// 서버 종료
	if err := app.Shutdown(); err != nil {
		utils.Errorf("서버 종료 실패: %v", err)
	}
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
