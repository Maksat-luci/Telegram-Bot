package main

import (
	"flag"
	"log"

	"github.com/Maksat-luci/Telegram-Bot/internal"
	"github.com/Maksat-luci/Telegram-Bot/internal/config"
	"github.com/Maksat-luci/Telegram-Bot/pkg/logging"
)

var cfgPath string

func init() {
	flag.StringVar(&cfgPath, "config", "configs/dev.yml", "config file path")
}

func main() {
	flag.Parse()

	log.Print("config initializing")
	cfg := config.GetConfig(cfgPath)

	log.Print("logger initializng")
	logging.Init(cfg.AppConfig.LogLevel)
	logger := logging.GetLogger()

	logger.Println("Creating Application")
	app, err := internal.NewApp(logger, cfg)
	if err != nil {
		logger.Fatal(err)
	}

	logger.Println("Running Application")
	app.Run()
}
