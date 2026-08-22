package graph

import (
	"tsunagu/backend/internal/sandbox"
	"tsunagu/backend/internal/sync"
)

type Resolver struct {
	Sy *sync.Syncer
	Sc *sandbox.SupervisedClient
}
