//go:build windows

package host

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"
)


func defaultSocketDir() string {
	tempDir := os.TempDir()
	return filepath.Join(tempDir, "open-browser-use")
}

func listenSocket(socketPath string) (net.Listener, error) {
	// On Windows, use TCP on localhost
	addr := fmt.Sprintf("127.0.0.1:%d", TCPPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	return listener, nil
}

func dialSocket(socketPath string, timeout time.Duration) (net.Conn, error) {
	addr := socketPath
	if addr == "" || addr == "tcp://open-browser-use" {
		addr = fmt.Sprintf("127.0.0.1:%d", TCPPort)
	}
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", addr, err)
	}
	return conn, nil
}

func chmodSocket(path string) error {
	return nil
}

func chmodFile(path string) error {
	return nil
}

func mkdirAll(path string) error {
	return os.MkdirAll(path, 0755)
}

func defaultTimeout() time.Duration {
	return 10 * time.Second
}
