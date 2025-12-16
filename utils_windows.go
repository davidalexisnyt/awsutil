//go:build windows

package main

import (
	"os"
	"os/signal"
	"syscall"
)

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
// setupSignalHandlerWindows sets up Windows-specific console control handler to catch Ctrl+C.
// This is necessary because when a child process is attached to the console,
// Ctrl+C goes to the child process, not the parent Go process.
func setupSignalHandlerWindows(sigChan chan os.Signal) {
	// Load kernel32.dll to access SetConsoleCtrlHandler
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	setConsoleCtrlHandler := kernel32.NewProc("SetConsoleCtrlHandler")

	// Define the handler function
	// Windows callback signature: BOOL WINAPI HandlerRoutine(DWORD dwCtrlType)
	handler := syscall.NewCallback(func(ctrlType uintptr) uintptr {
		// CTRL_C_EVENT = 0, CTRL_BREAK_EVENT = 1
		if ctrlType == 0 || ctrlType == 1 {
			// Send interrupt signal to the channel
			select {
			case sigChan <- os.Interrupt:
			default:
			}
			return 1 // TRUE - we handled the event
		}
		return 0 // FALSE - let other handlers process it
	})

	// Register the console control handler (TRUE = add handler)
	ret, _, _ := setConsoleCtrlHandler.Call(handler, 1)
	if ret == 0 {
		// If we can't set up the handler, fall back to standard signal handling
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		return
	}

	// Also set up standard signal handling as a fallback
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
}
