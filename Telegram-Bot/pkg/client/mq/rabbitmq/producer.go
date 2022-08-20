package rabbitmq

import (
	"fmt"

	"github.com/Maksat-luci/Telegram-Bot/pkg/client/mq"
	"github.com/streadway/amqp"
)
// ProducerConfig который внутри себя содержит базовую конфигурацию
type ProducerConfig struct {
	BaseConfig
}
// rabbitMQProducer структура которая содержит внутри себя ссылку на базовую структуру rabbitmqBase
type rabbitMQProducer struct {
	*rabbitMQBase
}

// NewRabbitMQProducer конструктор который подключается к серверу RabbitMQ возвращает продьюсера
func NewRabbitMQProducer(cfg ProducerConfig) (mq.Producer, error) {
	// создаём обьект структуры продьюсера 
	producer := &rabbitMQProducer{
		rabbitMQBase: &rabbitMQBase{
			done: make(chan bool),
		},
	}
	// строим стрингу и получаем адрес который вылеплен из конфига
	addr := fmt.Sprintf("amqp://%s:%s@%s:%s/", cfg.Username, cfg.Password, cfg.Host, cfg.Port)
	// подключаемся к RabbitMQ
	err := producer.connect(addr)

	if err != nil {
		return nil, err
	}
	// запускаем горутину которая проверяет на наличие ошибок каналы и после этого пытается переподключиться
	go producer.handleReconnect(addr)

	// возвращаем продьюсера
	return producer, nil 
}

// функция отправки сообщения в очередь
func (r *rabbitMQProducer) Publish(target string, body []byte) error {
	// проверка на подключение
	if !r.Connected() {
		return errNotConnected
	}
	// отправляет сообщение в очередь
	err := r.ch.Publish(
		"",
		target,
		false,
		false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType: "text/plain",
			Body: body,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish message dur %v", err)
	}
	return nil
}
// аналогичный метод закрытия всех каналов и соединений 
func (r *rabbitMQProducer) Close() error {
	if err := r.Close(); err != nil {
		return err
	}
	return nil
}