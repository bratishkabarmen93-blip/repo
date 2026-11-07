//go:build !windows

package main

import "fmt"

func showError(msg string) { fmt.Println(msg) }
