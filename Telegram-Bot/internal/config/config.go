package config

import (
	"log"
	"sync"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config основная структура конфигурации
type Config struct {
	IsDebug       bool `yaml:"is_debug" env:"ST_BOT_IS_DEBUG" env-default:"false"`
	IsDevelopment bool `yaml:"is_development" env:"ST_BOT_IS_DEVELOPMENT" env-default:"false"`
	Telegram      struct {
		Token string `yaml:"token" env:"ST_BOT_TELEGRAM_TOKEN"  env-default:"5497403137:AAE8gjAgTjUqzEObDSxf2PxKVriPurMAJb0"`
	}
	RabbitMQ struct {
		Host     string `yaml:"host" env:"ST_BOT_RABBIT_HOST" `
		Port     string `yaml:"port" env:"ST_BOT_RABBIT_PORT" `
		Username string `yaml:"username" env:"ST_BOT_RABBIT_USERNAME" `
		Password string `yaml:"password" env:"ST_BOT_RABBIT_PASSWORD" `
		Consumer struct {
			// Youtube            string `yaml:"youtube" env:"ST_BOT_RABBIT_CONSUMER_YOUTUBE" `
			// Imgur              string `yaml:"imgur" env:"ST_BOT_RABBIT_CONSUMER_IMGUR" `
			Queue              string `yaml:"queue"`
			MessagesBufferSize int    `yaml:"messages_buff_size" env:"ST_BOT_RABBIT_CONSUMER_MBS" env-default:"100"`
		} `yaml:"consumer"`
		Producer struct {
			// Youtube string `yaml:"youtube" env:"ST_BOT_RABBIT_PRODUCER_YOUTUBE" `
			// Imgur   string `yaml:"imgur" env:"ST_BOT_RABBIT_PRODUCER_IMGUR" `
			Queue string `yaml:"queue"`
		} `yaml:"producer"`
	}`yaml:"rabbit_mq"`
	Imgur struct {
		RefreshToken string `yaml:"refresh_token"`
		AccessToken  string `yaml:"access_token"`
		ClientID     string `yaml:"client_id"`
		ClientSecret string `yaml:"client_secret"`
		URL          string `yaml:"url"`
	} `yaml:"imgur"`

	AppConfig AppConfig `yaml:"app"`
}

//AppConfig струтура приложения именно конфигурации
type AppConfig struct {
	Eventworkers int `yaml:"event_worker" env-default:"3"`
	EventWorkers struct {
		Youtube int `yaml:"youtube" env:"ST_BOT_EVENT_WORKERS_YT" env-default:"3"`
		Imgur   int `yaml:"imgur" env:"ST_BOT_EVENT_WORKERS_IMGUR" env-default:"3"`
	} `yaml:"event_workers"`
	LogLevel string `yaml:"log_level" env:"ST_BOT_LOG_LEVEL" env-default:"error"`
}

var instance *Config
var once sync.Once

//GetConfig функция которая срабатывает один раз, и возвращает конфиг
func GetConfig(path string) *Config {
	once.Do(func() {
		log.Printf("read application config in path %s", path)

		instance = &Config{}

		if err := cleanenv.ReadConfig(path, instance); err != nil {
			help, _ := cleanenv.GetDescription(instance, nil)
			log.Print(help)
			log.Fatal(err)
		}
	})
	return instance
}
