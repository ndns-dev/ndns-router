package services

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sh5080/ndns-router/pkg/types"
	"github.com/sh5080/ndns-router/pkg/utils"
)

// StorageAdapter는 StorageService와 types.Storage 간의 어댑터 역할을 합니다
type StorageAdapter struct {
	storageService StorageService
}

// NewStorageAdapter는 새 스토리지 어댑터를 생성합니다
func NewStorageAdapter(storageService StorageService) types.Storage {
	return &StorageAdapter{
		storageService: storageService,
	}
}

// AddServer는 서버를 추가합니다
func (sa *StorageAdapter) AddServer(serverURL string) error {
	client := sa.storageService.GetClient().(*redis.Client)
	ctx := sa.storageService.GetContext()
	key := sa.storageService.GetServerListKey()

	if err := client.SAdd(ctx, key, serverURL).Err(); err != nil {
		return fmt.Errorf("서버 목록에 추가 실패: %w", err)
	}

	// 서버 목록 만료시간 설정
	client.Expire(ctx, key, 24*time.Hour)

	utils.Infof("서버 추가됨: %s", serverURL)
	return nil
}

// RemoveServer는 서버를 제거합니다
func (sa *StorageAdapter) RemoveServer(serverURL string) error {
	client := sa.storageService.GetClient().(*redis.Client)
	ctx := sa.storageService.GetContext()

	// 서버 목록에서 제거
	if err := client.SRem(ctx, sa.storageService.GetServerListKey(), serverURL).Err(); err != nil {
		return fmt.Errorf("서버 목록에서 제거 실패: %w", err)
	}

	// 상태 정보 제거 (통합 해시에서)
	if err := client.HDel(ctx, sa.storageService.GetServerStateKey(), serverURL).Err(); err != nil {
		return fmt.Errorf("서버 상태 정보 제거 실패: %w", err)
	}

	utils.Infof("서버 제거됨: %s", serverURL)
	return nil
}

// GetAllServers는 모든 서버 목록을 반환합니다
func (sa *StorageAdapter) GetAllServers() ([]string, error) {
	client := sa.storageService.GetClient().(*redis.Client)
	ctx := sa.storageService.GetContext()

	// 서버 목록 조회 (Set)
	servers, err := client.SMembers(ctx, sa.storageService.GetServerListKey()).Result()
	if err != nil {
		return nil, fmt.Errorf("서버 목록 조회 실패: %w", err)
	}

	return servers, nil
}

// UpdateServerStatus는 서버 상태를 업데이트합니다
func (sa *StorageAdapter) UpdateServerStatus(status types.ServerState) error {
	client := sa.storageService.GetClient().(*redis.Client)
	ctx := sa.storageService.GetContext()

	// 상태 데이터 JSON 직렬화
	data, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("서버 상태 직렬화 실패: %w", err)
	}

	// 파이프라인으로 여러 작업 병합
	pipeline := client.Pipeline()

	// 통합 상태 정보 업데이트 (단일 해시에 저장)
	pipeline.HSet(ctx, sa.storageService.GetServerStateKey(), status.URL, string(data))

	// 만료 시간 설정
	pipeline.Expire(ctx, sa.storageService.GetServerStateKey(), 24*time.Hour)
	pipeline.Expire(ctx, sa.storageService.GetServerListKey(), 24*time.Hour)

	// 모든 명령 실행
	_, err = pipeline.Exec(ctx)
	if err != nil {
		return fmt.Errorf("서버 상태 업데이트 실패: %w", err)
	}

	return nil
}

