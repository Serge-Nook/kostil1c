package export

import (
	"fmt"
	"strings"
	"unicode"
)

// MaxNameLen is the maximum file name length (without extension), in characters.
const MaxNameLen = 100

var reservedNames = map[string]bool{
	"CON": true, "PRN": true, "AUX": true, "NUL": true,
	"COM1": true, "COM2": true, "COM3": true, "COM4": true, "COM5": true,
	"COM6": true, "COM7": true, "COM8": true, "COM9": true,
	"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true, "LPT5": true,
	"LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
}

// SafeName turns a department name into a file name valid on Windows and Linux,
// truncated to MaxNameLen characters.
func SafeName(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case strings.ContainsRune(`<>:"/\|?*`, r), r < 32, unicode.IsControl(r):
			b.WriteRune('_')
		case unicode.IsSpace(r):
			b.WriteRune(' ')
		default:
			b.WriteRune(r)
		}
	}
	s := strings.Join(strings.Fields(b.String()), " ")
	s = truncate(s, MaxNameLen)
	s = strings.TrimRight(s, " .")
	if s == "" {
		s = "Без названия"
	}
	if reservedNames[strings.ToUpper(s)] {
		s = "_" + s
	}
	return s
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

// Namer hands out unique file names (case-insensitive, as on Windows).
type Namer struct{ used map[string]bool }

// NewNamer returns an empty Namer.
func NewNamer() *Namer { return &Namer{used: map[string]bool{}} }

// Name returns a unique safe file name for the department.
func (n *Namer) Name(dept string) string {
	base := SafeName(dept)
	name := base
	for i := 2; n.used[strings.ToLower(name)]; i++ {
		suffix := fmt.Sprintf(" (%d)", i)
		name = strings.TrimRight(truncate(base, MaxNameLen-len([]rune(suffix))), " .") + suffix
	}
	n.used[strings.ToLower(name)] = true
	return name
}
