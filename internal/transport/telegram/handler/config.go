package handler

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

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
