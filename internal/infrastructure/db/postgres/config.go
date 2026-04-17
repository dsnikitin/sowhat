package postgres

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type Config struct {
	DSN            string `env:"DSN"`
	MigrationsPath string `env:"MIGRATIONS_PATH" yaml:"migrations_path"`
}

func (c Config) Validate() error {
	return validation.ValidateStruct(&c,
		validation.Field(&c.DSN, validation.Required),
		validation.Field(&c.MigrationsPath, validation.Required),
	)
}
