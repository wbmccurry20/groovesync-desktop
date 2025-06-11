package downloader

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
)

// Track represents a single track with metadata.
type Track struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

// getBinaryPaths locates both yt-dlp and ffmpeg binaries based on the OS and environment.
func getBinaryPaths() (ytdlpPath, ffmpegPath string, err error) {
	var ytdlpBinaryName, ffmpegBinaryName string

	// Determine binary names based on OS
	switch runtime.GOOS {
	case "darwin":
		ytdlpBinaryName = "yt-dlp_macos"
		ffmpegBinaryName = "ffmpeg"
	case "linux":
		ytdlpBinaryName = "yt-dlp"
		ffmpegBinaryName = "ffmpeg"
	case "windows":
		ytdlpBinaryName = "yt-dlp.exe"
		ffmpegBinaryName = "ffmpeg.exe"
	default:
		return "", "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	// Try the directory of the running executable (for bundled app)
	execPath, err := os.Executable()
	if err == nil {
		execDir := filepath.Dir(execPath)
		// On macOS, if inside .app bundle, adjust the path to Contents/Resources/
		if runtime.GOOS == "darwin" {
			execDir = filepath.Join(execDir, "..", "Resources")
		}
		// Try bin/ subdirectory
		binDir := filepath.Join(execDir, "bin")
		ytdlpPath = filepath.Join(binDir, ytdlpBinaryName)
		ffmpegPath = filepath.Join(binDir, ffmpegBinaryName)

		// Check if both binaries exist
		if _, err := os.Stat(ytdlpPath); err == nil {
			if _, err := os.Stat(ffmpegPath); err == nil {
				return ytdlpPath, ffmpegPath, nil
			}
		}
	}

	// Fallback: try the bin/ directory in the project root (for development)
	baseDir, err := os.Getwd()
	if err != nil {
		return "", "", fmt.Errorf("failed to get working directory: %v", err)
	}

	binDir := filepath.Join(baseDir, "bin")
	ytdlpPath = filepath.Join(binDir, ytdlpBinaryName)
	ffmpegPath = filepath.Join(binDir, ffmpegBinaryName)

	// Check if both binaries exist
	if _, err := os.Stat(ytdlpPath); err != nil {
		return "", "", fmt.Errorf("yt-dlp binary not found at %s: %v", ytdlpPath, err)
	}
	if _, err := os.Stat(ffmpegPath); err != nil {
		return "", "", fmt.Errorf("ffmpeg binary not found at %s: %v", ffmpegPath, err)
	}

	return ytdlpPath, ffmpegPath, nil
}

// RunYTDLPParallel downloads tracks from a playlist URL in parallel.
func RunYTDLPParallel(
	playlistURL, downloadDir, playlistName, userFormat string,
	updateStatus func(string),
	updateProgress func(current, total int),
	updateTracks func([]Track),
) error {
	ytdlpPath, _, err := getBinaryPaths()
	if err != nil {
		updateStatus("Error: Failed to locate yt-dlp binary")
		log.Printf("Error locating yt-dlp binary: %v", err)
		return err
	}
	log.Printf("Using yt-dlp binary: %s", ytdlpPath)

	if userFormat == "" {
		userFormat = "wav"
	}

	playlistDir := filepath.Join(downloadDir, playlistName)
	if err := os.MkdirAll(playlistDir, 0755); err != nil {
		updateStatus("Error: Failed to create playlist directory")
		log.Printf("Error creating playlist directory: %v", err)
		return err
	}

	// Extract track URLs and metadata
	trackURLs, playlistTitle, tracks, err := extractTrackURLs(ytdlpPath, playlistURL)
	if err != nil {
		updateStatus("Error: Failed to extract playlist tracks")
		log.Printf("Error extracting tracks: %v", err)
		return err
	}
	if len(trackURLs) == 0 {
		updateStatus("No tracks found in playlist")
		log.Printf("No tracks found for playlist: %s", playlistURL)
		return fmt.Errorf("no tracks found in playlist")
	}
	log.Printf("Found %d tracks in playlist %s", len(trackURLs), playlistName)

	// Auto-fill playlist name if not provided
	if playlistName == "" {
		playlistName = playlistTitle
	}
	updateTracks(tracks)
	updateStatus("Starting downloads...")
	updateProgress(0, len(trackURLs))

	// Dynamic concurrency based on CPU cores
	maxConcurrentDownloads := runtime.NumCPU() * 2
	if maxConcurrentDownloads < 1 {
		maxConcurrentDownloads = 1
	}
	semaphore := make(chan struct{}, maxConcurrentDownloads)
	var wg sync.WaitGroup
	var failedTracks []string
	var mu sync.Mutex

	for idx, track := range tracks {
		wg.Add(1)
		semaphore <- struct{}{}
		go func(track Track, trackIdx int) {
			defer wg.Done()
			defer func() { <-semaphore }()

			updateStatus(fmt.Sprintf("Downloading Track %d/%d: %s", trackIdx+1, len(trackURLs), track.Title))
			log.Printf("Downloading track %d/%d: %s", trackIdx+1, len(trackURLs), track.URL)

			err := downloadTrackWithRetry(ytdlpPath, track.URL, playlistDir, []string{userFormat, "wav", "opus", "mp3"}, func(percent float64) {
				// Update progress as a percentage for the current track
				overallProgress := (float64(trackIdx) + (percent / 100.0)) / float64(len(trackURLs)) * 100
				updateProgress(int(overallProgress*float64(len(trackURLs))/100), len(trackURLs))
			})
			if err != nil {
				log.Printf("Error downloading track %d: %s", trackIdx+1, err)
				mu.Lock()
				failedTracks = append(failedTracks, fmt.Sprintf("Track %d: %s (Error: %v)", trackIdx+1, track.URL, err))
				mu.Unlock()
			}

			updateProgress(trackIdx+1, len(trackURLs))
		}(track, idx)
	}

	wg.Wait()
	close(semaphore)

	// Removed automatic export to Rekordbox
	// Now handled manually via the UI button

	// Final status update
	if len(failedTracks) > 0 {
		updateStatus(fmt.Sprintf("Downloaded with %d failures. Check logs.", len(failedTracks)))
		log.Printf("Failed to download the following tracks: %v", failedTracks)
		return fmt.Errorf("some tracks failed to download")
	}

	updateStatus(fmt.Sprintf("All tracks for playlist %s downloaded successfully!", playlistName))
	log.Printf("All tracks for playlist %s downloaded successfully.", playlistName)
	return nil
}

// extractTrackURLs uses yt-dlp to list track URLs and metadata without downloading.
func extractTrackURLs(binary, playlistURL string) ([]string, string, []Track, error) {
	cmd := exec.Command(binary, "--flat-playlist", "--no-warnings", "-J", playlistURL)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("yt-dlp error: %v\nOutput: %s", err, string(output))
		return nil, "", nil, err
	}

	type entry struct {
		URL   string `json:"url"`
		Title string `json:"title"`
	}
	type playlistData struct {
		Entries []entry `json:"entries"`
		Title   string  `json:"title"`
	}

	var data playlistData
	if err := json.Unmarshal(output, &data); err != nil {
		return nil, "", nil, err
	}

	var urls []string
	var tracks []Track
	for _, e := range data.Entries {
		if e.URL != "" {
			urls = append(urls, e.URL)
			tracks = append(tracks, Track{URL: e.URL, Title: e.Title})
		}
	}
	return urls, data.Title, tracks, nil
}

