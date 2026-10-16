package main

import (
	_ "embed"
	"os"

	"fyne.io/fyne/v2"

	"fyne.io/fyne/v2/app"
)

const (
	appName    = "Костыль 1С"
	appVersion = "1.0.1"
	appAuthor  = "Горшков Сергей Владимирович"
	appSite    = "https://nookbat.ru/"
)

//go:embed build/icon.png
var iconPNG []byte

var appIcon = fyne.NewStaticResource("icon.png", iconPNG)

func main() {
	a := app.NewWithID("ru.nookbat.kostil1c")
	a.SetIcon(appIcon)
	u := newUI(a)
	if len(os.Args) > 1 {
		u.load(os.Args[1])
	}
	u.win.ShowAndRun()
}
