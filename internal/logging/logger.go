package logging

import (
	"io"
	"log"
	"os"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Init initializes a global logger with log rotation.
func Init() {
	// Ensure log directory exists
	logDir := "logs"
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		if err := os.MkdirAll(logDir, 0755); err != nil {
			log.Printf("WARNING: Failed to create log directory: %v. Falling back to current directory.", err)
			logDir = "." // Fallback to current directory
		}
	}

	// Configure log rotation
	logFile := &lumberjack.Logger{
		Filename:   logDir + "/groovesync.log", // Log file location
		MaxSize:    5,                          // Max megabytes before rotation
		MaxBackups: 3,                          // Max number of old logs to retain
		MaxAge:     28,                         // Max days to retain old logs
		Compress:   true,                       // Compress old logs
	}

	// Set log output to both console and rotating file
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)

	// Set log format to include timestamps and file:line numbers
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// Initial log entry
	log.Println("Logger initialized with rotation")
}
