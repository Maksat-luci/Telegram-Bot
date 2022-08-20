package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Maksat-luci/Telegram-Bot/internal/config"
	"github.com/Maksat-luci/Telegram-Bot/internal/events"
	"github.com/Maksat-luci/Telegram-Bot/internal/service"
	"github.com/Maksat-luci/Telegram-Bot/pkg/client/imgur"
	"github.com/Maksat-luci/Telegram-Bot/pkg/client/mq"
	"github.com/Maksat-luci/Telegram-Bot/pkg/client/mq/rabbitmq"
	"github.com/Maksat-luci/Telegram-Bot/pkg/logging"
	tele "gopkg.in/telebot.v3"
)

type app struct {
	cfg          *config.Config
	logger       *logging.Logger
	httpServer   *http.Server
	imgurService service.ImgurService
	bot          *tele.Bot
	producer     mq.Producer
}

// App интерфейс для работы со структурой
type App interface {
	Run()
}

// NewApp конструктор интерфейса который имплементировала структура
func NewApp(logger *logging.Logger, cfg *config.Config) (App, error) {
	client := http.Client{}

	imgurClient := imgur.NewClient(cfg.Imgur.URL, cfg.Imgur.AccessToken, cfg.Imgur.ClientID, &client)
	imgurService := service.NewImgurService(imgurClient, logger)

	return &app{
		cfg:          cfg,
		logger:       logger,
		imgurService: imgurService,
	}, nil
}

func (a *app) Run() {
	a.startConsume()
	a.startBot()
	a.bot.Start()


}
func (a *app) startConsume() {
	a.logger.Info("start Consuming")
	// используем конструктор консьюмера и получаем интерфейс консьюмера
	consumer, err := rabbitmq.NewRabbitMQConsumer(rabbitmq.ConsumerConfig{
		BaseConfig: rabbitmq.BaseConfig{
			Host:     a.cfg.RabbitMQ.Host,
			Port:     a.cfg.RabbitMQ.Port,
			Username: a.cfg.RabbitMQ.Username,
			Password: a.cfg.RabbitMQ.Password,
		},
		PrefetchCount: a.cfg.RabbitMQ.Consumer.MessagesBufferSize,
	})
	// валидируем на ошибки
	if err != nil {
		a.logger.Fatal(err)
	}
	// получаем интерфейс продьюсера с помошью конструктора
	producer, err := rabbitmq.NewRabbitMQProducer(rabbitmq.ProducerConfig{
		BaseConfig: rabbitmq.BaseConfig{
			Host:     a.cfg.RabbitMQ.Host,
			Port:     a.cfg.RabbitMQ.Port,
			Username: a.cfg.RabbitMQ.Username,
			Password: a.cfg.RabbitMQ.Password,
		},
	})
	// валидируем на ошибки
	if err != nil {
		a.logger.Fatal(err)
	}
	// отправляем в консьюмер очередь, затем считываем с очереди сообщения переконвертируем канал Delivery в наш канал Message и возвращаем его, всё это происходит паралельно
	messages, err := consumer.Consume(a.cfg.RabbitMQ.Consumer.Queue)
	// валидируем на ошибки
	if err != nil {
		a.logger.Fatal(err)
	}

	// запускаем цикл по количеству событий, указанному в конфиге
	for i := 0; i < a.cfg.AppConfig.Eventworkers; i++ {
		// с помошью конструктора получаем обьект Воркера
		worker := events.NewWorker(i, consumer, producer, messages, a.logger,a.bot)
		// запускаем горутину процесс
		go worker.Proccess()
		// уведомляем о старте воркера
		a.logger.Infof("Event Worker #%d started", i)
	}
	a.producer = producer
}

func (a *app) startBot() {
	a.logger.Info()

	pref := tele.Settings{
		Token:   a.cfg.Telegram.Token,
		Poller:  &tele.LongPoller{Timeout: 10 * time.Second},
		Verbose: false,
		OnError: a.OnBotError,
	}
	var botErr error
	a.bot, botErr = tele.NewBot(pref)
	if botErr != nil {
		a.logger.Fatal(botErr)
		return
	}

	a.bot.Handle("/yt", func(c tele.Context) error {
		sendText := "Твой трек не найден"
		trackName := c.Message().Payload
		request := events.SearchTrackRequest{
			RequestID: fmt.Sprintf("%d", c.Sender().ID),
			Name:      trackName,
		}

		marshal, err := json.Marshal(request)
		if err != nil {
			c.Send("Не удалось сконвертировать ваш запрос")
		}

		if err := a.producer.Publish(a.cfg.RabbitMQ.Producer.Queue, marshal); err != nil {
			return c.Send("не удалось обработать ваш запрос, по следующей причине ", err)
		}

		sendText = "какой-то трек, текст заглушка, проблемы с Google service Youtube Data API"
		return c.Send(fmt.Sprintf("Вот твой трек: %s", sendText))
	})

	a.bot.Handle(tele.OnPhoto, func(c tele.Context) error {
		photo := c.Message().Photo

		file, err := a.bot.File(&photo.File)

		if err != nil {
			return c.Send("Не удалось скачать изображение!")
		}

		defer file.Close()

		buf := new(bytes.Buffer)
		_, err = buf.ReadFrom(file)

		if err != nil {
			return c.Send("Не удалось скачать изображение!")

		}

		if buf.Len() > 10_048_576 {
			return c.Send("Лимит 10mb!")
		}

		var image string
		image, err = a.imgurService.ShareImage(context.Background(), buf.Bytes())
		if err != nil {
			return c.Send("Не удалось залить изображение!")
		}
		return c.Send(image)
	})

}

func (a *app) OnBotError(err error, ctx tele.Context) {
	a.logger.Error(err)
}
