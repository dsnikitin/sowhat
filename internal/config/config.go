package config

import (
	"encoding/json"
	"flag"
	"os"

	"dario.cat/mergo"
	"github.com/caarlos0/env/v11"
	"github.com/pkg/errors"
)

type Config struct {
	BotAuthToken   string `env:"BOT_AUTH_TOKEN" json:"-"`
	DataBaseDSN    string `env:"DATABASE_DSN" json:"-"`
	MigrationsPath string `env:"MIGRATIONS_PATH" json:"migrations_path"`
	LogLevel       string `env:"LOG_LEVEL" json:"log_level"`
	IsDevelop      bool   `env:"IS_DEVELOP" json:"is_develop"`
	ConfigFilePath string `env:"CONFIG" json:"-"`
}

func New() (*Config, error) {
	cfg := &Config{}

	flag.StringVar(&cfg.BotAuthToken, "t", "", "telegram bot api auth token")
	flag.StringVar(&cfg.DataBaseDSN, "d", "", "database dsn")
	flag.StringVar(&cfg.MigrationsPath, "m", "./migrations", "path to migrations scripts")
	flag.StringVar(&cfg.LogLevel, "l", "info", "log level")
	flag.BoolVar(&cfg.IsDevelop, "dev", false, "is develop environment")
	flag.StringVar(&cfg.ConfigFilePath, "c", "./config.json", "path to config file")
	flag.Parse()

	if cfg.ConfigFilePath != "" {
		if err := loadFromJSONFile(cfg); err != nil {
			return nil, errors.Wrap(err, "load from json file")
		}
	}

	if err := env.Parse(cfg); err != nil {
		return nil, errors.Wrap(err, "parse envs")
	}

	return cfg, nil
}

// loadFromJSONFile загружает конфигурацию из json-файла и записывает в пустые поля структуры Config.
func loadFromJSONFile(cfg *Config) error {
	data, err := os.ReadFile(cfg.ConfigFilePath)
	if err != nil {
		return errors.Wrap(err, "read config file")
	}

	fileCfg := &Config{}
	if err = json.Unmarshal(data, fileCfg); err != nil {
		return errors.Wrap(err, "unmarshal config file data")
	}

	// заполняет только пустые поля в cfg значениями из fileCfg
	err = mergo.Merge(cfg, fileCfg)
	return errors.Wrap(err, "merge configs")
}
