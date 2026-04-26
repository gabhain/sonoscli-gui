package main

import (
	"log/slog"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"sonoscli-gui/internal/ui"
)

func main() {
	// Enable Debug logging for sonoscli internals
	opts := &slog.HandlerOptions{Level: slog.LevelDebug}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, opts)))

	a := app.NewWithID("com.sonoscli.gui")
	a.Settings().SetTheme(&ui.SonosTheme{})
	
	w := a.NewWindow("sonoscli-gui")
	w.Resize(fyne.NewSize(1100, 800))

	sa := ui.NewSonosApp(a, w)
	sa.Start()

	w.ShowAndRun()
}
