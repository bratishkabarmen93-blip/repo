//go:build js

package main

import "errors"

// The browser build (used for testing) keeps its save in memory only.
var memSave []byte

func readSaveData() ([]byte, error) {
	if memSave == nil {
		return nil, errors.New("no save")
	}
	return memSave, nil
}

func writeSaveData(b []byte) error {
	memSave = b
	return nil
}
