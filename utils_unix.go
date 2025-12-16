//go:build !windows

package main

import (
	"os"
	"os/signal"
	"syscall"
)

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
// setupSignalHandlerWindows is a stub for non-Windows platforms.
// On Unix systems, standard signal handling is used directly in setupSignalHandler.
func setupSignalHandlerWindows(sigChan chan os.Signal) {
	// This should never be called on non-Windows platforms
	// Standard signal handling is used instead
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
}
