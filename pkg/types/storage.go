package types

// Storage 서버 상태 저장소 인터페이스
type Storage interface {
	// 서버 관리
	AddServer(serverURL string) error                                                    // 서버 URL 추가
	RemoveServer(serverURL string) error                                                 // 서버 URL 제거
	GetAllServers() ([]string, error)                                                    // 모든 서버 URL 조회
	InitializeServer(serverURL string, maxRequests int) error                            // 서버 초기화 (존재하지 않는 경우만)
	ValidateAndInitializeServer(serverURL string, maxRequests int, isHealthy bool) error // 헬스체크 결과에 따라 서버 초기화 또는 제거

	// 서버 상태 관리
	UpdateServerStatus(status ServerState) error           // 서버 상태 정보 업데이트
	GetServerStatus(serverURL string) (ServerState, error) // 서버 상태 정보 조회
	GetAllServerStatus() (map[string]ServerState, error)   // 모든 서버 상태 정보 조회

	// 최적화된 액세스 메서드
	GetServerHealth(serverURL string) (bool, error)   // 서버 건강 상태만 빠르게 조회
	GetServerLoad(serverURL string) (int, int, error) // 서버 부하 정보만 빠르게 조회
	GetHealthyServers() ([]string, error)             // 건강한 서버 목록 조회

	// 자원 정리
	Close() error // 저장소 연결 종료
}
