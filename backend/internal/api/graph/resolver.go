package graph

import (
	"tsunagu/backend/internal/db/sqlcgen"
	"tsunagu/backend/internal/download"
	"tsunagu/backend/internal/sandbox"
	"tsunagu/backend/internal/sync"
)

type Resolver struct {
	Sy       *sync.Syncer
	Sc       *sandbox.SupervisedClient
	Dm       *download.Manager
	Q        *sqlcgen.Queries
	MediaDir string
	Name     string
	Version  string
	BuildTime string
}
