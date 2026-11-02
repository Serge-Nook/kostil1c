package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/go-gl/glfw/v3.4/glfw"
)

// On Windows a GUI-subsystem program has no console: when OpenGL 2.1 is
// unavailable (no GPU driver, "Microsoft Basic Display Adapter", RDP, VM) Fyne
// only logs the error and exits, so nothing appears on screen. Therefore:
//   - the log goes to %LOCALAPPDATA%\Kostil1C\kostil1c.log;
//   - OpenGL 2.1 is checked in a child process (--gl-probe);
//   - if it is missing, the program restarts itself from mesa\kostil1c.exe,
//     next to which lies the Mesa llvmpipe opengl32.dll (an app-local DLL
//     takes precedence over the system one);
//   - if the window still cannot be opened, a message box is shown.
//
// KOSTIL1C_GL=software (or --software-gl) forces Mesa, KOSTIL1C_GL=system skips the check.

const (
	glProbeArg = "--gl-probe"
	envGL      = "KOSTIL1C_GL"
	envChild   = "KOSTIL1C_CHILD"
)

// mesaEnv makes Mesa use llvmpipe directly instead of trying Zink/D3D12 first;
// it is ignored by vendor OpenGL drivers.
var mesaEnv = []string{"GALLIUM_DRIVER=llvmpipe"}

var (
	user32      = syscall.NewLazyDLL("user32.dll")
	messageBoxW = user32.NewProc("MessageBoxW")
	logPath     string
)

func messageBox(text string, icon uintptr) {
	t, _ := syscall.UTF16PtrFromString(text)
	c, _ := syscall.UTF16PtrFromString(appName)
	messageBoxW.Call(0, uintptr(unsafe.Pointer(t)), uintptr(unsafe.Pointer(c)), icon)
}

func exeDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func platformStartup(args []string) []string {
	if len(args) > 0 && args[0] == glProbeArg {
		os.Exit(glProbe())
	}
	setupLog()
	log.Printf("%s %s, %s/%s, %s", appName, appVersion, runtime.GOOS, runtime.GOARCH, exeDir())

	mode := strings.ToLower(os.Getenv(envGL))
	var rest []string
	for _, a := range args {
		if a == "--software-gl" {
			mode = "software"
			continue
		}
		rest = append(rest, a)
	}

	if exists(filepath.Join(exeDir(), "opengl32.dll")) {
		mode = "system" // this is the Mesa copy: opengl32.dll next to the exe is used
	}
	switch mode {
	case "system":
	case "software":
		startMesa(rest, "forced")
	default:
		if err := probe(filepath.Join(exeDir(), "kostil1c.exe")); err != nil {
			log.Printf("system OpenGL 2.1 check failed: %v", err)
			startMesa(rest, "system OpenGL 2.1 unavailable")
		}
	}
	return rest
}

func setupLog() {
	dir, err := os.UserCacheDir() // %LOCALAPPDATA%
	if err != nil {
		return
	}
	dir = filepath.Join(dir, "Kostil1C")
	if os.MkdirAll(dir, 0o755) != nil {
		return
	}
	flags := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	if os.Getenv(envChild) != "" {
		flags = os.O_CREATE | os.O_WRONLY | os.O_APPEND
	}
	logPath = filepath.Join(dir, "kostil1c.log")
	f, err := os.OpenFile(logPath, flags, 0o644)
	if err != nil {
		logPath = ""
		return
	}
	log.SetOutput(io.MultiWriter(f, os.Stderr))
}

// glProbe runs in a child process: it creates a hidden window with the context
// Fyne requests (OpenGL 2.1) and prints the renderer name.
func glProbe() int {
	runtime.LockOSThread()
	if err := glfw.Init(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	defer glfw.Terminate()
	glfw.WindowHint(glfw.Visible, glfw.False)
	glfw.WindowHint(glfw.ContextVersionMajor, 2)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	w, err := glfw.CreateWindow(16, 16, "probe", nil, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 3
	}
	w.MakeContextCurrent()
	glGetString := syscall.NewLazyDLL("opengl32.dll").NewProc("glGetString")
	for _, name := range []uintptr{0x1F00, 0x1F01, 0x1F02} { // GL_VENDOR, GL_RENDERER, GL_VERSION
		if p, _, _ := glGetString.Call(name); p != 0 {
			fmt.Print(goString(p), "; ")
		}
	}
	w.Destroy()
	return 0
}

func goString(p uintptr) string {
	b := unsafe.Slice(*(**byte)(unsafe.Pointer(&p)), 256)
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

func hidden(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
}

func probe(exe string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, glProbeArg)
	cmd.Env = append(os.Environ(), mesaEnv...)
	hidden(cmd)
	out, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err != nil {
		return fmt.Errorf("%v: %s", err, strings.TrimSpace(string(out)))
	}
	log.Printf("OpenGL (%s): %s", exe, strings.TrimSpace(string(out)))
	return nil
}

// startMesa restarts the program from mesa\kostil1c.exe and exits.
func startMesa(args []string, reason string) {
	exe := filepath.Join(exeDir(), "mesa", "kostil1c.exe")
	if !exists(exe) || !exists(filepath.Join(exeDir(), "mesa", "opengl32.dll")) {
		log.Printf("Mesa (%s): %s not found", reason, exe)
		reportStartFailure(nil)
		os.Exit(1)
	}
	if err := probe(exe); err != nil {
		log.Printf("Mesa OpenGL check failed: %v", err)
		reportStartFailure(nil)
		os.Exit(1)
	}
	log.Printf("starting Mesa software renderer (%s)", reason)
	cmd := exec.Command(exe, args...)
	cmd.Env = append(append(os.Environ(), mesaEnv...), envGL+"=system", envChild+"=1")
	if err := cmd.Start(); err != nil {
		log.Printf("start %s: %v", exe, err)
		reportStartFailure(err)
		os.Exit(1)
	}
	os.Exit(0)
}

func reportStartFailure(cause any) {
	msg := "Не удалось открыть окно программы."
	if cause != nil {
		msg = fmt.Sprintf("Программа аварийно завершилась:\n%v", cause)
	} else {
		msg += "\n\nНедоступен OpenGL 2.1 (нет драйвера видеокарты, удалённый рабочий стол или виртуальная машина)."
		if !exists(filepath.Join(exeDir(), "mesa")) && !exists(filepath.Join(exeDir(), "opengl32.dll")) {
			msg += "\nПапка mesa рядом с программой не найдена — установите программу через Setup.exe или распакуйте portable-архив целиком."
		}
	}
	if logPath != "" {
		msg += "\n\nЖурнал: " + logPath
	}
	log.Print(msg)
	messageBox(msg, 0x10) // MB_ICONERROR
}
