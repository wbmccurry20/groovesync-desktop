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
		// Validate inputs
		if playlistEntry.Text == "" || nameEntry.Text == "" {
			app.Queue(func() {
				statusLabel.SetText("Error: Playlist URL and Name are required")
				statusLabel.Refresh()
			})
			return
		}

		go startDownload(
			playlistEntry.Text,
			nameEntry.Text,
			formatSelect.Selected,
			downloadDirEntry.Text,
			progressBar,
			statusLabel,
			app,
		)
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

func startDownload(playlistURL, playlistName, audioFormat, downloadDir string, progressBar *widget.ProgressBar, statusLabel *widget.Label, app fyne.App) {
	// Default download directory if none is provided
	if downloadDir == "" {
		downloadDir = "./downloads"
	}

	// Ensure the download directory exists
	if err := os.MkdirAll(downloadDir, os.ModePerm); err != nil {
		app.Queue(func() {
			statusLabel.SetText("Error: Could not create download directory")
			statusLabel.Refresh()
		})
		log.Printf("Failed to create directory %s: %v", downloadDir, err)
		return
	}

	// Update status label for starting the download
	app.Queue(func() {
		statusLabel.SetText("Starting download...")
		progressBar.SetValue(0)
		progressBar.Refresh()
		statusLabel.Refresh()
	})

	// Call the downloader
	err := downloader.RunYTDLPParallel(
		playlistURL,
		downloadDir,
		playlistName,
		audioFormat,
		func(status string) {
			// Update status from downloader callbacks
			app.Queue(func() {
				statusLabel.SetText(status)
				statusLabel.Refresh()
			})
		},
		func(current, total int) {
			// Update progress bar from downloader callbacks
			app.Queue(func() {
				progressBar.Max = float64(total)
				progressBar.SetValue(float64(current))
				progressBar.Refresh()
			})
		},
	)

	if err != nil {
		app.Queue(func() {
			statusLabel.SetText("Download failed. Check logs.")
			statusLabel.Refresh()
		})
		log.Printf("Download failed: %v", err)
		return
	}

	// Final success update
	app.Queue(func() {
		statusLabel.SetText("Download completed successfully!")
		statusLabel.Refresh()
		log.Println("Download completed successfully")
	})
}
