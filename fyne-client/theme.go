package main

import (
	"image/color"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type supernovaTheme struct{}

func (m supernovaTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.NRGBA{R: 18, G: 18, B: 18, A: 255} // Very dark background
	case theme.ColorNameButton:
		return color.NRGBA{R: 30, G: 30, B: 30, A: 255}
	case theme.ColorNameDisabledButton:
		return color.NRGBA{R: 50, G: 50, B: 50, A: 255}
	case theme.ColorNameHover:
		return color.NRGBA{R: 255, G: 0, B: 110, A: 40} // Pink hover
	case theme.ColorNameFocus:
		return color.NRGBA{R: 157, G: 78, B: 221, A: 120} // Purple focus
	case theme.ColorNamePrimary:
		return color.NRGBA{R: 157, G: 78, B: 221, A: 255} // #9d4edd Purple
	case theme.ColorNameForeground:
		return color.NRGBA{R: 240, G: 240, B: 240, A: 255}
	case theme.ColorNamePlaceHolder:
		return color.NRGBA{R: 120, G: 120, B: 120, A: 255}
	case theme.ColorNameSelection:
		return color.NRGBA{R: 0, G: 245, B: 212, A: 60} // Cyan selection
	}
	return theme.DefaultTheme().Color(name, theme.VariantDark)
}

func (m supernovaTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (m supernovaTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (m supernovaTheme) Size(name fyne.ThemeSizeName) float32 {
	if name == theme.SizeNamePadding {
		return 8
	}
	return theme.DefaultTheme().Size(name)
}
