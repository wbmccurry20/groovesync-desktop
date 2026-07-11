// groovesync/internal/downloader/downloader.go
package downloader

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// Track defines a single track, exported for use in main
type Track struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

// GetYTDLPBinaryPath determines the path to the yt-dlp binary based on the OS and architecture.
func GetYTDLPBinaryPath() (string, error) {
	// Define binary name based on OS
	binaryName := "yt-dlp_macos"
	if runtime.GOOS == "linux" {
		binaryName = "yt-dlp_linux"
	} else if runtime.GOOS == "windows" {
		binaryName = "yt-dlp.exe"
	}

	// First, try the project root's frontend/dist/bin/ directory (development mode)
	baseDir, err := os.Getwd()
	if err != nil {
		log.Printf("Failed to get working directory: %v", err)
	}
	devBinaryPath := filepath.Join(baseDir, "frontend", "dist", "bin", binaryName)
	if _, err := os.Stat(devBinaryPath); err == nil {
		log.Printf("Found yt-dlp binary at %s", devBinaryPath)
		return devBinaryPath, nil
	}

	// Check if running inside the .app bundle (packaged mode)
	var bundleBinaryPath string // Declare at function level to fix scope
	execPath, err := os.Executable()
	if err == nil {
		bundlePath := filepath.Dir(execPath)
		bundleBinaryPath = filepath.Join(bundlePath, "bin", binaryName)
		if _, err := os.Stat(bundleBinaryPath); err == nil {
			log.Printf("Found yt-dlp binary at %s", bundleBinaryPath)
			return bundleBinaryPath, nil
		}
	}

	// Fallback to Homebrew-installed yt-dlp
	homebrewBinary := "/opt/homebrew/bin/yt-dlp"
	if runtime.GOOS == "windows" {
		homebrewBinary = "yt-dlp.exe"
	} else if runtime.GOOS == "linux" {
		homebrewBinary = "/usr/local/bin/yt-dlp"
	}
	if _, err := os.Stat(homebrewBinary); err == nil {
		log.Printf("Found yt-dlp binary at %s", homebrewBinary)
		return homebrewBinary, nil
	}

	return "", fmt.Errorf("yt-dlp binary not found at %s, %s, or %s", devBinaryPath, bundleBinaryPath, homebrewBinary)
}

// GetFFmpegLocation returns the directory that contains the bundled ffmpeg and
// ffprobe binaries. yt-dlp accepts this via --ffmpeg-location so audio
// extraction/conversion works without ffmpeg being on the system PATH.
func GetFFmpegLocation() string {
	// Development mode: frontend/dist/bin
	if wd, err := os.Getwd(); err == nil {
		dev := filepath.Join(wd, "frontend", "dist", "bin")
		if _, err := os.Stat(filepath.Join(dev, ffmpegName())); err == nil {
			return dev
		}
	}
	// Packaged mode: <executable dir>/bin
	if execPath, err := os.Executable(); err == nil {
		bundle := filepath.Join(filepath.Dir(execPath), "bin")
		if _, err := os.Stat(filepath.Join(bundle, ffmpegName())); err == nil {
			return bundle
		}
	}
	// Fallback to a Homebrew/system location.
	if runtime.GOOS == "darwin" {
		if _, err := os.Stat("/opt/homebrew/bin/ffmpeg"); err == nil {
			return "/opt/homebrew/bin"
		}
	}
	return ""
}

func ffmpegName() string {
	if runtime.GOOS == "windows" {
		return "ffmpeg.exe"
	}
	return "ffmpeg"
}

// ytdlpBinary resolves the yt-dlp binary path, falling back to the bare command
// name if resolution fails so the caller can still attempt to run it.
func ytdlpBinary() string {
	if path, err := GetYTDLPBinaryPath(); err == nil {
		return path
	}
	return "yt-dlp"
}

