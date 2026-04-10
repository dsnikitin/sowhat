package repository

import (
	"context"

	"github.com/dsnikitin/sowhat/internal/consts/platform"
	"github.com/dsnikitin/sowhat/internal/infrastructure/db/postgres"
	"github.com/dsnikitin/sowhat/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
)

type UserRepository struct {
	db *postgres.DB
}

func NewUserRepository(db *postgres.DB) *UserRepository {
	return &UserRepository{db: db}
}

const createUserSQL = `
	INSERT INTO sowhat.users(external_id, name, platform)
	VALUES(@externalID, @name, @platform)
`

func (r *UserRepository) CreateUser(ctx context.Context, externalID, name string, pl platform.Type) error {
	_, err := r.db.Exec(ctx, createUserSQL, pgx.NamedArgs{"externalID": externalID, "name": name, "platform": pl})
	return errors.Wrap(err, "exec")
}

const getUserSQL = `
	SELECT id, external_id, name, platform, registered_at
	FROM sowhat.users
	WHERE id = @id AND platform = @platform
`

func (r *UserRepository) GetUserByID(ctx context.Context, userID int64, pt platform.Type) (models.User, error) {
	args := pgx.NamedArgs{"id": userID, "platform": pt}
	fieldsPointer := func(u *models.User) []any { return u.FieldPointers() }

	user, err := postgres.QueryRow(ctx, r.db, getUserSQL, args, fieldsPointer)
	return user, errors.Wrap(err, "query row")
}

const getUserByExternalIDSQL = `
	SELECT id, external_id, name, platform, registered_at
	FROM sowhat.users
	WHERE external_id = @externalID AND platform = @platform
`

func (r *UserRepository) GetUserByExternalID(ctx context.Context, externalID string, pt platform.Type) (models.User, error) {
	args := pgx.NamedArgs{"externalID": externalID, "platform": pt}
	fieldsPointer := func(u *models.User) []any { return u.FieldPointers() }

	user, err := postgres.QueryRow(ctx, r.db, getUserByExternalIDSQL, args, fieldsPointer)
	return user, errors.Wrap(err, "query row")
}
