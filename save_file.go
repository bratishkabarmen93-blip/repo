//go:build !js

package main

import (
	"os"
	"path/filepath"
)

// savePath is %AppData%\OutpostDefense\save.json on Windows.
func savePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "OutpostDefense", "save.json"), nil
}

func readSaveData() ([]byte, error) {
	p, err := savePath()
	if err != nil {
		return nil, err
	}
	return os.ReadFile(p)
}

// writeSaveData replaces the save file atomically.
func writeSaveData(b []byte) error {
	p, err := savePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}
