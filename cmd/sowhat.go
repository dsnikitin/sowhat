package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/dsnikitin/sowhat/internal/app"
	"github.com/dsnikitin/sowhat/internal/config"
	"github.com/dsnikitin/sowhat/internal/pkg/logger"
)

var (
	buildVersion string = "n/a"
	buildDate    string = "n/a"
	buildCommit  string = "n/a"
)

func main() {
	log.Println("Build version:", buildVersion)
	log.Println("Build date:", buildDate)
	log.Println("Build commit:", buildCommit)

	cfg, err := config.New()
	if err != nil {
		logger.Log.Fatalw("Failed to init config", "error", err.Error())
	}

	if err = logger.Setup(cfg.Log); err != nil {
		logger.Log.Fatalw("Failed to setup logger", "error", err.Error())
	}

	application := app.New(cfg)
	go application.Run()

	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	<-shutdownSignal

	application.Shutdown()
}
