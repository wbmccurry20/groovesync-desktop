// groovesync/internal/logging/logging.go
package logging

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
)

// Init configures logging to write to both stdout (visible when run from a
// terminal) and a persistent log file. When the app is launched as a .app
// bundle, stdout is discarded, so the file is the only way to inspect logs.
func Init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	logPath := logFilePath()
	if logPath != "" {
		if err := os.MkdirAll(filepath.Dir(logPath), 0755); err == nil {
			if f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
				log.SetOutput(io.MultiWriter(os.Stdout, f))
				log.Printf("Logging to %s", logPath)
				return
			}
		}
	}

	// Fallback: stdout only.
	log.SetOutput(os.Stdout)
}

// logFilePath returns a per-user, discoverable location for the log file.
func logFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "groovesync.log")
	}
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Logs", "GrooveSync", "groovesync.log")
	}
	return filepath.Join(home, ".groovesync", "groovesync.log")
}