// downloadTrackWithRetry attempts to download a track with retries.
func downloadTrackWithRetry(binary, url, playlistDir string, formats []string, progressCb func(float64)) error {
	operation := func() error {
		return downloadTrackWithFallback(binary, url, playlistDir, formats, progressCb)
	}
	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 5 * time.Minute
	return backoff.Retry(operation, bo)
}

// downloadTrackWithFallback tries each format until one succeeds, with progress tracking.
func downloadTrackWithFallback(binary, url, playlistDir string, formats []string, progressCb func(float64)) error {
	log.Printf("Starting download for track: %s", url)
	defer log.Printf("Finished attempt for track: %s", url)

	ytdlpPath, ffmpegPath, err := getBinaryPaths()
	if err != nil {
		return fmt.Errorf("failed to locate binaries: %v", err)
	}

	// Ensure binaries are executable (non-Windows)
	if runtime.GOOS != "windows" {
		if err := os.Chmod(ytdlpPath, 0755); err != nil {
			return fmt.Errorf("failed to make yt-dlp executable: %v", err)
		}
		if err := os.Chmod(ffmpegPath, 0755); err != nil {
			return fmt.Errorf("failed to make ffmpeg executable: %v", err)
		}
	}

	// Regex to parse yt-dlp progress (e.g., "[download]  50.0% of 10.00MiB")
	progressRegex := regexp.MustCompile(`\[download\]\s+(\d+\.\d+)%`)

	for _, f := range formats {
		log.Printf("Attempting %s format for track: %s", f, url)
		opts := []string{
			"-x",
			"--audio-format", f,
			"--audio-quality", "0",
			"--ffmpeg-location", filepath.Dir(ffmpegPath),
			"-o", filepath.Join(playlistDir, "%(title)s.%(ext)s"),
			url,
		}

		cmd := exec.Command(ytdlpPath, opts...)
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return fmt.Errorf("failed to pipe stderr: %v", err)
		}

		if err := cmd.Start(); err != nil {
			return fmt.Errorf("failed to start yt-dlp: %v", err)
		}

		// Read stderr to parse progress
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			log.Printf("yt-dlp output: %s", line)
			if matches := progressRegex.FindStringSubmatch(line); len(matches) > 1 {
				if percent, err := strconv.ParseFloat(matches[1], 64); err == nil {
					progressCb(percent)
				}
			}
		}

		if err := cmd.Wait(); err != nil {
			log.Printf("Failed %s for track %s: %v", f, url, err)
			continue
		}

		log.Printf("Track downloaded successfully in %s format: %s", f, url)
		progressCb(100.0) // Ensure 100% is reported on completion
		return nil
	}
	return fmt.Errorf("all format attempts failed for track: %s", url)
}