// RunYTDLPParallel downloads tracks from a playlist URL in parallel.
func RunYTDLPParallel(
	playlistURL, downloadDir, playlistName, userFormat string,
	updateStatus func(string),
	updateProgress func(current, total int),
) error {
	log.Printf("RunYTDLPParallel started for %s", playlistURL)

	if userFormat == "" {
		userFormat = "wav"
	}

	playlistDir := filepath.Join(downloadDir, playlistName)
	log.Printf("Using playlist directory: %s", playlistDir)

	// Extract track URLs
	trackURLs, err := extractTrackURLs(playlistURL)
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

			err := downloadTrackWithFallback(trackURL, playlistDir, []string{userFormat, "wav", "opus", "mp3"})
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
func extractTrackURLs(playlistURL string) ([]string, error) {
	log.Printf("Extracting tracks from %s", playlistURL)
	cmdArgs := []string{"--flat-playlist", "--no-warnings", "-J", playlistURL}
	cookiesPath := filepath.Join(os.Getenv("HOME"), "cookies.txt")
	if _, err := os.Stat(cookiesPath); err == nil {
		cmdArgs = append([]string{"--cookies", cookiesPath}, cmdArgs...)
		log.Printf("Using cookies file: %s", cookiesPath)
	}
	cmd := exec.Command(ytdlpBinary(), cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("yt-dlp error: %v\nOutput: %s", err, string(output))
		return nil, err
	}
	log.Printf("extractTrackURLs raw output: %s", string(output))
	return parseURLsFromJSON(string(output))
}

// parseURLsFromJSON extracts track URLs from JSON output of yt-dlp --flat-playlist.
func parseURLsFromJSON(jsonStr string) ([]string, error) {
	type entry struct {
		URL   string `json:"url"`
		Title string `json:"title"`
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
	log.Printf("Parsed %d URLs from JSON", len(urls))
	return urls, nil
}

// downloadTrackWithFallback tries each format until one succeeds.
func downloadTrackWithFallback(url, playlistDir string, formats []string) error {
	log.Printf("Starting download for track: %s", url)
	defer log.Printf("Finished attempt for track: %s", url)

	cookiesPath := filepath.Join(os.Getenv("HOME"), "cookies.txt")
	ffmpegLocation := GetFFmpegLocation()
	ytDLP := ytdlpBinary()
	for _, f := range formats {
		log.Printf("Attempting %s format for track: %s", f, url)
		cmdArgs := []string{
			"-x",
			"--audio-format", f,
			"--audio-quality", "0",
			"-o", filepath.Join(playlistDir, "%(title)s.%(ext)s"),
			url,
		}
		if ffmpegLocation != "" {
			cmdArgs = append([]string{"--ffmpeg-location", ffmpegLocation}, cmdArgs...)
		}
		if _, err := os.Stat(cookiesPath); err == nil {
			cmdArgs = append([]string{"--cookies", cookiesPath}, cmdArgs...)
		}
		cmd := exec.Command(ytDLP, cmdArgs...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("Failed %s for track %s: %v\nOutput: %s", f, url, err, string(output))
			continue
		}
		log.Printf("Track downloaded successfully in %s format: %s", f, url)
		return nil
	}
	return fmt.Errorf("all format attempts failed for track: %s", url)
}

// ExportToRekordbox generates a Rekordbox XML file for the playlist.
func ExportToRekordbox(playlistDir, playlistName, format string) error {
	type TrackEntry struct {
		TrackID  int    `xml:"TrackID,attr"`
		Name     string `xml:"Name,attr"`
		Location string `xml:"Location,attr"`
	}
	type PlaylistNode struct {
		Name   string       `xml:"Name,attr"`
		Tracks []TrackEntry `xml:"TRACK"`
	}
	type Collection struct {
		Entries []TrackEntry `xml:"TRACK"`
	}
	type RekordboxXML struct {
		XMLName    xml.Name       `xml:"DJ_PLAYLISTS"`
		Version    string         `xml:"Version,attr"`
		Collection Collection     `xml:"COLLECTION"`
		Playlists  []PlaylistNode `xml:"PLAYLISTS>NODE"`
	}

	files, err := os.ReadDir(playlistDir)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %v", playlistDir, err)
	}

	var tracks []TrackEntry
	for i, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), "."+format) {
			continue
		}
		name := strings.TrimSuffix(f.Name(), "."+format)
		location := filepath.Join(playlistDir, f.Name())
		tracks = append(tracks, TrackEntry{
			TrackID:  i + 1,
			Name:     name,
			Location: "file://localhost/" + filepath.ToSlash(location),
		})
	}

	rbXML := RekordboxXML{
		Version:    "1.0.0",
		Collection: Collection{Entries: tracks},
		Playlists: []PlaylistNode{{
			Name:   playlistName,
			Tracks: tracks,
		}},
	}

	outputPath := filepath.Join(playlistDir, playlistName+".xml")
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create XML file %s: %v", outputPath, err)
	}
	defer f.Close()

	enc := xml.NewEncoder(f)
	enc.Indent("", "  ")
	if err := enc.Encode(rbXML); err != nil {
		return fmt.Errorf("failed to encode XML: %v", err)
	}

	return nil
}
