//go:build unix

package mokusync

import (
	"runtime"
	"syscall"

	"golang.org/x/sys/unix"
)

// reuseAddr lets the responder share port 5353 with the system's own mDNS daemon.
func reuseAddr(_, _ string, c syscall.RawConn) error {
	var err error
	if ctlErr := c.Control(func(fd uintptr) {
		err = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)
		// BSD-derived systems (macOS, iOS) also need SO_REUSEPORT to share a multicast port.
		if err == nil && runtime.GOOS != "linux" && runtime.GOOS != "android" {
			err = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
		}
	}); ctlErr != nil {
		return ctlErr
	}
	return err
}
