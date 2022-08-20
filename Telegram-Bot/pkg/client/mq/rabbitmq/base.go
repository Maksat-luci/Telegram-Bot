package rabbitmq

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/streadway/amqp"
)

//BaseConfig базовая структура конфига//... базовая структура для структура Consumer and producer
type BaseConfig struct {
	Host     string
	Port     string
	Username string
	Password string
}

// константное время реконнекта
const (
	reconnectDelay = 5 * time.Second
)

// константные ошибки в случае не подключения
var (
	errNotConnected  = errors.New("no connection to RabbitMQ")
	errAlreadyClosed = errors.New("already connection closed to RabbitMQ")
)

// rabbitMQBase основная структура от которой наследуются consumer и producer
type rabbitMQBase struct {
	lock        sync.Mutex
	isConnected bool
	conn        *amqp.Connection
	ch          *amqp.Channel
	done        chan bool
	notifyClose chan *amqp.Error
	reconnects  []chan<- bool
}

// функция которая обьявляет очередь
func (r *rabbitMQBase) DeclareQueue(name string, durable, autoDelete, exclusive bool, args map[string]interface{}) error {
	// проверка на подключение
	if !r.Connected() {
		return errNotConnected
	}
	// метод который создаёт очередь с нужными настройками
	_, err := r.ch.QueueDeclare(
		name,
		durable,
		autoDelete,
		exclusive,
		false,
		args,
	)

	if err != nil {
		return fmt.Errorf("failed to declare queue due %v", err)
	}

	return nil
}

func (r *rabbitMQBase) handleReconnect(addr string) {
	// запускаем бесконечный цикл который будет чекать ошибки и в случае ошибки пытаться переподключиться
	for {
		select {
		// если получатель закончил работу то он отключает её
		case <-r.done:
			return
		case err := <-r.notifyClose:
			// если с канала прилетает ошибка то мы меняем статус подключения
			r.setConnected(false)

			if err == nil {
				return
			}
			// уведоляем о том что пытаемся переподключиться к RabbitMQ
			log.Print("Trying to reconnect to RabbitMQ...")
			// запускаем форик и проверяет подключился ли наш адрес снова
			for !r.boolConnect(addr) {
				log.Print("Failed to connect to RabbitMQ. Retrying...")
				// засыпает на 5 секунд для повторной проверки "подключился ли адррес или нет"
				time.Sleep(reconnectDelay)
			}

			// если всё хорошо то уведомляем через консоль об успешном поделючении
			log.Print("send signal about successfully reconnect to RabbitMQ")
			// меняем значение канала на true и добавляем в массив каналов
			for _, ch := range r.reconnects {
				ch <- true
			}
		}
	}
}

// методы уведомлений тех или иных случаев
func (r *rabbitMQBase) notifyReconnect(ch chan<- bool) {
	r.reconnects = append(r.reconnects, ch)
}

// методы уведомлений тех или иных случаев
func (r *rabbitMQBase) boolConnect(addr string) bool {
	return r.connect(addr) == nil
}

func (r *rabbitMQBase) connect(addr string) error {
	// проверка на подключение
	if r.Connected() {
		return nil
	}
	// получаем соединение и обьект для дальнейшей работы с RabbitMQ
	conn, err := amqp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ due %v", err)
	}
	//
	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel due %v", err)
	}
	// заполняем структуру базового RabbitMQbase
	r.conn = conn
	r.ch = ch
	r.notifyClose = make(chan *amqp.Error)
	// устанавливаем параметр что мы подключены
	r.setConnected(true)
	// в случае ошибки закрывает консьюмера и уведомляет всех продьюсеров что этот консьюмер сломался
	ch.NotifyClose(r.notifyClose)
	fmt.Println("Notify")
	// уведомляем в консоль что мы подключились
	log.Print("Successfully connected to RabbitMQ")

	return nil
}

// методы уведомлений тех или иных случаев
func (r *rabbitMQBase) setConnected(flag bool) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.isConnected = flag
}

// методы уведомлений тех или иных случаев
func (r *rabbitMQBase) Connected() bool {
	r.lock.Lock()
	defer r.lock.Unlock()

	return r.isConnected
}

// close закрывает все каналы и соединения происходит полное закрытие
func (r *rabbitMQBase) close() error {
	// проверка на подключение
	if !r.Connected() {
		return errAlreadyClosed
	}
	// закрывает канал
	if err := r.ch.Close(); err != nil {
		return fmt.Errorf("failed to close channel due %v", err)
	}
	// закрывает все соединения в RabbitMQ и всё что связано с ним
	if err := r.conn.Close(); err != nil {
		return fmt.Errorf("failed to close connection due %v", err)
	}
	// закрываем канал
	close(r.done)
	// меняеи значение isConnected на false
	r.setConnected(false)
	return nil
}
