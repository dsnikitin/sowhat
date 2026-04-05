package telegram

import (
	"time"

	"github.com/dsnikitin/sowhat/internal/transport/telegram/handler"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type Config struct {
	AuthToken      string            `env:"AUTH_TOKEN"`
	PollerTimeout  time.Duration     `env:"POLLER_TIMEOUT" yaml:"poller_timeout"`
	RequestTimeout time.Duration     `env:"REQUEST_TIMEOUT" yaml:"request_timeout"`
	UI             *handler.UIConfig `envPrefix:"UI_" yaml:"ui"`
}

func (c Config) Validate() error {
	return validation.ValidateStruct(&c,
		validation.Field(&c.AuthToken, validation.Required),
		validation.Field(&c.RequestTimeout, validation.Required),
		validation.Field(&c.UI, validation.Required, validation.By(func(any) error {
			return c.UI.Validate()
		})),
	)
}
