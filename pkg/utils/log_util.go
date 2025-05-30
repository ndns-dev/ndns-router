package utils

import (
	"fmt"
	"log"
	"os"
	"time"
)

// Log levels
const (
	LevelDebug = "DEBUG"
	LevelInfo  = "INFO"
	LevelWarn  = "WARN"
	LevelError = "ERROR"
	LevelFatal = "FATAL"
)

// Log handles application logging
type Log struct {
	debug *log.Logger
	info  *log.Logger
	warn  *log.Logger
	error *log.Logger
	fatal *log.Logger
}

// NewLog creates a new Log instance
func NewLog() *Log {
	return &Log{
		debug: log.New(os.Stdout, "", log.LstdFlags),
		info:  log.New(os.Stdout, "", log.LstdFlags),
		warn:  log.New(os.Stdout, "", log.LstdFlags),
		error: log.New(os.Stderr, "", log.LstdFlags),
		fatal: log.New(os.Stderr, "", log.LstdFlags),
	}
}

// Default logger instance
var defaultLog = NewLog()

func formatMessage(level, message string) string {
	return fmt.Sprintf("%s [%s] %s", time.Now().Format("2006/01/02 15:04:05"), level, message)
}

// Instance methods
func (l *Log) Debug(msg string)                          { l.debug.Println(formatMessage(LevelDebug, msg)) }
func (l *Log) Debugf(format string, args ...interface{}) { l.Debug(fmt.Sprintf(format, args...)) }
func (l *Log) Info(msg string)                           { l.info.Println(formatMessage(LevelInfo, msg)) }
func (l *Log) Infof(format string, args ...interface{})  { l.Info(fmt.Sprintf(format, args...)) }
func (l *Log) Warn(msg string)                           { l.warn.Println(formatMessage(LevelWarn, msg)) }
func (l *Log) Warnf(format string, args ...interface{})  { l.Warn(fmt.Sprintf(format, args...)) }
func (l *Log) Error(msg string)                          { l.error.Println(formatMessage(LevelError, msg)) }
func (l *Log) Errorf(format string, args ...interface{}) { l.Error(fmt.Sprintf(format, args...)) }
func (l *Log) Fatal(msg string)                          { l.fatal.Println(formatMessage(LevelFatal, msg)); os.Exit(1) }
func (l *Log) Fatalf(format string, args ...interface{}) { l.Fatal(fmt.Sprintf(format, args...)) }

// Package-level functions using default logger
func Debug(msg string)                          { defaultLog.Debug(msg) }
func Debugf(format string, args ...interface{}) { defaultLog.Debugf(format, args...) }
func Info(msg string)                           { defaultLog.Info(msg) }
func Infof(format string, args ...interface{})  { defaultLog.Infof(format, args...) }
func Warn(msg string)                           { defaultLog.Warn(msg) }
func Warnf(format string, args ...interface{})  { defaultLog.Warnf(format, args...) }
func Error(msg string)                          { defaultLog.Error(msg) }
func Errorf(format string, args ...interface{}) { defaultLog.Errorf(format, args...) }
func Fatal(msg string)                          { defaultLog.Fatal(msg) }
func Fatalf(format string, args ...interface{}) { defaultLog.Fatalf(format, args...) }
