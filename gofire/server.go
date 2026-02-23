package gofire

import (
	"context"
	"errors"
	"github.com/RezaEskandarii/gofire/types/config"
)

// AddServer is deprecated. Use jobmanager.New instead, which creates a single
// dependency container and starts all services (job processors, web dashboard) in one call.
//
// Deprecated: Server startup is now integrated into jobmanager.New. Calling AddServer
// separately created duplicate database connections and is no longer supported.
func AddServer(ctx context.Context, cfg *config.GofireConfig) error {
	_ = ctx
	_ = cfg
	return errors.New("deprecated: use jobmanager.New instead - it creates a single container and starts all services")
}
