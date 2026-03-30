package config

import (
	"flag"
	"fmt"
	"os"

	"dario.cat/mergo"
	"github.com/caarlos0/env/v11"
	tbot "github.com/dsnikitin/sowhat/internal/bot/telegram"
	pgdb "github.com/dsnikitin/sowhat/internal/infra/db/postgres"
	"github.com/dsnikitin/sowhat/internal/infra/llm/gigachat"
	"github.com/dsnikitin/sowhat/internal/infra/transcriber/salute"
	"github.com/dsnikitin/sowhat/internal/pkg/logger"
	"github.com/pkg/errors"
	"go.yaml.in/yaml/v2"
)

type Config struct {
	TeleBot        *tbot.Config     `envPrefix:"TELEGRAM_BOT_" yaml:"telegram_bot"`
	Salute         *salute.Config   `envPrefix:"SALUTE_" yaml:"salute"`
	GigaChat       *gigachat.Config `envPrefix:"GIGACHAT_" yaml:"gigachat"`
	PgDB           *pgdb.Config     `envPrefix:"POSTGRES_DB_" yaml:"postgres_db"`
	Log            *logger.Config   `envPrefix:"LOG_" yaml:"log"`
	ConfigFilePath string           `env:"CONFIG" yaml:"-"`
}

func New() (*Config, error) {
	cfg := &Config{
		TeleBot:  &tbot.Config{},
		Salute:   &salute.Config{},
		GigaChat: &gigachat.Config{},
		PgDB:     &pgdb.Config{},
		Log:      &logger.Config{},
	}

	flag.StringVar(&cfg.TeleBot.AuthToken, "t", "", "telegram bot api auth token")
	flag.StringVar(&cfg.PgDB.DSN, "d", "", "database dsn")
	flag.StringVar(&cfg.PgDB.MigrationsPath, "m", "./migrations", "path to migrations scripts")
	flag.StringVar(&cfg.Log.Level, "l", "info", "log level")
	flag.BoolVar(&cfg.Log.IsDevelop, "dev", false, "is develop environment")
	flag.StringVar(&cfg.ConfigFilePath, "c", "./config.yaml", "path to config file")
	flag.Parse()

	if cfg.ConfigFilePath != "" {
		if err := loadFromYAMLFile(cfg); err != nil {
			return nil, errors.Wrap(err, "load from yaml file")
		}
	}

	if err := env.Parse(cfg); err != nil {
		return nil, errors.Wrap(err, "parse envs")
	}

	fmt.Printf("CONFIG.TeleBot = %+v\n", cfg.TeleBot)
	fmt.Printf("CONFIG.Salute = %+v\n", cfg.Salute)
	fmt.Printf("CONFIG.GigaChat = %+v\n", cfg.GigaChat)
	fmt.Printf("CONFIG.PgDB = %+v\n", cfg.PgDB)
	fmt.Printf("CONFIG.Log = %+v\n", cfg.Log)
	fmt.Printf("CONFIG.ConfigFilePath = %+v\n", cfg.ConfigFilePath)

	return cfg, nil
}

// loadFromYAMLFile загружает конфигурацию из yaml-файла и записывает в пустые поля структуры Config.
func loadFromYAMLFile(cfg *Config) error {
	data, err := os.ReadFile(cfg.ConfigFilePath)
	if err != nil {
		return errors.Wrap(err, "read config file")
	}

	fileCfg := &Config{
		TeleBot:  &tbot.Config{},
		Salute:   &salute.Config{},
		GigaChat: &gigachat.Config{},
		PgDB:     &pgdb.Config{},
		Log:      &logger.Config{},
	}

	if err = yaml.Unmarshal(data, fileCfg); err != nil {
		return errors.Wrap(err, "unmarshal config file data")
	}

	// заполняет только пустые поля в cfg значениями из fileCfg
	err = mergo.Merge(cfg, fileCfg)
	return errors.Wrap(err, "merge configs")
}
