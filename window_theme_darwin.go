//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

static void setWindowDarkAppearance(void *handle, int dark) {
	NSWindow *window = (__bridge NSWindow *)handle;
	if (window == nil) {
		return;
	}
	window.appearance = [NSAppearance appearanceNamed:
		dark ? NSAppearanceNameDarkAqua : NSAppearanceNameAqua];
	window.backgroundColor = dark
		? [NSColor colorWithRed:28.0/255.0 green:28.0/255.0 blue:30.0/255.0 alpha:1.0]
		: [NSColor colorWithRed:248.0/255.0 green:249.0/255.0 blue:252.0/255.0 alpha:1.0];
}
*/
import "C"

import "github.com/wailsapp/wails/v3/pkg/application"

func setNativeWindowTheme(window *application.WebviewWindow, dark bool) {
	application.InvokeSync(func() {
		value := C.int(0)
		if dark {
			value = 1
		}
		C.setWindowDarkAppearance(window.NativeWindow(), value)
	})
}
