package utils

import (
	"fmt"
	"log"
	"os"
	"time"
)

var (
	// 로그 레벨 정의
	LevelDebug = "DEBUG"
	LevelInfo  = "INFO"
	LevelWarn  = "WARN"
	LevelError = "ERROR"
	LevelFatal = "FATAL"

	// 기본 로거 인스턴스
	logger = NewLogger()

	infoLogger  = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime)
	errorLogger = log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime)
	fatalLogger = log.New(os.Stderr, "FATAL: ", log.Ldate|log.Ltime)
)

// Logger는 애플리케이션의 로깅을 담당하는 구조체입니다
type Logger struct {
	debugLogger *log.Logger
	infoLogger  *log.Logger
	warnLogger  *log.Logger
	errorLogger *log.Logger
	fatalLogger *log.Logger
}

// NewLogger는 새로운 로거 인스턴스를 생성합니다
func NewLogger() *Logger {
	return &Logger{
		debugLogger: log.New(os.Stdout, "", log.LstdFlags),
		infoLogger:  log.New(os.Stdout, "", log.LstdFlags),
		warnLogger:  log.New(os.Stdout, "", log.LstdFlags),
		errorLogger: log.New(os.Stderr, "", log.LstdFlags),
		fatalLogger: log.New(os.Stderr, "", log.LstdFlags),
	}
}

// getTimePrefix는 로그 메시지에 현재 시간을 추가합니다
func getTimePrefix() string {
	return time.Now().Format("2006/01/02 15:04:05")
}

// formatMessage는 로그 메시지 형식을 지정합니다
func formatMessage(level, message string) string {
	return fmt.Sprintf("%s [%s] %s", getTimePrefix(), level, message)
}

// Debug는 디버그 레벨 로그를 출력합니다
func (l *Logger) Debug(message string) {
	l.debugLogger.Println(formatMessage(LevelDebug, message))
}

// Debugf는 형식화된 디버그 레벨 로그를 출력합니다
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.Debug(fmt.Sprintf(format, args...))
}

// Info는 정보 레벨 로그를 출력합니다
func (l *Logger) Info(message string) {
	l.infoLogger.Println(formatMessage(LevelInfo, message))
}

// Infof는 형식화된 정보 레벨 로그를 출력합니다
func (l *Logger) Infof(format string, args ...interface{}) {
	l.Info(fmt.Sprintf(format, args...))
}

// Warn은 경고 레벨 로그를 출력합니다
func (l *Logger) Warn(message string) {
	l.warnLogger.Println(formatMessage(LevelWarn, message))
}

// Warnf는 형식화된 경고 레벨 로그를 출력합니다
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.Warn(fmt.Sprintf(format, args...))
}

// Error는 오류 레벨 로그를 출력합니다
func (l *Logger) Error(message string) {
	l.errorLogger.Println(formatMessage(LevelError, message))
}

// Errorf는 형식화된 오류 레벨 로그를 출력합니다
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.Error(fmt.Sprintf(format, args...))
}

// Fatal은 치명적 오류 로그를 출력하고 프로그램을 종료합니다
func (l *Logger) Fatal(message string) {
	l.fatalLogger.Println(formatMessage(LevelFatal, message))
	os.Exit(1)
}

// Fatalf는 형식화된 치명적 오류 로그를 출력하고 프로그램을 종료합니다
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.Fatal(fmt.Sprintf(format, args...))
}

// 전역 메서드들

// Debug는 디버그 레벨 로그를 출력합니다
func Debug(message string) {
	logger.Debug(message)
}

// Debugf는 형식화된 디버그 레벨 로그를 출력합니다
func Debugf(format string, args ...interface{}) {
	logger.Debugf(format, args...)
}

// Info는 정보 레벨 로그를 출력합니다
func Info(msg string) {
	infoLogger.Println(msg)
}

// Infof는 포맷된 정보 메시지를 로깅합니다
func Infof(format string, args ...interface{}) {
	infoLogger.Printf(format+"\n", args...)
}

// Warn은 경고 레벨 로그를 출력합니다
func Warn(message string) {
	logger.Warn(message)
}

// Warnf는 형식화된 경고 레벨 로그를 출력합니다
func Warnf(format string, args ...interface{}) {
	logger.Warnf(format, args...)
}

// Error는 에러 메시지를 로깅합니다
func Error(msg string) {
	errorLogger.Println(msg)
}

// Errorf는 포맷된 에러 메시지를 로깅합니다
func Errorf(format string, args ...interface{}) {
	errorLogger.Printf(format+"\n", args...)
}

// Fatal은 치명적인 에러를 로깅하고 프로그램을 종료합니다
func Fatal(msg string) {
	fatalLogger.Println(msg)
	os.Exit(1)
}

// Fatalf는 포맷된 치명적인 에러를 로깅하고 프로그램을 종료합니다
func Fatalf(format string, args ...interface{}) {
	fatalLogger.Printf(format+"\n", args...)
	os.Exit(1)
}
