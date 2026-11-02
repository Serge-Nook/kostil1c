//go:build !windows

package main

import "log"

func platformStartup(args []string) []string { return args }

func reportStartFailure(cause any) {
	if cause != nil {
		log.Printf("Программа аварийно завершилась: %v", cause)
	} else {
		log.Print("Не удалось открыть окно программы (нужен OpenGL 2.1).")
	}
}
