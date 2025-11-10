package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
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
