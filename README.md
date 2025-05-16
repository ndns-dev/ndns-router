# NDNS Router

NDNS Router는 여러 개의 동일한 NDNS API 서버 인스턴스에 대한 로드 밸런싱을 제공하는 Go 언어로 작성된 고성능 리버스 프록시 서버입니다.

## 주요 기능

- 여러 NDNS API 서버 인스턴스에 대한 로드 밸런싱
- 스마트 로드 밸런싱 전략 (최소 부하 + 라운드 로빈)
- 서버 이전/현재 상태 추적
- 자동 서버 상태 모니터링 및 헬스 체크
- Redis 또는 메모리 기반 서버 상태 저장
- 환경 변수를 통한 설정 관리

## 아키텍처

NDNS Router는 다음과 같은 주요 컴포넌트로 구성됩니다:

1. **HTTP 리버스 프록시**: 클라이언트 요청을 적절한 API 서버로 라우팅합니다.
2. **서버 풀 관리자**: API 서버 목록과 상태를 관리합니다.
3. **로드 밸런서**: 스마트 알고리즘을 이용해 최적의 서버를 선택합니다.
4. **헬스 체커**: 서버의 상태를 주기적으로 점검합니다.
5. **저장소**: Redis 기반 서버 상태 저장소를 제공합니다.
6. **환경 설정 관리자**: 애플리케이션 설정을 로드하고 관리합니다.

## 로드 밸런싱 전략

NDNS Router는 다음과 같은 로드 밸런싱 전략을 사용합니다:

- **스마트 밸런싱**: 최소 부하 상태와 라운드 로빈을 조합하여 최적의 서버를 선택합니다.
  - 가장 부하가 적은 서버를 먼저 선택합니다.
  - 모든 서버의 부하가 높은 경우 라운드 로빈으로 대체합니다.
  - 건강한 서버가 없는 경우에도 라운드 로빈을 사용합니다.

## 설치 및 실행

### 요구 사항

- Go 1.16 이상
- (선택) Redis 서버 (지속적인 서버 상태 관리를 위해)

### 빌드

```bash
go build -o ndns-router ./pkg
```

### 환경 변수 설정

환경 변수는 다음과 같은 방법으로 설정할 수 있습니다:

1. 시스템 환경 변수
2. `.env` 파일 (프로젝트 루트에 위치)

### 필요한 환경 변수

| 변수명 | 설명 | 기본값 | 필수 여부 |
|--------|------|--------|-----------|
| SERVER_LIST | NDNS API 서버 목록 (쉼표로 구분) | - | 필수 |
| PORT | 라우터 서버의 포트 번호 | 8080 | 선택 |
| APP_ENV | 애플리케이션 환경 (dev, prod) | dev | 선택 |
| MAX_REQUESTS | 서버당 최대 동시 요청 수 | 10 | 선택 |
| USE_REDIS | Redis 저장소 사용 여부 | false | 선택 |
| REDIS_ADDR | Redis 서버 주소 | localhost:6379 | 선택 |
| REDIS_PASSWORD | Redis 서버 비밀번호 | - | 선택 |
| REDIS_DB | Redis 데이터베이스 번호 | 0 | 선택 |

### 환경 변수 설정 예시

`.env` 파일 예시:
```
SERVER_LIST=http://api1.example.com,http://api2.example.com
PORT=9090
APP_ENV=prod
MAX_REQUESTS=15
USE_REDIS=true
REDIS_ADDR=redis.example.com:6379
REDIS_PASSWORD=secretpassword
REDIS_DB=1
```

### 실행 예시

```bash
# 환경 변수 직접 설정
SERVER_LIST="http://api1.example.com,http://api2.example.com" ./ndns-router

# 또는 .env 파일 사용
./ndns-router
```

## 저장소 선택

NDNS Router는 다음 두 가지 저장소 방식을 지원합니다:

### 1. 메모리 저장소 (기본값)

- 서버가 재시작되면 서버 상태 정보가 초기화됩니다.
- 다중 NDNS Router 인스턴스 간에 상태 공유가 불가능합니다.
- 추가 설정이 필요 없어 간단하게 사용할 수 있습니다.
- 개발 환경이나 단일 인스턴스 운영에 적합합니다.

### 2. Redis 저장소

- 서버가 재시작되어도 서버 상태 정보가 유지됩니다.
- 여러 NDNS Router 인스턴스 간에 서버 상태 공유가 가능합니다.
- Redis 설치 및 설정이 필요합니다.
- 프로덕션 환경이나 다중 인스턴스 운영에 적합합니다.

