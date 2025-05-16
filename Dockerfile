FROM golang:1.19-alpine AS builder

WORKDIR /app

# 의존성 설치
COPY go.mod go.sum ./
RUN go mod download

# 소스 코드 복사
COPY . .

# 빌드
RUN CGO_ENABLED=0 GOOS=linux go build -o ndns-router ./pkg

# 실행 이미지
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# 바이너리 복사
COPY --from=builder /app/ndns-router .

# 기본 환경 변수 설정
ENV PORT=8080
ENV APP_ENV=production
ENV MAX_REQUESTS=10
ENV REDIS_ADDR=redis:6379

# 주의: SERVER_LIST는 실행 시 환경 변수로 전달해야 합니다.
# 예: docker run -e SERVER_LIST=http://api1:3000,http://api2:3000 ...

EXPOSE 8080

CMD ["./ndns-router"] 