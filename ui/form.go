package ui

import (
	"groovesync/internal/downloader"
	"log"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func NewDownloadUI(app fyne.App) {
	w := app.NewWindow("GrooveSync")

	// Status Label for feedback
	statusLabel := widget.NewLabel("Ready to start downloading...")
	progressBar := widget.NewProgressBar()

	// Form for user input
	playlistEntry := widget.NewEntry()
	playlistEntry.SetPlaceHolder("Enter Playlist URL")

	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Enter Playlist Name")

	formatSelect := widget.NewSelect([]string{"wav", "mp3", "opus"}, nil)

	downloadDirEntry := widget.NewEntry()
	downloadDirEntry.SetPlaceHolder("Enter Download Directory (optional)")

	// Start Download Button
	startButton := widget.NewButton("Start Download", func() {
		log.Println("Start Download button clicked")

		// Validate inputs
		if playlistEntry.Text == "" || nameEntry.Text == "" {
			statusLabel.SetText("Error: Playlist URL and Name are required")
			return
		}

		// Set default download directory if not specified
		downloadDir := downloadDirEntry.Text
		if downloadDir == "" {
			downloadDir = "./downloads"
		}

		// Ensure the download directory exists
		if err := os.MkdirAll(downloadDir, os.ModePerm); err != nil {
			statusLabel.SetText("Error: Could not create download directory")
			log.Printf("Failed to create directory %s: %v", downloadDir, err)
			return
		}

		// Start download in a goroutine
		go func() {
			// Callback for updating status
			updateStatus := func(status string) {
				statusLabel.SetText(status)
			}

			// Callback for updating progress bar
			updateProgress := func(current, total int) {
				progressBar.Max = float64(total)
				progressBar.SetValue(float64(current))
			}

			// Call downloader
			err := downloader.RunYTDLPParallel(
				playlistEntry.Text,
				downloadDir,
				nameEntry.Text,
				formatSelect.Selected,
				updateStatus,
				updateProgress,
			)
			if err != nil {
				statusLabel.SetText("Download failed. Check logs.")
				log.Printf("Download failed: %v", err)
				return
			}

			// Final success update
			statusLabel.SetText("Download completed successfully!")
			log.Println("Download completed successfully")
		}()
	})

	// Layout the form and feedback elements
	form := container.NewVBox(
		widget.NewLabel("GrooveSync Downloader"),
		widget.NewForm(
			widget.NewFormItem("Playlist URL", playlistEntry),
			widget.NewFormItem("Playlist Name", nameEntry),
			widget.NewFormItem("Audio Format", formatSelect),
			widget.NewFormItem("Download Directory", downloadDirEntry),
		),
		startButton,
		statusLabel,
		progressBar,
	)

	w.SetContent(form)
	w.Resize(fyne.NewSize(400, 300))
	w.Show()
}
