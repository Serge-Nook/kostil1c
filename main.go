package main

import (
	_ "embed"
	"fmt"
	"log"
	"os"

	"fyne.io/fyne/v2"

	"fyne.io/fyne/v2/app"
)

const (
	appName    = "Костыль 1С"
	appVersion = "1.0.3"
	appAuthor  = "Горшков Сергей Владимирович"
	appSite    = "https://nookbat.ru/"
	appEmail   = "i@nookbat.ru"
)

//go:embed build/icon.png
var iconPNG []byte

var appIcon = fyne.NewStaticResource("icon.png", iconPNG)

func main() {
	args := platformStartup(os.Args[1:])
	defer func() {
		if r := recover(); r != nil {
			reportStartFailure(fmt.Sprint(r))
			os.Exit(2)
		}
	}()

	a := app.NewWithID("ru.nookbat.kostil1c")
	a.SetIcon(appIcon)
	started := false
	a.Lifecycle().SetOnStarted(func() { started = true })
	u := newUI(a)
	if len(args) > 0 {
		u.load(args[0])
	}
	u.win.ShowAndRun()
	if !started {
		reportStartFailure(nil)
		os.Exit(1)
	}
	log.Print("exit")
}
