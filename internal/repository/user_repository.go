package repository

import (
	"context"
	"gofire/internal/models"
)

// UserRepository handles user-related database operations.
type UserRepository interface {
	// Create adds a new user and returns its ID.
	Create(ctx context.Context, username, password string) (int64, error)

	// Find looks up a user matching the given username and password.
	Find(ctx context.Context, username, password string) (*models.User, error)

	// FindByUsername looks up a user matching the given username.
	FindByUsername(ctx context.Context, username string) (*models.User, error)

	// Delete removes a user by their ID.
	Delete(ctx context.Context, username string) error
}