// ExportToRekordbox generates a Rekordbox-compatible XML file.
func ExportToRekordbox(playlistDir, playlistName string) error {
	tracks, err := os.ReadDir(playlistDir)
	if err != nil {
		return fmt.Errorf("failed to read playlist directory: %v", err)
	}

	// Filter out non-audio files and collect track paths
	var audioTracks []string
	for _, track := range tracks {
		if track.IsDir() {
			continue
		}
		ext := filepath.Ext(track.Name())
		if ext == ".wav" || ext == ".mp3" || ext == ".aac" || ext == ".flac" || ext == ".opus" {
			audioTracks = append(audioTracks, filepath.Join(playlistDir, track.Name()))
		}
	}

	if len(audioTracks) == 0 {
		return fmt.Errorf("no audio tracks found in directory: %s", playlistDir)
	}

	// Generate Rekordbox XML
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
    <DJ_PLAYLISTS Version="1.0.0">
        <PRODUCT Name="rekordbox" Version="6.0.0" Company="Pioneer DJ"/>
        <COLLECTION Entries="` + fmt.Sprintf("%d", len(audioTracks)) + `">
    `

	for i, trackPath := range audioTracks {
		trackName := filepath.Base(trackPath)
		trackName = trackName[:len(trackName)-len(filepath.Ext(trackName))] // Remove extension
		// Rekordbox expects the Location to be a file:// URI with absolute path
		absPath, err := filepath.Abs(trackPath)
		if err != nil {
			return fmt.Errorf("failed to get absolute path for track %s: %v", trackPath, err)
		}
		location := "file://localhost" + absPath
		if runtime.GOOS == "windows" {
			location = "file://localhost/" + strings.ReplaceAll(absPath, `\`, `/`)
		}
		xmlContent += fmt.Sprintf(
			`<TRACK TrackID="%d" Name="%s" Location="%s"/>`,
			i+1, trackName, location,
		)
	}

	xmlContent += `
        </COLLECTION>
        <PLAYLISTS>
            <NODE Type="0" Name="ROOT" Count="1">
                <NODE Type="1" Name="` + playlistName + `" Count="` + fmt.Sprintf("%d", len(audioTracks)) + `">
    `

	for i := range audioTracks {
		xmlContent += fmt.Sprintf(`<TRACK Key="%d"/>`, i+1)
	}

	xmlContent += `
                </NODE>
            </NODE>
        </PLAYLISTS>
    </DJ_PLAYLISTS>`

	xmlPath := filepath.Join(playlistDir, playlistName+".xml")
	if err := os.WriteFile(xmlPath, []byte(xmlContent), 0644); err != nil {
		return fmt.Errorf("failed to write Rekordbox XML: %v", err)
	}

	return nil
}
