package main

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/Serge-Nook/kostil1c/internal/export"
	"github.com/Serge-Nook/kostil1c/internal/table"
)

const (
	prefExportDir  = "exportDir"
	prefFormats    = "formats"
	prefWithInfo   = "withInfo"
	prefLastOpened = "lastOpenedDir"
)

type ui struct {
	app fyne.App
	win fyne.Window

	doc      *table.Doc
	depts    []table.Department
	selected []bool

	fileLabel *widget.Label
	summary   *widget.Label
	counter   *widget.Label
	search    *widget.Entry
	listBox   *fyne.Container
	exportBtn *widget.Button
}

func newUI(a fyne.App) *ui {
	u := &ui{app: a, win: a.NewWindow(appName + " " + appVersion)}
	u.win.SetIcon(appIcon)

	openBtn := widget.NewButtonWithIcon("Открыть CSV…", theme.FolderOpenIcon(), u.openFile)
	openBtn.Importance = widget.HighImportance
	aboutBtn := widget.NewButtonWithIcon("О программе", theme.InfoIcon(), u.showAbout)

	u.fileLabel = widget.NewLabel("Файл не загружен. Откройте CSV или перетащите его в окно.")
	u.fileLabel.Truncation = fyne.TextTruncateEllipsis
	u.summary = widget.NewLabel("")

	u.search = widget.NewEntry()
	u.search.SetPlaceHolder("Поиск подразделения…")
	u.search.OnChanged = func(string) { u.rebuildList() }

	allBtn := widget.NewButtonWithIcon("Выбрать все", theme.CheckButtonCheckedIcon(), func() { u.setAll(true) })
	noneBtn := widget.NewButtonWithIcon("Снять все", theme.CheckButtonIcon(), func() { u.setAll(false) })

	u.listBox = container.NewVBox()
	u.counter = widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	u.exportBtn = widget.NewButtonWithIcon("Экспорт…", theme.DocumentSaveIcon(), u.showExport)
	u.exportBtn.Importance = widget.HighImportance

	top := container.NewVBox(
		container.NewBorder(nil, nil, openBtn, aboutBtn, u.fileLabel),
		u.summary,
		container.NewBorder(nil, nil, nil, container.NewHBox(allBtn, noneBtn), u.search),
		widget.NewSeparator(),
	)
	bottom := container.NewVBox(
		widget.NewSeparator(),
		container.NewBorder(nil, nil, nil, u.exportBtn, u.counter),
	)
	u.win.SetContent(container.NewBorder(top, bottom, nil, nil, container.NewVScroll(u.listBox)))
	u.win.Resize(fyne.NewSize(900, 640))
	u.win.SetOnDropped(func(_ fyne.Position, uris []fyne.URI) {
		for _, uri := range uris {
			if strings.EqualFold(uri.Extension(), ".csv") {
				u.load(uri.Path())
				return
			}
		}
		if len(uris) > 0 {
			u.load(uris[0].Path())
		}
	})
	u.refresh()
	return u
}

func (u *ui) openFile() {
	d := dialog.NewFileOpen(func(r fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(err, u.win)
			return
		}
		if r == nil {
			return
		}
		path := r.URI().Path()
		r.Close()
		u.load(path)
	}, u.win)
	d.SetFilter(storage.NewExtensionFileFilter([]string{".csv", ".CSV"}))
	if dir := u.app.Preferences().String(prefLastOpened); dir != "" {
		if l, err := storage.ListerForURI(storage.NewFileURI(dir)); err == nil {
			d.SetLocation(l)
		}
	}
	d.Resize(fyne.NewSize(800, 560))
	d.Show()
}

