package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"
)

// crashed converts a recovered panic into an error that carries the stack trace.
func crashed(r any) error {
	return fmt.Errorf("panic: %v\n\n%s", r, debug.Stack())
}

func writeCrashLog(msg string) string {
	text := fmt.Sprintf("Outpost Defense crash report\n%s\n\n%s\n", time.Now().Format(time.RFC3339), msg)
	var dirs []string
	if exe, err := os.Executable(); err == nil {
		dirs = append(dirs, filepath.Dir(exe))
	}
	dirs = append(dirs, os.TempDir())
	for _, d := range dirs {
		p := filepath.Join(d, "OutpostDefense-crash.txt")
		if err := os.WriteFile(p, []byte(text), 0o644); err == nil {
			return p
		}
	}
	return "(could not write a crash log)"
}
