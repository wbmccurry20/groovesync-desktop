package logging

import (
	"io"
	"log"
	"os"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Init initializes a global logger with log rotation.
func Init() {
	// Ensure the log directory exists
	logDir := "logs"
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		err := os.MkdirAll(logDir, 0755)
		if err != nil {
			log.Fatalf("Failed to create log directory: %v", err)
		}
	}

	// Configure log rotation
	logFile := &lumberjack.Logger{
		Filename:   "logs/groovesync.log", // Logs file path
		MaxSize:    5,                     // Max megabytes before rotation
		MaxBackups: 3,                     // Max number of backups
		MaxAge:     28,                    // Max days to retain
		Compress:   true,                  // Compress old logs
	}

	// Set log output to both the console and the rotating file
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)

	// Set log format to include timestamps and file/line info
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// Log initialization
	log.Println("Logger initialized with rotation")
}
