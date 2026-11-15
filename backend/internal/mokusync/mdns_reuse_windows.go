package mokusync

import (
	"syscall"

	"golang.org/x/sys/windows"
)

// reuseAddr lets the responder share port 5353 with Windows' own mDNS service.
func reuseAddr(_, _ string, c syscall.RawConn) error {
	var err error
	if ctlErr := c.Control(func(fd uintptr) {
		err = windows.SetsockoptInt(windows.Handle(fd), windows.SOL_SOCKET, windows.SO_REUSEADDR, 1)
	}); ctlErr != nil {
		return ctlErr
	}
	return err
}