// GetServerStatus는 서버 상태를 조회합니다
func (sa *StorageAdapter) GetServerStatus(serverURL string) (types.ServerState, error) {
	var status types.ServerState
	client := sa.storageService.GetClient().(*redis.Client)
	ctx := sa.storageService.GetContext()

	// 통합된 상태 정보 조회
	data, err := client.HGet(ctx, sa.storageService.GetServerStateKey(), serverURL).Result()
	if err == redis.Nil {
		// 데이터가 없으면 기본값 반환
		return types.ServerState{
			URL:         serverURL,
			Healthy:     true,
			CurrentLoad: 0,
			MaxRequests: 10,
			LastCheck:   time.Now(),
		}, nil
	} else if err != nil {
		return status, fmt.Errorf("서버 상태 조회 실패: %w", err)
	}

	// JSON 역직렬화
	if err := json.Unmarshal([]byte(data), &status); err != nil {
		return status, fmt.Errorf("서버 상태 역직렬화 실패: %w", err)
	}

	// URL 필드 설정 (정보 복원을 위해)
	status.URL = serverURL

	return status, nil
}

// GetAllServerStatus는 모든 서버의 상태를 조회합니다
func (sa *StorageAdapter) GetAllServerStatus() (map[string]types.ServerState, error) {
	result := make(map[string]types.ServerState)
	client := sa.storageService.GetClient().(*redis.Client)
	ctx := sa.storageService.GetContext()

	// 모든 서버 상태 정보 조회 (통합 해시에서)
	data, err := client.HGetAll(ctx, sa.storageService.GetServerStateKey()).Result()
	if err != nil {
		return result, fmt.Errorf("모든 서버 상태 조회 실패: %w", err)
	}

	// 각 서버 상태 파싱
	for url, statusJSON := range data {
		var status types.ServerState
		if err := json.Unmarshal([]byte(statusJSON), &status); err != nil {
			utils.Warnf("상태 파싱 오류 (URL: %s): %v", url, err)
			continue
		}

		// URL 필드 설정 (정보 복원을 위해)
		status.URL = url
		result[url] = status
	}

	return result, nil
}

// GetServerHealth는 서버의 건강 상태를 조회합니다
func (sa *StorageAdapter) GetServerHealth(serverURL string) (bool, error) {
	// 상태 정보에서 건강 상태만 추출
	status, err := sa.GetServerStatus(serverURL)
	if err != nil {
		return false, fmt.Errorf("서버 건강 상태 조회 실패: %w", err)
	}

	return status.Healthy, nil
}

// GetServerLoad는 서버의 부하 정보를 조회합니다
func (sa *StorageAdapter) GetServerLoad(serverURL string) (int, int, error) {
	// 상태 정보에서 부하 정보만 추출
	status, err := sa.GetServerStatus(serverURL)
	if err != nil {
		return 0, 0, fmt.Errorf("서버 부하 정보 조회 실패: %w", err)
	}

	return status.CurrentLoad, status.MaxRequests, nil
}

// GetHealthyServers는 건강한 서버 목록을 반환합니다
func (sa *StorageAdapter) GetHealthyServers() ([]string, error) {
	allStatus, err := sa.GetAllServerStatus()
	if err != nil {
		return nil, fmt.Errorf("건강한 서버 목록 조회 실패: %w", err)
	}

	healthyServers := make([]string, 0)
	for url, status := range allStatus {
		if status.Healthy {
			healthyServers = append(healthyServers, url)
		}
	}

	return healthyServers, nil
}

