//go:build !windows

package host

import (
	"net"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const DefaultSocketDirUnix = "/tmp/open-browser-use"

func defaultSocketDir() string {
	if runtime.GOOS == "darwin" {
		return filepath.Join(os.TempDir(), "open-browser-use")
	}
	return DefaultSocketDirUnix
}

func listenSocket(socketPath string) (net.Listener, error) {
	return net.Listen("unix", socketPath)
}

func dialSocket(socketPath string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout("unix", socketPath, timeout)
}

func chmodSocket(path string) error {
	return os.Chmod(path, 0o600)
}

func chmodFile(path string) error {
	return os.Chmod(path, 0o600)
}

func mkdirAll(path string) error {
	return os.MkdirAll(path, 0o700)
}

func defaultTimeout() time.Duration {
	return 10 * time.Second
}
