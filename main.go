package main

import (
	"context"
	"embed"
	"fmt"
	"groovesync/internal/downloader"
	"groovesync/internal/logging"
	"log"
	"os"
	"path/filepath"
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
		Width:  1200, // Increased for new dashboard layout
		Height: 800,  // Increased for new dashboard layout
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
	CreatedAt  string  `json:"createdAt"` // Changed to string
	Error      string  `json:"error,omitempty"`
	Type       string  `json:"type"` // "single", "playlist", "batch"
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
	return &App{
		downloads: make(map[string]*DownloadJob),
		settings: &AppSettings{
			DefaultFormat:    "wav",
			DefaultQuality:   "high",
			DefaultOutputDir: "./downloads",
			MaxConcurrent:    4,
			Theme:            "dark",
		},
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) StartDownload(url, name, format, dir string) error {
	if dir == "" {
		dir = a.settings.DefaultOutputDir // Use the default from settings
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Create a download job
	jobID := a.CreateDownloadJob(url, name, format, dir, "single")

	// Run the download in a goroutine so it doesn't block
	go func() {
		a.mu.Lock()
		job := a.downloads[jobID]
		job.Status = "downloading"
		a.mu.Unlock()
		runtime.EventsEmit(a.ctx, "downloadUpdated", job)

		err := downloader.RunYTDLPParallel(
			url, dir, name, format,
			func(status string) {
				a.mu.Lock()
				job.Status = status
				a.mu.Unlock()
				runtime.EventsEmit(a.ctx, "downloadUpdated", job)
			},
			func(current, total int) {
				a.mu.Lock()
				job.Progress = float64(current) / float64(total) * 100
				a.mu.Unlock()
				runtime.EventsEmit(a.ctx, "downloadUpdated", job)
			},
			func(tracks []downloader.Track) {
				runtime.EventsEmit(a.ctx, "tracks", tracks)
			},
		)

		if err != nil {
			a.mu.Lock()
			job.Status = "failed"
			job.Error = err.Error()
			a.mu.Unlock()
			runtime.EventsEmit(a.ctx, "downloadUpdated", job)
		}
	}()

	return nil
}

func (a *App) ExportToRekordbox(playlistName string) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Find the most recent completed job with the given playlist name
	var latestJob *DownloadJob
	var latestTime time.Time
	for _, job := range a.downloads {
		if job.Title == playlistName && job.Status == "completed" {
			// Parse the CreatedAt string back to time.Time for comparison
			jobTime, err := time.Parse(time.RFC3339, job.CreatedAt)
			if err != nil {
				continue // Skip if parsing fails
			}
			if latestJob == nil || jobTime.After(latestTime) {
				latestJob = job
				latestTime = jobTime
			}
		}
	}

	if latestJob == nil {
		return fmt.Errorf("no completed download found for playlist: %s", playlistName)
	}

	// Call the downloader's export function
	playlistDir := filepath.Join(latestJob.OutputPath, playlistName)
	if err := downloader.ExportToRekordbox(playlistDir, playlistName); err != nil {
		return fmt.Errorf("failed to export to Rekordbox: %w", err)
	}

	// Emit the export path to the frontend
	exportPath := filepath.Join(playlistDir, playlistName+".xml")
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
		runtime.EventsEmit(a.ctx, "downloadCancelled", id)
		return nil
	}
	return fmt.Errorf("download not found")
}

func (a *App) GetSupportedFormats() []string {
	return []string{"wav", "mp3", "aac", "flac", "opus"}
}

func (a *App) ValidateURL(url string) bool {
	// Basic URL validation - can be enhanced
	return url != "" && (len(url) > 10)
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
		CreatedAt:  time.Now().Format(time.RFC3339), // Format as string
		Type:       jobType,
	}

	a.downloads[id] = job
	runtime.EventsEmit(a.ctx, "downloadCreated", job)
	return id
}

func (a *App) GetDownloadStats() map[string]interface{} {
	a.mu.RLock()
	defer a.mu.RUnlock()

	stats := map[string]interface{}{
		"total":     len(a.downloads),
		"active":    0,
		"completed": 0,
		"failed":    0,
		"cancelled": 0,
	}

	for _, job := range a.downloads {
		switch job.Status {
		case "downloading", "queued":
			stats["active"] = stats["active"].(int) + 1
		case "completed":
			stats["completed"] = stats["completed"].(int) + 1
		case "failed":
			stats["failed"] = stats["failed"].(int) + 1
		case "cancelled":
			stats["cancelled"] = stats["cancelled"].(int) + 1
		}
	}

	return stats
}