// ValidateAndInitializeServer는 서버를 검증하고 초기화합니다
func (sa *StorageAdapter) ValidateAndInitializeServer(serverURL string, maxRequests int, isHealthy bool) error {
	client := sa.storageService.GetClient().(*redis.Client)
	ctx := sa.storageService.GetContext()

	// 건강하지 않은 서버는 목록에서 제거하지 않고 상태만 업데이트
	if !isHealthy {
		utils.Infof("헬스체크 실패 서버: %s", serverURL)

		// 서버가 이미 존재하는지 확인
		exists, err := client.SIsMember(ctx, sa.storageService.GetServerListKey(), serverURL).Result()
		if err != nil {
			return fmt.Errorf("서버 존재 여부 확인 실패: %w", err)
		}

		// 존재하면 건강 상태만 업데이트
		if exists {
			state := types.ServerState{
				URL:         serverURL,
				Healthy:     false,
				CurrentLoad: 0,
				MaxRequests: maxRequests,
				LastCheck:   time.Now(),
			}

			return sa.UpdateServerStatus(state)
		}

		// 처음부터 비정상 서버는 추가
		if err := sa.AddServer(serverURL); err != nil {
			return err
		}

		state := types.ServerState{
			URL:         serverURL,
			Healthy:     false,
			CurrentLoad: 0,
			MaxRequests: maxRequests,
			LastCheck:   time.Now(),
		}

		return sa.UpdateServerStatus(state)
	}

	// 건강한 서버는 초기화 또는 업데이트
	// 서버가 이미 존재하는지 확인
	exists, err := client.SIsMember(ctx, sa.storageService.GetServerListKey(), serverURL).Result()
	if err != nil {
		return fmt.Errorf("서버 존재 여부 확인 실패: %w", err)
	}

	// 존재하지 않는 경우에만 초기화
	if !exists {
		if err := sa.AddServer(serverURL); err != nil {
			return err
		}

		// 기본 상태 설정
		status := types.ServerState{
			URL:         serverURL,
			Healthy:     true,
			CurrentLoad: 0,
			MaxRequests: maxRequests,
			LastCheck:   time.Now(),
		}

		if err := sa.UpdateServerStatus(status); err != nil {
			return err
		}

		utils.Infof("새 서버 초기화 완료: %s (최대 요청 수: %d)", serverURL, maxRequests)
	} else {
		// 기존 서버의 상태 업데이트
		serverStatus, err := sa.GetServerStatus(serverURL)
		if err != nil {
			return fmt.Errorf("기존 서버 상태 조회 실패: %w", err)
		}

		// 최대 요청 수 업데이트, 건강 상태 업데이트
		serverStatus.MaxRequests = maxRequests
		serverStatus.Healthy = true
		serverStatus.LastCheck = time.Now()

		if err := sa.UpdateServerStatus(serverStatus); err != nil {
			return fmt.Errorf("기존 서버 상태 업데이트 실패: %w", err)
		}

		utils.Infof("기존 서버 상태 업데이트: %s", serverURL)
	}

	return nil
}

// Close는 스토리지 연결을 종료합니다
func (sa *StorageAdapter) Close() error {
	return sa.storageService.Disconnect()
}

// InitializeServer는 서버를 초기화합니다
func (sa *StorageAdapter) InitializeServer(serverURL string, maxRequests int) error {
	// 서버가 이미 존재하는지 확인
	client := sa.storageService.GetClient().(*redis.Client)
	ctx := sa.storageService.GetContext()

	exists, err := client.SIsMember(ctx, sa.storageService.GetServerListKey(), serverURL).Result()
	if err != nil {
		return fmt.Errorf("서버 존재 여부 확인 실패: %w", err)
	}

	// 존재하지 않는 경우에만 초기화
	if !exists {
		if err := sa.AddServer(serverURL); err != nil {
			return err
		}

		// 기본 상태 설정
		status := types.ServerState{
			URL:         serverURL,
			Healthy:     true,
			CurrentLoad: 0,
			MaxRequests: maxRequests,
			LastCheck:   time.Now(),
		}

		if err := sa.UpdateServerStatus(status); err != nil {
			return err
		}

		utils.Infof("새 서버 초기화 완료: %s (최대 요청 수: %d)", serverURL, maxRequests)
	} else {
		// 기존 서버의 최대 요청 수 업데이트
		serverStatus, err := sa.GetServerStatus(serverURL)
		if err != nil {
			return fmt.Errorf("기존 서버 상태 조회 실패: %w", err)
		}

		// 최대 요청 수만 업데이트
		serverStatus.MaxRequests = maxRequests

		if err := sa.UpdateServerStatus(serverStatus); err != nil {
			return fmt.Errorf("기존 서버 상태 업데이트 실패: %w", err)
		}

		utils.Infof("기존 서버 상태 업데이트: %s", serverURL)
	}

	return nil
}
