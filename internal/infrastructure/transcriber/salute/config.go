package salute

import (
	"github.com/dsnikitin/sowhat/internal/infrastructure/oauth"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

type Config struct {
	OAuth            *oauth.Config `envPrefix:"OAUTH_" yaml:"oauth"`
	RestAPI          *apiConfig    `envPrefix:"REST_API_" yaml:"rest_api"`
	SupportedFormats string        `env:"SUPPORTED_FORMATS" yaml:"supported_formats"`
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
	UploadData     string `env:"UPLOAD_DATA" yaml:"upload_data"`
	AsyncRecognize string `env:"ASYNC_RECOGNIZE" yaml:"async_recognize"`
	GetTaskStatus  string `env:"GET_TASK_STATUS" yaml:"get_task_status"`
	DownloadData   string `env:"DOWNLOAD_DATA" yaml:"download_data"`
}

func (c apiConfig) Validate() error {
	return validation.ValidateStruct(&c,
		validation.Field(&c.UploadData, validation.Required, is.URL),
		validation.Field(&c.AsyncRecognize, validation.Required, is.URL),
		validation.Field(&c.GetTaskStatus, validation.Required, is.URL),
		validation.Field(&c.DownloadData, validation.Required, is.URL),
	)
}
