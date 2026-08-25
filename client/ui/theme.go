package ui

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type customTheme struct {
	fyne.Theme
	primaryColor color.Color
}

func (t *customTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if name == theme.ColorNamePrimary || name == theme.ColorNameFocus {
		return t.primaryColor
	}
	return t.Theme.Color(name, variant)
}

func ParseColor(c string) color.Color {
	switch strings.ToLower(c) {
	case "red":
		return color.RGBA{R: 244, G: 67, B: 54, A: 255}
	case "green":
		return color.RGBA{R: 76, G: 175, B: 80, A: 255}
	case "purple":
		return color.RGBA{R: 156, G: 39, B: 176, A: 255}
	case "orange":
		return color.RGBA{R: 255, G: 152, B: 0, A: 255}
	case "yellow":
		return color.RGBA{R: 255, G: 235, B: 59, A: 255}
	case "blue":
		fallthrough
	default:
		return color.RGBA{R: 33, G: 150, B: 243, A: 255}
	}
}

func ApplyAppTheme(a fyne.App, colorName string) {
	custom := &customTheme{
		Theme:        theme.DefaultTheme(),
		primaryColor: ParseColor(colorName),
	}
	a.Settings().SetTheme(custom)
}
