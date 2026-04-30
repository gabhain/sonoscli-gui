package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type SonosTheme struct{}

func (m SonosTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if variant == theme.VariantDark {
		switch name {
		case theme.ColorNameBackground:
			return color.NRGBA{R: 18, G: 18, B: 18, A: 255}
		case theme.ColorNameInputBackground:
			return color.NRGBA{R: 30, G: 30, B: 30, A: 255}
		case theme.ColorNamePrimary:
			return color.NRGBA{R: 255, G: 165, B: 0, A: 255} // Sonos Orange
		}
		return theme.DarkTheme().Color(name, variant)
	}
	return theme.LightTheme().Color(name, variant)
}

func (m SonosTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DarkTheme().Icon(name)
}

func (m SonosTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DarkTheme().Font(style)
}

func (m SonosTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DarkTheme().Size(name)
}
