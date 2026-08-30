//go:build windows

package graph

import "golang.org/x/sys/windows"

func diskStats(path string) (totalBytes, freeBytes int64, err error) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, 0, err
	}
	var freeAvail, total, totalFree uint64
	if err := windows.GetDiskFreeSpaceEx(p, &freeAvail, &total, &totalFree); err != nil {
		return 0, 0, err
	}
	return int64(total), int64(freeAvail), nil
}
