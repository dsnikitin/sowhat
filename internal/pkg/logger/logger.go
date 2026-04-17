package logger

import (
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log = zap.Must(zap.NewDevelopment()).Sugar()

type Config struct {
	Level     string `env:"LEVEL" yaml:"level"`
	IsDevelop bool   `env:"IS_DEVELOP" yaml:"is_develop"`
}

func Setup(cfg *Config) error {
	lvl, err := zap.ParseAtomicLevel(cfg.Level)
	if err != nil {
		return errors.Wrap(err, "parse level")
	}

	var zapCfg zap.Config
	if cfg.IsDevelop {
		zapCfg = zap.NewDevelopmentConfig()
	} else {
		zapCfg = zap.NewProductionConfig()
	}

	zapCfg.Level = lvl
	log, err := zapCfg.Build(zap.AddStacktrace(zapcore.ErrorLevel))
	if err != nil {
		return errors.Wrap(err, "build logger")
	}

	Log = log.Sugar()
	return nil
}
