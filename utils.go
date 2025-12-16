package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
func findAvailableLocalPort(startPort int) (int, error) {
	for port := startPort; port < startPort+1000; port++ {
		// Try to listen on all interfaces (same as HTTP server will use)
		// This checks if the port is truly available for binding
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			// Port is available - close the listener immediately
			listener.Close()

			// Small delay to ensure port is fully released (especially on Windows)
			time.Sleep(10 * time.Millisecond)

			return port, nil
		}
	}

	return 0, fmt.Errorf("could not find available port starting from %d", startPort)
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
func generateBastionID() (string, error) {
	bytes := make([]byte, 8)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
// setupSignalHandler sets up signal handling for Ctrl+C that works on both Windows and Unix systems.
// On Windows, it uses console control handlers to catch Ctrl+C events.
// On Unix systems, it uses standard signal handling.
func setupSignalHandler(sigChan chan os.Signal) {
	if runtime.GOOS == "windows" {
		setupSignalHandlerWindows(sigChan)
	} else {
		// On Unix systems, standard signal handling works fine
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	}
}
