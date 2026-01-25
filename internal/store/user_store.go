package store

import (
	"context"
	"github.com/RezaEskandarii/gofire/types"
)

// UserStore handles user-related database operations.
type UserStore interface {
	// Create adds a new user and returns its ID.
	Create(ctx context.Context, username, password string) (int64, error)

	// Find looks up a user matching the given username and password.
	Find(ctx context.Context, username, password string) (*types.User, error)

	// FindByUsername looks up a user matching the given username.
	FindByUsername(ctx context.Context, username string) (*types.User, error)

	// Delete removes a user by their ID.
	Delete(ctx context.Context, username string) error
}
