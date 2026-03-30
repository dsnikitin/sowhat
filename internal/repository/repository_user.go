package repository

import (
	"context"

	"github.com/dsnikitin/sowhat/internal/infra/db/postgres"
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
	INSERT INTO sowhat.users(id, name)
	VALUES(@id, @user)
`

func (r *UserRepository) Create(ctx context.Context, id int64, name string) error {
	_, err := r.db.Exec(ctx, createUserSQL, pgx.NamedArgs{"id": id, "name": name})
	return errors.Wrap(err, "exec")
}
