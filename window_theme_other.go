//go:build !darwin

package main

import "github.com/wailsapp/wails/v3/pkg/application"

func setNativeWindowTheme(_ *application.WebviewWindow, _ bool) {}
