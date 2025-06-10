# 실행 이미지
FROM ubuntu:20.04

RUN apt-get update && apt-get install -y ca-certificates tzdata

WORKDIR /app

# 로컬에서 미리 빌드한 바이너리와 환경설정 파일 복사
COPY ./ndns-router .
COPY .env .

# 기본 환경 변수 설정
ENV PORT=8080
ENV APP_ENV=production
ENV MAX_REQUESTS=10

EXPOSE 8080

CMD ["./ndns-router"]