package postgres

import (
	"context"
	"strings"
	"time"

	"github.com/dsnikitin/sowhat/internal/pkg/errx"
	"github.com/dsnikitin/sowhat/internal/pkg/logger"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
)

type DB struct {
	pool *pgxpool.Pool
	tx   pgx.Tx
	cfg  *Config
}

type Config struct {
	DSN            string `env:"DSN" yaml:"dsn"`
	MigrationsPath string `env:"MIGRATIONS_PATH" yaml:"migrations_path"`
}

func New(cfg *Config) (*DB, error) {
	pool, err := connect(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "connect to database")
	}

	return &DB{pool: pool, cfg: cfg}, nil
}

func QueryRow[T any](
	ctx context.Context, db *DB, sql string, args pgx.NamedArgs, fieldsPointer func(*T) []any,
) (obj T, err error) {
	if db.tx != nil {
		err = db.tx.QueryRow(ctx, sql, args).Scan(fieldsPointer(&obj)...)
	} else {
		err = db.pool.QueryRow(ctx, sql, args).Scan(fieldsPointer(&obj)...)
	}

	if err == pgx.ErrNoRows {
		return obj, errx.ErrNotFound
	}

	return obj, errors.Wrap(err, "scan db row")
}

func Query[T any](
	ctx context.Context, db *DB, sql string, args pgx.NamedArgs, fieldsPointer func(*T) []any,
) (objs []T, err error) {
	var rows pgx.Rows
	if db.tx != nil {
		rows, err = db.tx.Query(ctx, sql, args)
	} else {
		rows, err = db.pool.Query(ctx, sql, args)
	}

	if err != nil {
		return nil, errors.Wrap(err, "query")
	}
	defer rows.Close()

	for rows.Next() {
		var obj T
		if err := rows.Scan(fieldsPointer(&obj)...); err != nil {
			return nil, errors.Wrap(err, "scan db row")
		}

		objs = append(objs, obj)
	}

	if err = rows.Err(); err != nil {
		return nil, errors.Wrap(err, "iteration error")
	}

	return objs, nil
}

func (db *DB) ApplyMigrations() error {
	logger.Log.Info("Applying migrations...")

	replacer := strings.NewReplacer("postgres://", "pgx://", "postgresql://", "pgx://")

	m, err := migrate.New("file://"+db.cfg.MigrationsPath, replacer.Replace(db.cfg.DSN))
	if err != nil {
		return errors.Wrap(err, "create migrate instance")
	}
	defer m.Close()

	err = m.Up()
	switch err {
	case nil:
		logger.Log.Info("Migrations successfully applied")
	case migrate.ErrNoChange:
		logger.Log.Info("No new migrations for applying")
	default:
		return errors.Wrap(err, "up migrations")
	}

	return nil
}

func (db *DB) Tx(ctx context.Context, fn func(*DB) error) error {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return errors.Wrap(err, "begin tx")
	}

	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			logger.Log.Errorw("Failed to rollback tx", "error", err.Error())
		}
	}()

	pTx := &DB{pool: db.pool, tx: tx}

	if err = fn(pTx); err != nil {
		return errors.Wrap(err, "do job into tx")
	}

	return errors.Wrap(tx.Commit(ctx), "commit tx")
}

func (db *DB) Exec(ctx context.Context, sql string, args pgx.NamedArgs) (res pgconn.CommandTag, err error) {
	if db.tx != nil {
		res, err = db.tx.Exec(ctx, sql, args)
	} else {
		res, err = db.pool.Exec(ctx, sql, args)
	}

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return pgconn.CommandTag{}, errx.ErrAlreadyExists
		}

		return pgconn.CommandTag{}, errors.Wrap(err, "pool exec error")
	}

	return res, nil
}

func (db *DB) Close(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		db.pool.Close()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		select {
		case <-done:
		default:
			return errors.Wrap(ctx.Err(), "close database")
		}
	}

	return nil
}

func connect(cfg *Config) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(context.Background(), cfg.DSN)
	if err != nil {
		return nil, errors.Wrap(err, "new pgxpool")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, errors.Wrap(err, "ping db")
	}

	logger.Log.Infow(
		"Successfuly connected to PostgresDB",
		"name", pool.Config().ConnConfig.Database,
		"host", pool.Config().ConnConfig.Host,
		"port", pool.Config().ConnConfig.Port,
	)

	return pool, nil
}
