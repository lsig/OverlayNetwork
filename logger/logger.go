package logger

import (
	"log"
	"os"
)

const (
	InfoLevel = iota
	WarningLevel
	ErrorLevel
)

type Logger struct {
	Level         int
	infoLogger    *log.Logger
	warningLogger *log.Logger
	errorLogger   *log.Logger
}

var logger *Logger

func init() {
	red := "\033[31m"
	green := "\033[32m"
	yellow := "\033[33m"
	reset := "\033[0m"

	logger = &Logger{
		Level:         InfoLevel,
		infoLogger:    log.New(os.Stdout, green+" INFO  | "+reset, log.LstdFlags),
		warningLogger: log.New(os.Stdout, yellow+" WARN  | "+reset, log.LstdFlags),
		errorLogger:   log.New(os.Stdout, red+" ERROR | "+reset, log.LstdFlags),
	}
}

func SetLevel(level int) {
	logger.Level = level
}

func Info(message string) {
	if logger.Level <= InfoLevel {
		logger.infoLogger.Println(message)
	}
}

func Warning(message string) {
	if logger.Level <= WarningLevel {
		logger.warningLogger.Println(message)
	}
}

func Error(message string) {
	if logger.Level <= ErrorLevel {
		logger.errorLogger.Println(message)
	}
}
