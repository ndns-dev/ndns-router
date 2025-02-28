# Go App Info
APP_NAME := ndns-router
MAIN_FILE := ./pkg/main.go
VERSION := $(shell git rev-parse --short HEAD)

# Docker Info
DOCKER_IMAGE := $(APP_NAME)
DOCKER_REPO := sh5080
PLATFORM := linux/amd64
PORT := 8080

# 로컬 Go 실행용
install:
	go mod tidy

build:
	go build -ldflags "-X 'main.Version=$(VERSION)'" -o $(APP_NAME) $(MAIN_FILE)

run:
	APP_ENV=dev go run $(MAIN_FILE)

run-prod:
	./$(APP_NAME)

# 크로스 컴파일 (리눅스용 바이너리 생성)
build-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-X 'main.Version=$(VERSION)'" -o $(APP_NAME) $(MAIN_FILE)

# Docker 빌드 (go build 포함)
docker-build: build-linux
	docker build --platform linux/amd64 -t $(DOCKER_IMAGE) .

# Docker 실행
docker-run:
	docker run \
		-e PORT=$(PORT) \
		-e APP_ENV=dev \
		-e MAX_REQUESTS=10 \
		-p $(PORT):$(PORT) \
		$(APP_NAME)

# Docker 쉘 진입
docker-shell:
	docker run --rm -it $(APP_NAME) sh

# Docker 배포 준비용 빌드 (tag 지정)
docker-tag:
	docker tag $(APP_NAME) $(DOCKER_REPO)/$(APP_NAME):latest
	docker tag $(APP_NAME) $(DOCKER_REPO)/$(APP_NAME):$(VERSION)

# Docker 이미지 푸시
docker-push: docker-tag
	docker push $(DOCKER_REPO)/$(APP_NAME):latest
	docker push $(DOCKER_REPO)/$(APP_NAME):$(VERSION)

# Docker 이미지 삭제
docker-clean:
	docker rmi $(DOCKER_IMAGE)
	docker rmi $(DOCKER_REPO)/$(APP_NAME):latest
	docker rmi $(DOCKER_REPO)/$(APP_NAME):$(VERSION)

# 로컬 바이너리 삭제
clean:
	rm -f $(APP_NAME)

# 로드 테스트 실행
load-test:
	@echo "로드 테스트 시작..."
	@for i in $$(seq 1 20); do \
		curl -s "http://localhost:$(PORT)/api/test" -H "Content-Type: application/json" -d '{"test":"data"}' > /dev/null & \
	done
	@echo "20개의 동시 요청을 보냈습니다."

# 라우터 상태 확인
check-status:
	curl -s "http://localhost:$(PORT)/router/status" | jq

# 서버 목록 확인
check-servers:
	curl -s "http://localhost:$(PORT)/router/servers" | jq

.PHONY: install build run run-prod build-linux docker-build docker-build-skip-go docker-run docker-shell docker-tag docker-push docker-clean clean load-test check-status check-servers 