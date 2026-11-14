package graph

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"sync"

	"tsunagu/backend/internal/db/sqlcgen"
)

const keyInstanceID = "instance_id"

var instanceMu sync.Mutex

// instanceID returns this server's permanent id, creating it the first time a client asks.
func (r *Resolver) instanceID(ctx context.Context) (string, error) {
	instanceMu.Lock()
	defer instanceMu.Unlock()
	id, err := r.Q.GetSetting(ctx, keyInstanceID)
	if err == nil && id != "" {
		return id, nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	id = hex.EncodeToString(b)
	if err := r.Q.SetSetting(ctx, sqlcgen.SetSettingParams{Key: keyInstanceID, Value: id}); err != nil {
		return "", err
	}
	return id, nil
}
