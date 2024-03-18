package logger

import (
	"fmt"
	"log"
	"os"
	"strings"
)

const (
	DebugLevel = iota
	InfoLevel
	WarningLevel
	ErrorLevel
)

type Logger struct {
	Level         int
	infoLogger    *log.Logger
	warningLogger *log.Logger
	errorLogger   *log.Logger
	debugLogger   *log.Logger
}

var logger *Logger

func init() {
	red := "\033[31m"
	green := "\033[32m"
	yellow := "\033[33m"
	magenta := "\033[95m"
	reset := "\033[0m"

	logger = &Logger{
		Level:         InfoLevel,
		infoLogger:    log.New(os.Stdout, green+" INFO  | "+reset, log.LstdFlags),
		warningLogger: log.New(os.Stdout, yellow+" WARN  | "+reset, log.LstdFlags),
		errorLogger:   log.New(os.Stdout, red+" ERROR | "+reset, log.LstdFlags),
		debugLogger:   log.New(os.Stdout, magenta+" DEBUG | "+reset, log.LstdFlags),
	}
}

func SetLevel(level int) {
	logger.Level = level
}

func Info(message string) {
	message = strings.Trim(message, "\n")
	if logger.Level <= InfoLevel {
		logger.infoLogger.Println(message)
	}
}

func Infof(format string, a ...any) {
	format = strings.Trim(format, "\n")
	message := fmt.Sprintf(format, a...)

	if logger.Level <= InfoLevel {
		logger.infoLogger.Println(message)
	}
}

func Warning(message string) {
	if logger.Level <= WarningLevel {
		logger.warningLogger.Println(message)
	}
}

func Warningf(format string, a ...any) {
	message := fmt.Sprintf(format, a...)

	if logger.Level <= WarningLevel {
		logger.warningLogger.Println(message)
	}
}

func Error(message string) {
	if logger.Level <= ErrorLevel {
		logger.errorLogger.Println(message)
	}
}

func Errorf(format string, a ...any) {
	message := fmt.Sprintf(format, a...)

	if logger.Level <= ErrorLevel {
		logger.errorLogger.Println(message)
	}
}

func Debug(message string) {
	if logger.Level <= DebugLevel {
		logger.debugLogger.Println(message)
	}
}

func Debugf(format string, a ...any) {
	message := fmt.Sprintf(format, a...)

	if logger.Level <= DebugLevel {
		logger.debugLogger.Println(message)
	}
}
