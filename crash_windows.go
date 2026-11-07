//go:build windows

package main

import "golang.org/x/sys/windows"

func showError(msg string) {
	t, _ := windows.UTF16PtrFromString(msg)
	c, _ := windows.UTF16PtrFromString("Outpost Defense")
	windows.MessageBox(0, t, c, windows.MB_OK|windows.MB_ICONERROR)
}
