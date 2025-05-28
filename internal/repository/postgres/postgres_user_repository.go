package postgres

import (
	"context"
	"database/sql"
	"errors"
	"gofire/internal/models"
	"gofire/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type postgresUserRepository struct {
	db *sql.DB
}

// NewPostgresUserRepository creates a new UserRepository with a DB connection
func NewPostgresUserRepository(db *sql.DB) repository.UserRepository {
	return &postgresUserRepository{db: db}
}

func (r *postgresUserRepository) Create(ctx context.Context, username, password string) (int64, error) {
	var id int64
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}
	query := `INSERT INTO gofire_schema.users (username, password) VALUES ($1, $2) RETURNING id`
	err = r.db.QueryRowContext(ctx, query, username, hashedPassword).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *postgresUserRepository) Find(ctx context.Context, username, password string) (*models.User, error) {
	query := `SELECT id, username, password FROM gofire_schema.users WHERE username = $1 AND password = $2`
	user := &models.User{}
	err := r.db.QueryRowContext(ctx, query, username, password).Scan(&user.ID, &user.Username, &user.Password)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // user not found
		}
		return nil, err
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, errors.New("user not found")
	}
	user.Password = ""
	return user, nil
}
func (r *postgresUserRepository) FindByUsername(ctx context.Context, username string) (*models.User, error) {
	query := `SELECT id, username, password FROM gofire_schema.users WHERE username = $1`
	user := &models.User{}
	err := r.db.QueryRowContext(ctx, query, username).Scan(&user.ID, &user.Username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // user not found
		}
		return nil, err
	}
	return user, nil
}

func (r *postgresUserRepository) Delete(ctx context.Context, username string) error {
	query := `DELETE FROM gofire_schema.users WHERE username = $1`
	result, err := r.db.ExecContext(ctx, query, username)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("no user found to delete")
	}
	return nil
}
