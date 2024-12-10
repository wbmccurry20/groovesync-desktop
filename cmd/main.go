package main

import (
	"groovesync/internal/logging"
	"groovesync/ui"

	"fyne.io/fyne/v2/app"
)

func main() {
	// Initialize the logger
	logging.Init()

	// Create the Fyne app
	myApp := app.New()

	// Initialize the UI
	ui.NewDownloadUI(myApp)

	// Start the Fyne app
	myApp.Run()
}
