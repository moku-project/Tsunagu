package sandbox

import (
	"net"
	"os"
	"path/filepath"
	"runtime"
)

// afUnixConnectWorks reports whether an AF_UNIX socket file can be created and
// connected to inside dir. On healthy systems this always succeeds; on some
// Windows hosts connect() returns WSAEINVAL for paths under %USERPROFILE%\AppData.
func afUnixConnectWorks(dir string) bool {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return false
	}
	sock := filepath.Join(dir, "afunix-probe.sock")
	_ = os.Remove(sock)
	l, err := net.Listen("unix", sock)
	if err != nil {
		return false
	}
	defer func() {
		_ = l.Close()
		_ = os.Remove(sock)
	}()
	c, err := net.Dial("unix", sock)
	if err != nil {
		return false
	}
	_ = c.Close()
	return true
}

// usableSandboxTempDir returns a directory to point the sandbox JVM's TMP/TEMP
// at when the default temp dir cannot host an AF_UNIX socket, or "" when no
// override is needed (the common case, including all non-Windows hosts).
func usableSandboxTempDir() string {
	if runtime.GOOS != "windows" {
		return ""
	}
	if afUnixConnectWorks(os.TempDir()) {
		return ""
	}
	candidates := []string{}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		candidates = append(candidates, filepath.Join(home, ".tsunagu", "jvm-tmp"))
	}
	if sysDrive := os.Getenv("SystemDrive"); sysDrive != "" {
		candidates = append(candidates, filepath.Join(sysDrive+`\`, "Temp", "tsunagu-jvm"))
	}
	candidates = append(candidates, `C:\Temp\tsunagu-jvm`)
	for _, c := range candidates {
		if afUnixConnectWorks(c) {
			return c
		}
	}
	return ""
}
