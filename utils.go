package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
)

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
func findAvailableLocalPort(startPort int) (int, error) {
	for port := startPort; port < startPort+1000; port++ {
		addr, err := net.Listen("tcp", fmt.Sprintf(":%d", port))

		if err == nil {
			addr.Close()
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