func (u *ui) load(path string) {
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	doc, err := table.Load(path)
	if err != nil {
		dialog.ShowError(fmt.Errorf("не удалось загрузить %s:\n%w", filepath.Base(path), err), u.win)
		return
	}
	u.app.Preferences().SetString(prefLastOpened, filepath.Dir(path))
	u.doc = doc
	u.depts = doc.Departments()
	u.selected = make([]bool, len(u.depts))
	for i := range u.selected {
		u.selected[i] = true
	}
	u.fileLabel.SetText(path)
	u.search.SetText("")
	u.summary.SetText(fmt.Sprintf("Сотрудников: %d   ·   Подразделений: %d   ·   Столбец подразделения: «%s»",
		len(doc.Rows), len(u.depts), doc.Header[table.DeptColumn]))
	u.rebuildList()
}

func (u *ui) visible() []int {
	q := strings.ToLower(strings.TrimSpace(u.search.Text))
	var out []int
	for i, d := range u.depts {
		if q == "" || strings.Contains(strings.ToLower(d.Name), q) {
			out = append(out, i)
		}
	}
	return out
}

func (u *ui) rebuildList() {
	u.listBox.RemoveAll()
	for _, i := range u.visible() {
		u.listBox.Add(u.deptRow(i))
	}
	u.listBox.Refresh()
	u.refresh()
}

func (u *ui) deptRow(i int) fyne.CanvasObject {
	d := u.depts[i]
	check := widget.NewCheck("", nil)
	check.SetChecked(u.selected[i])
	check.OnChanged = func(v bool) {
		u.selected[i] = v
		u.refresh()
	}
	name := widget.NewLabel(d.Name)
	name.Wrapping = fyne.TextWrapWord
	count := widget.NewLabel(fmt.Sprintf("%d сотр.", len(d.Rows)))
	row := container.NewBorder(nil, nil, check, count, newTapArea(name, func() { check.SetChecked(!check.Checked) }))
	return row
}

func (u *ui) setAll(v bool) {
	for _, i := range u.visible() {
		u.selected[i] = v
	}
	u.rebuildList()
}

func (u *ui) chosen() []table.Department {
	var out []table.Department
	for i, d := range u.depts {
		if u.selected[i] {
			out = append(out, d)
		}
	}
	return out
}

func (u *ui) refresh() {
	sel, people := 0, 0
	for i, d := range u.depts {
		if u.selected[i] {
			sel++
			people += len(d.Rows)
		}
	}
	u.counter.SetText(fmt.Sprintf("Выбрано подразделений: %d из %d   (сотрудников: %d)", sel, len(u.depts), people))
	if sel == 0 {
		u.exportBtn.Disable()
	} else {
		u.exportBtn.Enable()
	}
}

