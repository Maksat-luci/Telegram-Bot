package events

import (
	"encoding/json"
	"strconv"

	"github.com/Maksat-luci/Telegram-Bot/pkg/client/mq"
	"github.com/Maksat-luci/Telegram-Bot/pkg/logging"
	tele "gopkg.in/telebot.v3"
)

//worker структура которая содержит в себе и продьюсера и консьюмера
type worker struct {
	id            int
	client        mq.Consumer
	producer      mq.Producer
	responseQueue string
	messages      <-chan mq.Message
	logger        *logging.Logger
	bot           *tele.Bot
}

//Worker интерфейс с методом процесс
type Worker interface {
	Proccess()
}

//NewWorker конструктор который возвращает интерфейс Worker
func NewWorker(id int, client mq.Consumer, producer mq.Producer, messages <-chan mq.Message, logger *logging.Logger,bot *tele.Bot) Worker {
	return &worker{id: id, client: client, producer: producer, messages: messages, logger: logger,bot: bot}
}

//Proccess основной метод структуры worker
func (w *worker) Proccess() {
	// проходимся по всем сообщениям
	for msg := range w.messages {
		// создаём обьект структуры SearchTrack
		event := SearchTrackResponse{}
		// анмаршилим в обьект структуры
		if err := json.Unmarshal(msg.Body, &event); err != nil {
			// логируем ошибки
			w.logger.Errorf("[worker #%d]: failed to unmarshal event due to error %v", w.id, err)
			w.logger.Debugf("[worker #%d]: body: %s", w.id, msg.Body)
			// уведомляем о том что не сообшение не подтверждён консьюмером и позже его переотправляем
			w.reject(msg)
			continue
		}
		i,_ := strconv.ParseInt(event.RequestID, 10 , 64)
		id, err  := w.bot.ChatByID(i)
		if err != nil {
			w.logger.Errorf("[worker #%d]: failed to get chat bu id due to error %v", w.id, err)
		}
		message := "Запрос не обработан, произошла ошибка"
		if event.Success == "true"{
			message = event.Name
		}
		_, err = w.bot.Send(id,message)
		if err != nil {
			w.logger.Errorf("[worker #%d]: failed to Send chat bu id due to error %v", w.id, err)
		}

	}
}

func (w *worker) sendResponse(d map[string]string) {
	// маршалим мапу в джейсона подобную тип данных
	// по факту массив байтов
	b, err := json.Marshal(d)
	if err != nil {
		w.logger.Errorf("[worker #%d]: failed to response due to error %v", w.id, err)
		return
	}
	// отправляем в метод продьюсера, который публикует сообщение и отправляет в очередь
	if err := w.producer.Publish(w.responseQueue, b); err != nil {
		w.logger.Errorf("[worker #%d]: failed to response due to error %v", w.id, err)
	}
}

// функция которая уведомляет о том что сообщение не дошло до консьюмера
func (w *worker) reject(msg mq.Message) {
	if err := w.client.Reject(msg.ID, false); err != nil {
		w.logger.Errorf("[worker #%d]: failed to reject due to error %v", w.id, err)
	}
}

// функция которая уведомляет о том что сообшение дошло до консьюмера
func (w *worker) ack(msg mq.Message) {
	if err := w.client.Ack(msg.ID, false); err != nil {
		w.logger.Errorf("[worker #%d]: failed to ack due to error %v", w.id, err)
	}
}
