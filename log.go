package workerreloader

import (
	"log"
)

const (
	logLevelSilent = iota + 1 // 1
	logLevelError             // 2
	logLevelWarn              // 3
	logLevelInfo              // 4
)
const (
	LogLevelNameSilent = "silent"
	LogLevelNameError  = "error"
	LogLevelNameWarn   = "warn"
	LogLevelNameInfo   = "info"
)

var currentLogLevel = logLevelSilent

// SetLogLevel .
func SetLogLevel(level string) {
	switch level {
	case LogLevelNameError:
		currentLogLevel = logLevelError
	case LogLevelNameWarn:
		currentLogLevel = logLevelWarn
	case LogLevelNameInfo:
		currentLogLevel = logLevelInfo
	case LogLevelNameSilent:
		currentLogLevel = logLevelSilent
	default:
		log.Printf("[WARN] Unknown log level '%s', keep the current level %d", level, currentLogLevel)
	}
}

// logError
func logError(format string, v ...interface{}) {
	if currentLogLevel >= logLevelError {
		log.Printf("[ERROR] "+format, v...)
	}
}

// logWarn
func logWarn(format string, v ...interface{}) {
	if currentLogLevel >= logLevelWarn {
		log.Printf("[WARN] "+format, v...)
	}
}

// logInfo
func logInfo(format string, v ...interface{}) {
	if currentLogLevel >= logLevelInfo {
		log.Printf("[INFO] "+format, v...)
	}
}
