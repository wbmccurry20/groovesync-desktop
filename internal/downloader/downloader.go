package downloader

import (
    "encoding/json"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "runtime"
    "log"
    "sync"
)

// getYTDLPBinaryPath determines the path to the yt-dlp binary based on the OS and architecture.
func getYTDLPBinaryPath() (string, error) {
    // Check if running inside the .app bundle
    execPath, err := os.Executable()
    if err == nil {
        // Get the directory of the executable (e.g., GrooveSync.app/Contents/MacOS/)
        bundlePath := filepath.Dir(execPath)
        // Look for yt-dlp_macos in Contents/MacOS/bin/
        binaryPath := filepath.Join(bundlePath, "bin", "yt-dlp_macos")
        if _, err := os.Stat(binaryPath); err == nil {
            // Binary found inside the .app bundle
            return binaryPath, nil
        }
    }

    // Fall back to the project root's bin/ directory for development mode
    var baseDir string
    if execPath, err := os.Executable(); err == nil {
        // Navigate up to the project root (assuming executable is in project root during dev)
        baseDir = filepath.Dir(filepath.Dir(filepath.Dir(execPath)))
    } else {
        // Fallback to current working directory (less reliable)
        baseDir, err = os.Getwd()
        if err != nil {
            return "", fmt.Errorf("failed to get working directory: %v", err)
        }
    }

    var binaryName string
    switch runtime.GOOS {
    case "darwin":
        binaryName = "yt-dlp_macos"
    case "linux":
        binaryName = "yt-dlp_linux"
    case "windows":
        binaryName = "yt-dlp.exe"
    default:
        return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
    }

    binaryPath := filepath.Join(baseDir, "bin", binaryName)
    if _, err := os.Stat(binaryPath); err != nil {
        return "", fmt.Errorf("yt-dlp binary not found at %s: %v", binaryPath, err)
    }

    return binaryPath, nil
}

// RunYTDLPParallel downloads tracks from a playlist URL in parallel.
func RunYTDLPParallel(
    playlistURL, downloadDir, playlistName, userFormat string,
    updateStatus func(string),
    updateProgress func(current, total int),
) error {
    ytDlpBinary, err := getYTDLPBinaryPath()
    if err != nil {
        updateStatus("Error: Failed to locate yt-dlp binary")
        log.Printf("Error locating yt-dlp binary: %v", err)
        return err
    }
    log.Printf("Using yt-dlp binary: %s", ytDlpBinary)

    if userFormat == "" {
        userFormat = "wav"
    }

    playlistDir := filepath.Join(downloadDir, playlistName)
    log.Printf("Using playlist directory: %s", playlistDir)

    // Extract track URLs
    trackURLs, err := extractTrackURLs(ytDlpBinary, playlistURL)
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

    updateStatus("Starting downloads...")
    updateProgress(0, len(trackURLs))

    maxConcurrentDownloads := 3
    semaphore := make(chan struct{}, maxConcurrentDownloads)
    var wg sync.WaitGroup
    var failedTracks []string
    var mu sync.Mutex

    for idx, url := range trackURLs {
        wg.Add(1)
        semaphore <- struct{}{}
        go func(trackURL string, trackIdx int) {
            defer wg.Done()
            defer func() { <-semaphore }()

            log.Printf("Downloading track %d/%d: %s", trackIdx+1, len(trackURLs), trackURL)

            err := downloadTrackWithFallback(ytDlpBinary, trackURL, playlistDir, []string{userFormat, "wav", "opus", "mp3"})
            if err != nil {
                log.Printf("Error downloading track %d: %s", trackIdx+1, err)
                mu.Lock()
                failedTracks = append(failedTracks, fmt.Sprintf("Track %d: %s (Error: %v)", trackIdx+1, trackURL, err))
                mu.Unlock()
            }

            updateProgress(trackIdx+1, len(trackURLs))
        }(url, idx)
    }

    wg.Wait()
    close(semaphore)

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

// extractTrackURLs uses yt-dlp to list track URLs without downloading.
func extractTrackURLs(binary, playlistURL string) ([]string, error) {
    cmd := exec.Command(binary, "--flat-playlist", "--no-warnings", "-J", playlistURL)
    output, err := cmd.CombinedOutput()
    if err != nil {
        log.Printf("yt-dlp error: %v\nOutput: %s", err, string(output))
        return nil, err
    }
    return parseURLsFromJSON(string(output))
}

// parseURLsFromJSON extracts track URLs from JSON output of yt-dlp --flat-playlist.
func parseURLsFromJSON(jsonStr string) ([]string, error) {
    type entry struct {
        URL string `json:"url"`
    }
    type playlistData struct {
        Entries []entry `json:"entries"`
    }

    var data playlistData
    if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
        return nil, err
    }

    var urls []string
    for _, e := range data.Entries {
        if e.URL != "" {
            urls = append(urls, e.URL)
        }
    }
    return urls, nil
}

// downloadTrackWithFallback tries each format until one succeeds.
func downloadTrackWithFallback(binary, url, playlistDir string, formats []string) error {
    log.Printf("Starting download for track: %s", url)
    defer log.Printf("Finished attempt for track: %s", url)

    // Determine the directory containing yt-dlp_macos (Contents/MacOS/bin/)
    binDir := filepath.Dir(binary)
    // Use the same directory for ffmpeg-location
    ffmpegLocation := binDir

    for _, f := range formats {
        log.Printf("Attempting %s format for track: %s", f, url)
        opts := []string{
            "-x",
            "--audio-format", f,
            "--audio-quality", "0",
            "--ffmpeg-location", ffmpegLocation, // Add this to point to ffmpeg/ffprobe
            "-o", filepath.Join(playlistDir, "%(title)s.%(ext)s"),
            url,
        }

        cmd := exec.Command(binary, opts...)
        output, err := cmd.CombinedOutput()
        if err != nil {
            log.Printf("Failed %s for track %s: %v\nOutput: %s", f, url, err, string(output))
            continue
        }

        log.Printf("Track downloaded successfully in %s format: %s", f, url)
        return nil // Exit on success
    }
    return fmt.Errorf("all format attempts failed for track: %s", url)
}