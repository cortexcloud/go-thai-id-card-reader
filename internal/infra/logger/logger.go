package logger

import (
	"log"
	"os"
	"path/filepath"

	"github.com/cortex-x/go-thai-id-card-reader/internal/config"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
	DataLogger  *log.Logger
)

// Setup initializes the logging system with file rotation
func Setup(cfg *config.LogConfig) error {
	// Get the directory of the executable
	execPath, err := os.Executable()
	if err != nil {
		return err
	}
	execDir := filepath.Dir(execPath)
	
	// Create log file path relative to executable
	logFile := filepath.Join(execDir, cfg.File)
	
	// Ensure log directory exists
	logDir := filepath.Dir(logFile)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	// Setup rotating file writer
	fileWriter := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    cfg.MaxSize, // megabytes
		MaxAge:     cfg.MaxAge,  // days
		Compress:   cfg.Compress,
		LocalTime:  true,
	}

	// All logs go to file only
	InfoLogger = log.New(fileWriter, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(fileWriter, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	DataLogger = log.New(fileWriter, "DATA: ", log.Ldate|log.Ltime)

	return nil
}

// LogData logs raw data strings - simple and reusable
func LogData(data string) {
	if DataLogger != nil {
		DataLogger.Println(data)
	}
}

// Info logs info messages
func Info(v ...interface{}) {
	if InfoLogger != nil {
		InfoLogger.Println(v...)
	}
}

// Infof logs formatted info messages
func Infof(format string, v ...interface{}) {
	if InfoLogger != nil {
		InfoLogger.Printf(format, v...)
	}
}

// Error logs error messages
func Error(v ...interface{}) {
	if ErrorLogger != nil {
		ErrorLogger.Println(v...)
	}
}

// Errorf logs formatted error messages
func Errorf(format string, v ...interface{}) {
	if ErrorLogger != nil {
		ErrorLogger.Printf(format, v...)
	}
}