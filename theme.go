package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

const prefDarkTheme = "darkTheme"

type variantTheme struct {
	fyne.Theme
	variant fyne.ThemeVariant
}

func (t variantTheme) Color(n fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	return t.Theme.Color(n, t.variant)
}

func applyTheme(a fyne.App, dark bool) {
	v := theme.VariantLight
	if dark {
		v = theme.VariantDark
	}
	a.Settings().SetTheme(variantTheme{Theme: theme.DefaultTheme(), variant: v})
}

func isDark(a fyne.App) bool {
	return a.Preferences().BoolWithFallback(prefDarkTheme, a.Settings().ThemeVariant() == theme.VariantDark)
}
