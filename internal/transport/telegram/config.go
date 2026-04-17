package telegram

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type Config struct {
	AuthToken      string        `env:"AUTH_TOKEN"`
	PollerTimeout  time.Duration `env:"POLLER_TIMEOUT" yaml:"poller_timeout"`
	RequestTimeout time.Duration `env:"REQUEST_TIMEOUT" yaml:"request_timeout"`
	UI             *UIConfig     `envPrefix:"UI_" yaml:"ui"`
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

type UIConfig struct {
	SummaryMaxLength    int    `env:"SUMMARY_MAX_LENGTH" yaml:"summary_max_length"`
	TranscriptMaxLength int    `env:"TRANSCRIPT_MAX_LENGTH" yaml:"transcript_max_length"`
	MeetingsPerPage     int    `env:"MEETINGS_PER_PAGE" yaml:"meetings_per_page"`
	DateFormat          string `env:"DATE_FORMAT" yaml:"date_format"`
}

func (c UIConfig) Validate() error {
	return validation.ValidateStruct(&c,
		validation.Field(&c.SummaryMaxLength, validation.Min(1)),
		validation.Field(&c.TranscriptMaxLength, validation.Min(1)),
		validation.Field(&c.MeetingsPerPage, validation.Min(1)),
		validation.Field(&c.DateFormat, validation.Required),
	)
}