func (u *ui) showExport() {
	depts := u.chosen()
	if u.doc == nil || len(depts) == 0 {
		return
	}
	prefs := u.app.Preferences()

	saved := prefs.StringWithFallback(prefFormats, "csv,xlsx")
	checks := map[export.Format]*widget.Check{}
	formatBox := container.NewVBox()
	for _, f := range export.Formats {
		c := widget.NewCheck(f.Label(), nil)
		c.SetChecked(strings.Contains(","+saved+",", ","+string(f)+","))
		checks[f] = c
		formatBox.Add(c)
	}
	withInfo := widget.NewCheck("Добавлять информационные строки (1–3) из исходного файла", nil)
	withInfo.SetChecked(prefs.BoolWithFallback(prefWithInfo, true))

	dirEntry := widget.NewEntry()
	base := strings.TrimSuffix(filepath.Base(u.doc.Path), filepath.Ext(u.doc.Path))
	dirEntry.SetText(prefs.StringWithFallback(prefExportDir, filepath.Join(filepath.Dir(u.doc.Path), base+" — по подразделениям")))
	browse := widget.NewButtonWithIcon("Обзор…", theme.FolderOpenIcon(), func() {
		fd := dialog.NewFolderOpen(func(l fyne.ListableURI, err error) {
			if err == nil && l != nil {
				dirEntry.SetText(l.Path())
			}
		}, u.win)
		if l, err := storage.ListerForURI(storage.NewFileURI(dirEntry.Text)); err == nil {
			fd.SetLocation(l)
		}
		fd.Resize(fyne.NewSize(800, 560))
		fd.Show()
	})

	progress := widget.NewProgressBar()
	progress.Hide()
	status := widget.NewLabel(fmt.Sprintf("Будет экспортировано подразделений: %d", len(depts)))
	status.Wrapping = fyne.TextWrapWord

	var dlg dialog.Dialog
	var runBtn, openDirBtn *widget.Button
	closeBtn := widget.NewButtonWithIcon("Закрыть", theme.CancelIcon(), func() { dlg.Hide() })
	openDirBtn = widget.NewButtonWithIcon("Открыть папку", theme.FolderIcon(), func() {
		u.app.OpenURL(&url.URL{Scheme: "file", Path: filepath.ToSlash(dirEntry.Text)})
	})
	openDirBtn.Hide()
	runBtn = widget.NewButtonWithIcon("ЗАПУСК", theme.MediaPlayIcon(), func() {
		var formats []export.Format
		var names []string
		for _, f := range export.Formats {
			if checks[f].Checked {
				formats = append(formats, f)
				names = append(names, string(f))
			}
		}
		dir := strings.TrimSpace(dirEntry.Text)
		if len(formats) == 0 {
			dialog.ShowInformation("Экспорт", "Выберите хотя бы один формат.", u.win)
			return
		}
		if dir == "" {
			dialog.ShowInformation("Экспорт", "Выберите папку для экспорта.", u.win)
			return
		}
		prefs.SetString(prefFormats, strings.Join(names, ","))
		prefs.SetString(prefExportDir, dir)
		prefs.SetBool(prefWithInfo, withInfo.Checked)

		runBtn.Disable()
		closeBtn.Disable()
		openDirBtn.Hide()
		progress.SetValue(0)
		progress.Show()
		status.SetText("Экспорт…")
		info := withInfo.Checked
		go func() {
			res := export.Run(u.doc, depts, formats, dir, info, func(done, total int) {
				fyne.Do(func() { progress.SetValue(float64(done) / float64(total)) })
			})
			fyne.Do(func() {
				runBtn.Enable()
				closeBtn.Enable()
				msg := fmt.Sprintf("Готово. Создано файлов: %d в папке %s", len(res.Files), dir)
				if len(res.Errors) > 0 {
					var lines []string
					for _, e := range res.Errors {
						lines = append(lines, e.Error())
					}
					msg += fmt.Sprintf("\nОшибок: %d\n%s", len(res.Errors), strings.Join(lines, "\n"))
				}
				status.SetText(msg)
				if len(res.Files) > 0 {
					openDirBtn.Show()
				}
			})
		}()
	})
	runBtn.Importance = widget.HighImportance

	content := container.NewVBox(
		widget.NewLabelWithStyle("Форматы (можно выбрать несколько):", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		formatBox,
		withInfo,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Папка для экспорта:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewBorder(nil, nil, nil, browse, dirEntry),
		widget.NewSeparator(),
		status,
		progress,
		container.NewHBox(runBtn, openDirBtn, layoutSpacer(), closeBtn),
	)
	dlg = dialog.NewCustomWithoutButtons("Экспорт", content, u.win)
	dlg.Resize(fyne.NewSize(640, 0))
	dlg.Show()
}

func (u *ui) showAbout() {
	site, _ := url.Parse(appSite)
	content := container.NewVBox(
		widget.NewLabelWithStyle(appName, fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Версия: "+appVersion),
		widget.NewLabel("Автор: "+appAuthor),
		widget.NewHyperlink("nookbat.ru", site),
		widget.NewLabel(""),
		widget.NewLabel("Делит CSV-выгрузку из 1С (разделитель «;», UTF-8)\n"+
			"на отдельные таблицы по подразделениям (3-й столбец)\n"+
			"и сохраняет их в CSV, XLS (xlsx), PDF и HTML."),
	)
	dialog.NewCustom("О программе", "OK", content, u.win).Show()
}
