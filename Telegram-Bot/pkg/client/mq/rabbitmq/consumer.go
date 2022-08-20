package rabbitmq

import (
	"fmt"
	"log"
	"time"

	"github.com/Maksat-luci/Telegram-Bot/pkg/client/mq"
	"github.com/streadway/amqp"
)

//ConsumerConfig структура содержащий в себе базовую структуру с хостом именем и портом и паролем
type ConsumerConfig struct {
	BaseConfig
	// в крации нагружаемость консьюмера сообщениями
	PrefetchCount int
}

// rabbitMQConsumer структура содержит в себе ссылку на базовую структуру rabbitMQ а в нём все филды для комфортной работы
type rabbitMQConsumer struct {
	*rabbitMQBase
	reconnectCh   chan bool
	prefetchCount int
}

const (
	consumeDelay = 1 * time.Second
)

// NewRabbitMQConsumer конструктор который автоматически подключается к серверу RabbitMQ и возвращает интерфейс консьюмера
func NewRabbitMQConsumer(cfg ConsumerConfig) (mq.Consumer, error) {
	// получаем обьект и заполняем его  базовые филды
	consumer := &rabbitMQConsumer{
		prefetchCount: cfg.PrefetchCount,
		reconnectCh:   make(chan bool),
		rabbitMQBase: &rabbitMQBase{
			done: make(chan bool),
		},
	}
	// строим стринг и передаём из конфига нужные данные
	addr := fmt.Sprintf("amqp://%s:%s@%s:%s/", cfg.Username, cfg.Password, cfg.Host, cfg.Port)
	// устанавливаем соединение передаём туда адресс и заполняем основную структура RabbitMQBase
	err := consumer.connect(addr)
	// в случае ошибки валидируем её

	if err != nil {
		return nil, err
	}
	// записываем наш канал в список каналов основной структуры RabbitMQBase
	consumer.notifyReconnect(consumer.reconnectCh)
	// запускаем горутину которая в случае ошибки будет пытаться переподключиться
	go consumer.handleReconnect(addr)
	// возвращаем интерфейс консьюмера
	return consumer, nil
}

/*
Далее идут методы интерфейса КОНСЬЮМЕРА  для взаимодейтсвия с продюсером и так далее
*/

// забераем с очереди сообщения, далее структурируем его в канал сообщения, и возвращаем
func (r *rabbitMQConsumer) Consume(target string) (<-chan mq.Message, error) {
	// проверяем на подключение
	if !r.Connected() {
		return nil, errNotConnected
	}
	// получаем канал типа структуры с которого будет непрерывно записываться информация для наших консьюмеров
	// получаем сообщения с определенной очереди
	messages, err := r.consume(target)
	if err != nil {
		return nil, fmt.Errorf("failed to consume messages due %v", err)
	}
	// создаём канал типом структуры Message у которого филды ID uint64 and BODY []byte
	ch := make(chan mq.Message)
	// запускаем горутину
	go func() {
		// запускаем бесконечный цикл внутри горутины
		for {
			// c помощью селект проверяем каналы
			select {
			// считываем с канала данные
			case message, ok := <-messages:
				if !ok {
					// если ничего нет то засыпаем на 1 секунду
					time.Sleep(consumeDelay)
					continue
				}
				// создаём обьект структуры Message, заполнем его поля из полученного ранее структуры Delivery из канала message
				ch <- mq.Message{
					ID:   message.DeliveryTag,
					Body: message.Body,
				}
				// если с этого канала поступает плохая информация то мы переотправляем сообщение
			case <-r.reconnectCh:
				log.Print("Start to reconsume messages")
				for {
					// пытаемся получить канал со структурой для дальнейшей обработки
					messages, err = r.consume(target)
					if err != nil {
						break
					}
					// уведомляем о том что не удалось отправить сообщение
					log.Printf("failed to reconsume messages due %v", err)
				}

			case <-r.done:
				close(ch)
				return

			}
		}
	}()
	// возвращаем готовый канал сообщения который мы сконцтруировали и получили из канала значением которого является структура Delivery
	return ch, nil
}

// забирает данные с очереди которая указана в принемаемых параметрах, и возвращаем канал структуры Delivery с сообщением от продьюсера
func (r *rabbitMQConsumer)consume(target string) (<-chan amqp.Delivery, error) {
	// настраиваем Qos который отвечает за передачу сообщений consumeram до получение подтверждения от них
	err := r.ch.Qos(r.prefetchCount, 0, false)
	if err != nil {
		return nil, fmt.Errorf("failed to set QoS due %v", err)
	}
	// Consume метод который начинает немедленно доставлять сообщения консьюмером у него есть многие параметры которые нужно настраивать в зависимости от функционала
	// подключаемся к очереди и забираем отуда канал с данными Delivery
	messages, err := r.ch.Consume(
		target,
		"",
		false,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		return nil, err
	}
	// возвращаем канал с которого можно только считывать.
	return messages, nil
}

// Ack подтверждает-то что принял сообщение
func (r *rabbitMQConsumer)Ack(id uint64, multiple bool) error {
	// проверяем на подключение
	if !r.Connected() {
		return errNotConnected
	}
	// подтверждает доставку сообщения
	err := r.ch.Ack(id, multiple)
	if err != nil {
		return fmt.Errorf("failed to ack message with id %d due %v", id, err)
	}
	return nil
}

//Nack метод который уведомляет сервер о том что не удалось обработать сообщение
func (r *rabbitMQConsumer)Nack(id uint64, multiple bool, requeue bool) error {
	// проверка на подключение
	if !r.Connected() {
		return errNotConnected
	}
	// метод который говорит о том что сообщение не удалось обработать и ее необходимо доставить повторно или отбросить
	err := r.ch.Nack(id, multiple, requeue)
	if err != nil {
		return fmt.Errorf("failed to nack message with %d due %v", id, err)
	}
	return nil
}

// подобие Nack только для единичных сообщений
func (r *rabbitMQConsumer)Reject(id uint64, requeue bool) error {
	// проверка на подключение
	if !r.Connected() {
		return errNotConnected
	}
	// работает точно так же как Nack только reject уведомляет сервер только об одном сообщение а Nack имеет возможность уведомить о всех
	err := r.ch.Reject(id, requeue)
	if err != nil {
		return fmt.Errorf("failed to reject message with %d due %v", id, err)
	}
	return nil
}

// метод который закрывает все подключения и соединения в Rabbit MQ
func (r *rabbitMQConsumer)Close() error {
	if err := r.close(); err != nil {
		return err
	}

	return nil
}
