//go:build tools

// Package mobile pins what gomobile binds, so `go mod tidy` keeps it in go.mod.
// Moku-Mobile's scripts/build-tsunagu.sh builds the Android library from here.
package mobile

import _ "tsunagu/backend/embedded"
