// groovesync/main.go
package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"groovesync/internal/downloader"
	"groovesync/internal/logging"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed frontend/dist/*
var assets embed.FS

func main() {
	logging.Init()

	app := NewApp()

	err := wails.Run(&options.App{
		Title:  "GrooveSync",
		Width:  1200,
		Height: 800,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		log.Fatal(err)
	}
}

type App struct {
	ctx       context.Context
	downloads map[string]*DownloadJob
	settings  *AppSettings
	mu        sync.RWMutex
}

type DownloadJob struct {
	ID         string  `json:"id"`
	URL        string  `json:"url"`
	Title      string  `json:"title"`
	Status     string  `json:"status"`
	Progress   float64 `json:"progress"`
	Format     string  `json:"format"`
	OutputPath string  `json:"outputPath"`
	CreatedAt  string  `json:"createdAt"`
	Error      string  `json:"error,omitempty"`
	Type       string  `json:"type"`
}

type AppSettings struct {
	DefaultFormat    string `json:"defaultFormat"`
	DefaultQuality   string `json:"defaultQuality"`
	DefaultOutputDir string `json:"defaultOutputDir"`
	MaxConcurrent    int    `json:"maxConcurrent"`
	Theme            string `json:"theme"`
}

type Track struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

func NewApp() *App {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Failed to get user home dir: %v, falling back to current dir", err)
		homeDir = "."
	}
	defaultOut := filepath.Join(homeDir, "Downloads", "GrooveSync")
	if err := os.MkdirAll(defaultOut, 0755); err != nil {
		log.Printf("Failed to create default output dir: %v", err)
	}

	return &App{
		downloads: make(map[string]*DownloadJob),
		settings: &AppSettings{
			DefaultFormat:    "wav",
			DefaultQuality:   "high",
			DefaultOutputDir: defaultOut,
			MaxConcurrent:    4,
			Theme:            "dark",
		},
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	runtime.EventsEmit(a.ctx, "settingsUpdated", a.GetSettings())
	runtime.EventsEmit(a.ctx, "downloadStatsUpdated", a.GetDownloadStats())
}

func (a *App) StartDownload(url, name, format, dir string) error {
	logDir := dir
	if dir == "" {
		logDir = "(default)"
	}
	log.Printf("Starting download: URL=%s, Name=%s, Format=%s, Dir=%s", url, name, format, logDir)

	if !a.ValidateURL(url) {
		err := fmt.Errorf("invalid URL: %s", url)
		log.Printf("Validation failed: %v", err)
		return err
	}
	if !contains(a.GetSupportedFormats(), format) {
		err := fmt.Errorf("unsupported format: %s", format)
		log.Printf("Validation failed: %v", err)
		return err
	}
	if dir == "" {
		dir = a.settings.DefaultOutputDir
		log.Printf("Using default output directory: %s", dir)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("Failed to create directory %s: %v", dir, err)
		return err
	}

	// Fetch playlist metadata using the bundled yt-dlp binary.
	ytDLP, err := downloader.GetYTDLPBinaryPath()
	if err != nil {
		log.Printf("Failed to locate yt-dlp binary: %v", err)
		return fmt.Errorf("failed to locate yt-dlp binary: %w", err)
	}
	log.Printf("Using yt-dlp binary: %s", ytDLP)
	cmdArgs := []string{"--flat-playlist", "--no-warnings", "-J", url}
	cookiesPath := filepath.Join(os.Getenv("HOME"), "cookies.txt")
	if _, err := os.Stat(cookiesPath); err == nil {
		cmdArgs = append([]string{"--cookies", cookiesPath}, cmdArgs...)
		log.Printf("Using cookies file: %s", cookiesPath)
	}
	cmd := exec.Command(ytDLP, cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Failed to fetch playlist metadata: %v\nOutput: %s", err, string(output))
		return fmt.Errorf("failed to fetch playlist: %v, output: %s", err, string(output))
	}

	// Log the raw yt-dlp output for debugging
	log.Printf("yt-dlp raw output: %s", string(output))

	// Parse JSON output into tracks (title optional, fallback to ID)
	type entry struct {
		ID    string `json:"id"`
		URL   string `json:"url"`
		Title string `json:"title"`
	}
	type playlistData struct {
		Entries []entry `json:"entries"`
	}
	var data playlistData
	if err := json.Unmarshal(output, &data); err != nil {
		log.Printf("Failed to parse playlist JSON: %v", err)
		return fmt.Errorf("failed to parse playlist JSON: %v", err)
	}

	var tracks []Track
	for _, e := range data.Entries {
		if e.URL != "" {
			title := e.Title
			if title == "" {
				title = "Track " + e.ID
			}
			tracks = append(tracks, Track{URL: e.URL, Title: title})
		}
	}

	log.Printf("Emitting %d tracks to frontend", len(tracks))
	runtime.EventsEmit(a.ctx, "tracks", tracks)

	// Create output directory for playlist
	outputDir := filepath.Join(dir, name)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Printf("Failed to create output directory %s: %v", outputDir, err)
		return err
	}

	// Create a download job
	jobID := a.CreateDownloadJob(url, name, format, outputDir, "playlist")
	log.Printf("Created download job ID: %s", jobID)

	// Run the download in a goroutine with panic recovery
	go func() {
		log.Printf("Goroutine started for job %s", jobID)
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Panic in download goroutine for job %s: %v", jobID, r)
				a.mu.Lock()
				if job, exists := a.downloads[jobID]; exists {
					job.Status = "failed"
					job.Error = fmt.Sprintf("Unexpected error: %v", r)
					runtime.EventsEmit(a.ctx, "downloadUpdated", job)
				}
				a.mu.Unlock()
				runtime.EventsEmit(a.ctx, "downloadStatsUpdated", a.GetDownloadStats())
			}
		}()

		a.mu.Lock()
		job, exists := a.downloads[jobID]
		if !exists {
			log.Printf("Job %s not found after creation", jobID)
			a.mu.Unlock()
			return
		}
		job.Status = "downloading"
		a.mu.Unlock()
		runtime.EventsEmit(a.ctx, "downloadUpdated", job)

		log.Printf("Calling RunYTDLPParallel for job %s", jobID)
		err := downloader.RunYTDLPParallel(
			url,
			dir,
			name,
			format,
			func(status string) {
				a.mu.Lock()
				job.Status = status
				a.mu.Unlock()
				log.Printf("Status update for job %s: %s", jobID, status)
				runtime.EventsEmit(a.ctx, "downloadUpdated", job)
			},
			func(current, total int) {
				a.mu.Lock()
				if total > 0 {
					job.Progress = float64(current) / float64(total) * 100
				}
				a.mu.Unlock()
				log.Printf("Progress update for job %s: %d/%d (%.2f%%)", jobID, current, total, job.Progress)
				runtime.EventsEmit(a.ctx, "downloadUpdated", job)
			},
		)
		log.Printf("RunYTDLPParallel completed for job %s", jobID)

		a.mu.Lock()
		if err != nil {
			log.Printf("Download failed for job %s: %v", jobID, err)
			job.Status = "failed"
			job.Error = err.Error()
		} else {
			log.Printf("Download completed for job %s", jobID)
			job.Status = "completed"
		}
		a.mu.Unlock()
		runtime.EventsEmit(a.ctx, "downloadUpdated", job)
		runtime.EventsEmit(a.ctx, "downloadStatsUpdated", a.GetDownloadStats())
	}()

	return nil
}

func (a *App) ExportToRekordbox(playlistName string) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var latestJob *DownloadJob
	var latestTime time.Time
	for _, job := range a.downloads {
		if job.Title == playlistName && job.Status == "completed" {
			jobTime, err := time.Parse(time.RFC3339, job.CreatedAt)
			if err != nil {
				log.Printf("Failed to parse CreatedAt for job %s: %v", job.ID, err)
				continue
			}
			if latestJob == nil || jobTime.After(latestTime) {
				latestJob = job
				latestTime = jobTime
			}
		}
	}

	if latestJob == nil {
		log.Printf("No completed download found for playlist: %s", playlistName)
		return fmt.Errorf("no completed download found for playlist: %s", playlistName)
	}

	playlistDir := latestJob.OutputPath
	if err := downloader.ExportToRekordbox(playlistDir, playlistName, latestJob.Format); err != nil {
		log.Printf("Failed to export to Rekordbox for %s: %v", playlistName, err)
		return fmt.Errorf("failed to export to Rekordbox: %w", err)
	}

	exportPath := filepath.Join(playlistDir, playlistName+".xml")
	log.Printf("Export completed for %s: %s", playlistName, exportPath)
	runtime.EventsEmit(a.ctx, "exportCompleted", map[string]string{"path": exportPath})
	return nil
}

func (a *App) GetActiveDownloads() []*DownloadJob {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var active []*DownloadJob
	for _, job := range a.downloads {
		if job.Status == "downloading" || job.Status == "queued" {
			active = append(active, job)
		}
	}
	return active
}

func (a *App) GetAllDownloads() []*DownloadJob {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var all []*DownloadJob
	for _, job := range a.downloads {
		all = append(all, job)
	}
	return all
}

func (a *App) CancelDownload(id string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if job, exists := a.downloads[id]; exists {
		job.Status = "cancelled"
		log.Printf("Cancelled download job %s", id)
		runtime.EventsEmit(a.ctx, "downloadCancelled", id)
		runtime.EventsEmit(a.ctx, "downloadStatsUpdated", a.GetDownloadStats())
		return nil
	}
	log.Printf("Failed to cancel download: job %s not found", id)
	return fmt.Errorf("download not found")
}

func (a *App) GetSupportedFormats() []string {
	return []string{"wav", "mp3", "aac", "flac", "opus"}
}

func (a *App) ValidateURL(url string) bool {
	if url == "" || len(url) < 10 {
		return false
	}
	if !(strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")) {
		return false
	}
	return !strings.Contains(url, " ")
}

func (a *App) GetSettings() *AppSettings {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.settings
}

func (a *App) UpdateSettings(settings *AppSettings) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.settings = settings
	log.Printf("Updated settings: %+v", settings)
	runtime.EventsEmit(a.ctx, "settingsUpdated", settings)
	return nil
}

func (a *App) CreateDownloadJob(url, title, format, outputPath, jobType string) string {
	a.mu.Lock()
	defer a.mu.Unlock()

	id := fmt.Sprintf("job_%d", time.Now().UnixNano())
	job := &DownloadJob{
		ID:         id,
		URL:        url,
		Title:      title,
		Status:     "queued",
		Progress:   0,
		Format:     format,
		OutputPath: outputPath,
		CreatedAt:  time.Now().Format(time.RFC3339),
		Type:       jobType,
	}

	a.downloads[id] = job
	log.Printf("Created download job: %+v", job)
	runtime.EventsEmit(a.ctx, "downloadCreated", job)
	runtime.EventsEmit(a.ctx, "downloadStatsUpdated", a.GetDownloadStats())
	return id
}

func (a *App) GetDownloadStats() map[string]int {
	a.mu.RLock()
	defer a.mu.RUnlock()

	stats := map[string]int{
		"total":     len(a.downloads),
		"active":    0,
		"completed": 0,
		"failed":    0,
		"cancelled": 0,
	}

	for _, job := range a.downloads {
		switch job.Status {
		case "downloading", "queued":
			stats["active"]++
		case "completed":
			stats["completed"]++
		case "failed":
			stats["failed"]++
		case "cancelled":
			stats["cancelled"]++
		}
	}

	return stats
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
