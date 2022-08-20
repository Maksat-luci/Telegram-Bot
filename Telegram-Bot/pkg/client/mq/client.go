package mq

import "io"
//MessageQueue интерфейс очереди
type MessageQueue interface {
	io.Closer
	DeclareQueue(name string, durable, autoDelete, exclusive bool, args map[string]interface{}) error
}
// Producer интерфейс продьюсера
type Producer interface {
	MessageQueue
	Publish(target string, body []byte) error
}
// Consumer интерфейс  консьюмера
type Consumer interface {
	MessageQueue
	Consume(target string) (<-chan Message, error)
	Ack(id uint64, multiple bool) error
	Nack(id uint64, multiple bool, requeue bool) error
	Reject(id uint64, requeue bool) error
}

// Message структура Сообщения
type Message struct {
	ID   uint64
	Body []byte
}
