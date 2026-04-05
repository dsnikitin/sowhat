package oauth

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

type Config struct {
	AuthToken        string        `env:"AUTH_TOKEN"`
	Scope            string        `env:"SCOPE" yaml:"scope"`
	Endpoint         string        `env:"ENDPOINT" yaml:"endpoint"`
	RefreshThreshold time.Duration `env:"REFRESH_THRESHOLD" yaml:"refresh_threshold"`
	Consumer         string        `env:"CONSUMER" yaml:"consumer"`
}

func (c Config) Validate() error {
	return validation.ValidateStruct(&c,
		validation.Field(&c.AuthToken, validation.Required),
		validation.Field(&c.Scope, validation.Required),
		validation.Field(&c.Endpoint, validation.Required, is.URL),
		validation.Field(&c.Endpoint, validation.Required),
	)
}
