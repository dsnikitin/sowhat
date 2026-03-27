package logger

import (
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log = zap.Must(zap.NewDevelopment()).Sugar()

func Setup(level string, isDevelop bool) error {
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return errors.Wrap(err, "parse level")
	}

	var zapCfg zap.Config
	if isDevelop {
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
