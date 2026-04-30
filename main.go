package main

import (
	"flag"
	"log/slog"
	"os"
	"strconv"

	"sonoscli-gui/internal/ui"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

func main() {
	webPort := flag.Int("web", 0, "Start web server on specified port (e.g. 8080). If 0, starts desktop GUI.")
	flag.Parse()

	// Enable Debug logging for sonoscli internals
	opts := &slog.HandlerOptions{Level: slog.LevelDebug}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, opts)))

	a := app.New()
	a.Settings().SetTheme(&ui.SonosTheme{})

	w := a.NewWindow("sonoscli-gui")
	w.Resize(fyne.NewSize(1100, 800))

	sa := ui.NewSonosApp(a, w)
	sa.Start()

	if *webPort > 0 {
		fyne.Do(func() {
			sa.WebPortEntry.SetText(strconv.Itoa(*webPort))
			sa.WebToggle.SetChecked(true)
		})
	}

	w.ShowAndRun()
}
