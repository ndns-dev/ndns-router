package services

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sh5080/ndns-router/pkg/utils"
)

const (
	// Redis 키 접두사
	KeyPrefix = "ndns:router"

	// 서버 관련 키 (통합)
	ServerListKey  = KeyPrefix + ":servers:list"  // 서버 목록 (Set)
	ServerStateKey = KeyPrefix + ":servers:state" // 서버 상태 통합 정보 (Hash)

	// 만료 시간
	ServerDataTTL = time.Hour * 24 // 서버 데이터 만료 시간
)

// StorageService 저장소 관리를 위한 서비스 인터페이스
type StorageService interface {
	// Redis 연결 관리
	Connect(redisURL string) error
	Disconnect() error
	IsConnected() bool

	// 키 관리
	GetKeyPrefix() string
	GetServerListKey() string
	GetServerStateKey() string

	// Redis 클라이언트 가져오기
	GetClient() interface{}
	GetContext() context.Context
}

// redisStorageService Redis 저장소 서비스 구현체
type redisStorageService struct {
	client      *redis.Client
	ctx         context.Context
	isConnected bool
}

// NewRedisStorageService 새 Redis 저장소 서비스 생성
func NewRedisStorageService() StorageService {
	return &redisStorageService{
		ctx:         context.Background(),
		isConnected: false,
	}
}

// Connect Redis 연결 설정
func (r *redisStorageService) Connect(redisURL string) error {
	utils.Infof("Redis 연결 중: %s", redisURL)

	// Redis URL에서 클라이언트 생성
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return err
	}

	// 추가 설정
	opt.PoolSize = 10
	opt.MinIdleConns = 5
	opt.IdleTimeout = time.Minute * 5

	// 클라이언트 생성 및 연결 확인
	r.client = redis.NewClient(opt)

	// 연결 테스트
	if _, err := r.client.Ping(r.ctx).Result(); err != nil {
		return err
	}

	r.isConnected = true
	utils.Info("Redis 연결 성공")
	return nil
}

// Disconnect Redis 연결 종료
func (r *redisStorageService) Disconnect() error {
	if r.client != nil {
		err := r.client.Close()
		r.isConnected = false
		return err
	}
	return nil
}

// IsConnected Redis 연결 상태 확인
func (r *redisStorageService) IsConnected() bool {
	return r.isConnected
}

// GetKeyPrefix 키 접두사 가져오기
func (r *redisStorageService) GetKeyPrefix() string {
	return KeyPrefix
}

// GetServerListKey 서버 목록 키 가져오기
func (r *redisStorageService) GetServerListKey() string {
	return ServerListKey
}

// GetServerStateKey 서버 상태 통합 키 가져오기
func (r *redisStorageService) GetServerStateKey() string {
	return ServerStateKey
}

// GetClient Redis 클라이언트 가져오기
func (r *redisStorageService) GetClient() interface{} {
	return r.client
}

// GetContext Redis 컨텍스트 가져오기
func (r *redisStorageService) GetContext() context.Context {
	return r.ctx
}
