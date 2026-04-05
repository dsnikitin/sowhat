package gigachat

import (
	"github.com/dsnikitin/sowhat/internal/infrastructure/oauth"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

type Config struct {
	OAuth   *oauth.Config `envPrefix:"OAUTH_" yaml:"oauth"`
	RestAPI *apiConfig    `envPrefix:"REST_API_" yaml:"rest_api"`
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
	Chat          string `env:"CHAT" yaml:"chat"`
	GetEmbeddings string `env:"GET_EMBEDDINGS" yaml:"get_embeddings"`
	UploadFile    string `env:"UPLOAD_FILE" yaml:"upload_file"`
}

func (c apiConfig) Validate() error {
	return validation.ValidateStruct(&c,
		validation.Field(&c.Chat, validation.Required, is.URL),
		validation.Field(&c.GetEmbeddings, validation.Required, is.URL),
		validation.Field(&c.UploadFile, validation.Required, is.URL),
	)
}
