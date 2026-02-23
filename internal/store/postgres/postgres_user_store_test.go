package postgres

import (
	"context"
	"database/sql"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/RezaEskandarii/gofire/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewPostgresUserStore(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	userStore := NewPostgresUserStore(db)
	require.NotNil(t, userStore)
	var _ store.UserStore = userStore
}

func TestPostgresUserStore_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	userStore := NewPostgresUserStore(db)
	ctx := context.Background()

	mock.ExpectQuery("DELETE FROM  gofire_schema.users").
		WithArgs("newuser").
		WillReturnRows(sqlmock.NewRows([]string{}))

	mock.ExpectQuery("INSERT INTO gofire_schema.users").
		WithArgs("newuser", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	id, err := userStore.Create(ctx, "newuser", "password123")
	require.NoError(t, err)
	assert.Equal(t, int64(1), id)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresUserStore_FindByUsername_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	userStore := NewPostgresUserStore(db)
	ctx := context.Background()

	mock.ExpectQuery("SELECT id, username FROM gofire_schema.users").
		WithArgs("nonexistent").
		WillReturnError(sql.ErrNoRows)

	user, err := userStore.FindByUsername(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, user)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresUserStore_FindByUsername_Found(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	userStore := NewPostgresUserStore(db)
	ctx := context.Background()

	mock.ExpectQuery("SELECT id, username FROM gofire_schema.users").
		WithArgs("admin").
		WillReturnRows(sqlmock.NewRows([]string{"id", "username"}).AddRow(1, "admin"))

	user, err := userStore.FindByUsername(ctx, "admin")
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, int64(1), user.ID)
	assert.Equal(t, "admin", user.Username)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresUserStore_Delete(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	userStore := NewPostgresUserStore(db)
	ctx := context.Background()

	mock.ExpectExec("DELETE FROM gofire_schema.users").
		WithArgs("todelete").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = userStore.Delete(ctx, "todelete")
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresUserStore_Delete_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	userStore := NewPostgresUserStore(db)
	ctx := context.Background()

	mock.ExpectExec("DELETE FROM gofire_schema.users").
		WithArgs("nonexistent").
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = userStore.Delete(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no user found")
	assert.NoError(t, mock.ExpectationsWereMet())
}
