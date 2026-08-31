package flaresolverr

import "runtime"

const PinnedVersion = "v3.5.0"

type asset struct {
	url     string
	archive string
	binRel  string
}

var assets = map[string]asset{
	"linux/amd64": {
		url:     "https://github.com/FlareSolverr/FlareSolverr/releases/download/" + PinnedVersion + "/flaresolverr_linux_x64.tar.gz",
		archive: "tar.gz",
		binRel:  "flaresolverr/flaresolverr",
	},
	"windows/amd64": {
		url:     "https://github.com/FlareSolverr/FlareSolverr/releases/download/" + PinnedVersion + "/flaresolverr_windows_x64.zip",
		archive: "zip",
		binRel:  "flaresolverr/flaresolverr.exe",
	},
}

func platformAsset() (asset, bool) {
	a, ok := assets[runtime.GOOS+"/"+runtime.GOARCH]
	return a, ok
}

// SupportedOnPlatform reports whether a managed FlareSolverr binary exists for
// this OS/arch. macOS and linux/arm64 have no upstream build.
func SupportedOnPlatform() bool {
	_, ok := platformAsset()
	return ok
}