Redis 저장소를 사용하려면 다음 환경 변수를 설정하세요:
```
USE_REDIS=true
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=yourpassword  # 필요한 경우
REDIS_DB=0
```

## API 서버와의 통신

NDNS Router는 다음과 같은 방식으로 API 서버와 통신합니다:

1. **헬스 체크**: 정기적으로 각 API 서버의 `/health` 엔드포인트로 GET 요청을 보냅니다.
2. **프록시 요청**: 클라이언트의 모든 HTTP 요청을 선택된 API 서버로 전달합니다.

API 서버는 다음과 같은 헤더를 통해 라우터로부터 전달된 요청을 식별할 수 있습니다:

- `X-Proxy: NDNS-Router`
- `X-Proxy-Version: <router-version>`
- `X-Forwarded-For: <client-ip>`

## 라우터 관리 API

NDNS Router는 다음과 같은 관리 API 엔드포인트를 제공합니다:

- `GET /router/status`: 라우터의 현재 상태 정보를 반환합니다.
- `GET /router/servers`: 등록된 서버 목록과 각 서버의 상태 정보를 반환합니다.
- `GET /router/health?server=URL&status=STATUS`: API 서버의 상태를 수동으로 업데이트합니다.

## Docker 실행

### Docker 빌드
```bash
docker build -t ndns-router .
```

### Docker 실행
```bash
docker run -p 8080:8080 \
  -e SERVER_LIST="http://api1.example.com,http://api2.example.com" \
  -e MAX_REQUESTS=15 \
  -e USE_REDIS=true \
  -e REDIS_ADDR=redis-host:6379 \
  ndns-router
```

## 성능 최적화

NDNS Router는 성능을 최적화하기 위해 다음과 같은 기술을 사용합니다:

1. **동시성 제어**: Go의 고루틴과 채널을 이용한 비동기 작업 처리
2. **연결 풀링**: HTTP 연결 재사용으로 오버헤드 감소
3. **메모리 캐싱**: 서버 상태 정보를 메모리에 저장하여 빠른 접근
4. **스마트 로드 밸런싱**: 서버의 현재 부하 상태에 기반한 요청 분산
5. **Redis 지원**: 다중 인스턴스 환경에서 상태 공유로 효율성 향상

## 라이센스

이 프로젝트는 MIT 라이센스 하에 제공됩니다.

## Redis 스토리지 구조

NDNS 라우터는 Redis를 사용하여 서버 상태 정보를 저장하고 관리합니다. 다음과 같은 키 구조를 사용합니다:

- `ndns:router:servers:list` (Set): 등록된 NDNS API 서버 URL 목록
- `ndns:router:servers:status` (Hash): 각 서버의 전체 상태 정보 (JSON 형식)
- `ndns:router:servers:health` (Hash): 서버 건강 상태 (빠른 조회용)
- `ndns:router:servers:load` (Hash): 서버 부하 정보 (빠른 조회용)

이 구조를 통해 서버 재시작 후에도 상태 정보가 유지되며, 복잡한 분산 환경에서도 안정적인 운영이 가능합니다.

## 환경 변수 설정

NDNS 라우터는 다음 환경 변수를 통해 설정할 수 있습니다:

- `PORT`: 라우터 서버 포트 (기본값: 8080)
- `APP_ENV`: 애플리케이션 환경 (기본값: dev)
- `SERVER_LIST`: NDNS API 서버 URL 목록 (쉼표로 구분)
- `MAX_REQUESTS`: 서버당 최대 동시 요청 수 (기본값: 10)
- `REDIS_ADDR`: Redis 서버 주소 (기본값: localhost:6379)
- `REDIS_PASSWORD`: Redis 비밀번호 (기본값: 없음)
- `REDIS_DB`: Redis DB 번호 (기본값: 0)

## 실행 방법

```bash
# 환경 변수 설정
export SERVER_LIST=http://api1:3000,http://api2:3000,http://api3:3000
export REDIS_ADDR=redis:6379
export MAX_REQUESTS=15

# 실행
go run pkg/main.go
```

또는 Docker를 사용:

```bash
docker build -t ndns-router .
docker run -p 8080:8080 \
  -e SERVER_LIST=http://api1:3000,http://api2:3000,http://api3:3000 \
  -e REDIS_ADDR=redis:6379 \
  -e MAX_REQUESTS=15 \
  ndns-router
``` 