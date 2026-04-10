package gigachat

import (
	"github.com/dsnikitin/sowhat/internal/infrastructure/oauth"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

type Config struct {
	OAuth       *oauth.Config `envPrefix:"OAUTH_" yaml:"oauth"`
	RestAPI     *apiConfig    `envPrefix:"REST_API_" yaml:"rest_api"`
	CanBeMyself bool          `env:"CAN_BE_MYSELF" yaml:"can_be_myself"`
}

func (c Config) Validate() error {
	return validation.ValidateStruct(&c,
		validation.Field(&c.OAuth, validation.Required, validation.By(func(any) error {
			return c.OAuth.Validate()
		})),
		validation.Field(&c.RestAPI, validation.Required, validation.By(func(any) error {
			return c.RestAPI.Validate()
		})),
	)
}

type apiConfig struct {
	Completions string `env:"COMPLETIONS" yaml:"completions"`
	UploadFile  string `env:"UPLOAD_FILE" yaml:"upload_file"`
}

func (c apiConfig) Validate() error {
	return validation.ValidateStruct(&c,
		validation.Field(&c.Completions, validation.Required, is.URL),
		validation.Field(&c.UploadFile, validation.Required, is.URL),
	)
}
