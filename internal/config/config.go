package config

import (
	"flag"
	"fmt"
	"os"

	"dario.cat/mergo"
	"github.com/caarlos0/env/v11"
	"github.com/dsnikitin/sowhat/internal/infrastructure/db/postgres"
	"github.com/dsnikitin/sowhat/internal/infrastructure/llm/gigachat"
	"github.com/dsnikitin/sowhat/internal/infrastructure/oauth"
	"github.com/dsnikitin/sowhat/internal/infrastructure/transcriber/salute"
	"github.com/dsnikitin/sowhat/internal/pkg/logger"
	telebot "github.com/dsnikitin/sowhat/internal/transport/telegram"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pkg/errors"
	"go.yaml.in/yaml/v2"
)

type Config struct {
	TeleBot        *telebot.Config  `envPrefix:"TELEGRAM_BOT_" yaml:"telegram_bot"`
	SaluteSpeech   *salute.Config   `envPrefix:"SALUTE_SPEECH_" yaml:"salute_speech"`
	GigaChat       *gigachat.Config `envPrefix:"GIGACHAT_" yaml:"gigachat"`
	PgDB           *postgres.Config `envPrefix:"POSTGRES_DB_" yaml:"postgres_db"`
	Log            *logger.Config   `envPrefix:"LOG_" yaml:"log"`
	ConfigFilePath string           `env:"CONFIG_FILE"`
	OAuth          []*oauth.Config
}

func New() (*Config, error) {
	cfg := &Config{
		TeleBot:      &telebot.Config{},
		SaluteSpeech: &salute.Config{},
		GigaChat:     &gigachat.Config{},
		PgDB:         &postgres.Config{},
		Log:          &logger.Config{},
	}

	flag.StringVar(&cfg.ConfigFilePath, "c", "config.yaml", "path to config yaml-file")
	flag.Parse()

	if cfg.ConfigFilePath != "" {
		if err := loadFromYAMLFile(cfg); err != nil {
			return nil, errors.Wrap(err, "load from yaml file")
		}
	}

	if err := env.Parse(cfg); err != nil {
		return nil, errors.Wrap(err, "parse envs")
	}

	// TODO убрать
	fmt.Printf("CONFIG.TeleBot = %+v\n", cfg.TeleBot)
	fmt.Printf("CONFIG.Salute = %+v\n", cfg.SaluteSpeech)
	fmt.Printf("CONFIG.Salute.API = %+v\n", cfg.SaluteSpeech.RestAPI)
	fmt.Printf("CONFIG.GigaChat = %+v\n", cfg.GigaChat)
	fmt.Printf("CONFIG.GigaChat.API = %+v\n", cfg.GigaChat.RestAPI)
	fmt.Printf("CONFIG.PgDB = %+v\n", cfg.PgDB)
	fmt.Printf("CONFIG.Log = %+v\n", cfg.Log)
	fmt.Printf("CONFIG.ConfigFilePath = %+v\n", cfg.ConfigFilePath)
	fmt.Printf("CONFIG.OAuth = %+v\n", cfg.OAuth)

	if err := cfg.validate(); err != nil {
		return nil, errors.Wrap(err, "validate config")
	}

	cfg.OAuth = append(cfg.OAuth, cfg.SaluteSpeech.OAuth, cfg.GigaChat.OAuth)

	return cfg, nil
}

func (c Config) validate() error {
	return validation.ValidateStruct(&c,
		validation.Field(&c.TeleBot, validation.By(func(any) error {
			return c.TeleBot.Validate()
		})),
		validation.Field(&c.SaluteSpeech, validation.By(func(any) error {
			return c.SaluteSpeech.Validate()
		})),
		validation.Field(&c.GigaChat, validation.By(func(any) error {
			return c.GigaChat.Validate()
		})),
		validation.Field(&c.PgDB, validation.By(func(any) error {
			return c.PgDB.Validate()
		})),
	)
}

// loadFromYAMLFile загружает конфигурацию из yaml-файла и записывает в пустые поля структуры Config.
func loadFromYAMLFile(cfg *Config) error {
	data, err := os.ReadFile(cfg.ConfigFilePath)
	if err != nil {
		return errors.Wrap(err, "read config file")
	}

	fileCfg := &Config{
		TeleBot:      &telebot.Config{},
		SaluteSpeech: &salute.Config{},
		GigaChat:     &gigachat.Config{},
		PgDB:         &postgres.Config{},
		Log:          &logger.Config{},
	}

	if err = yaml.Unmarshal(data, fileCfg); err != nil {
		return errors.Wrap(err, "unmarshal config file data")
	}

	// заполняет только пустые поля в cfg значениями из fileCfg
	err = mergo.Merge(cfg, fileCfg)
	return errors.Wrap(err, "merge configs")
}
