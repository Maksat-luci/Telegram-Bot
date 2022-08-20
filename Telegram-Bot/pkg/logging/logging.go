package logging

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"runtime"

	"github.com/sirupsen/logrus"
)

type writerHook struct {
	Writer    []io.Writer
	LogLevels []logrus.Level
}

func (hook *writerHook) Fire(entry *logrus.Entry) error {
	line, err := entry.String()
	if err != nil {
		return err
	}
	for _, w := range hook.Writer {
		_, err = w.Write([]byte(line))

	}
	return err
}

func (hook *writerHook) Levels() []logrus.Level {
	return hook.LogLevels
}

var e *logrus.Entry

// Logger структура  
type Logger struct {
	*logrus.Entry
}
// GetLogger конструктор структуры Logger
func GetLogger() *Logger {
	return &Logger{e}
}

// GetLoggerWithField функция для 
func (l *Logger) GetLoggerWithField(k string, v interface{}) *Logger {
	return &Logger{l.WithField(k, v)}
}

// Init функция иницилизации логруса
func Init(level string) {
	logrusLevel, err := logrus.ParseLevel(level)
	if err != nil {
		log.Fatalln(err)
	}
	l := logrus.New()
	l.SetReportCaller(true)
	l.Formatter = &logrus.TextFormatter{
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			filename := path.Base(f.File)
			return fmt.Sprintf("%s:%d", filename, f.Line), fmt.Sprintf("%s()", f.Function)
		},
		DisableColors: false,
		FullTimestamp: true,
	}

	l.SetOutput(ioutil.Discard) // Send all logs to nowhere by deafult

	l.AddHook(&writerHook{
		Writer: []io.Writer{os.Stdout}, 
		LogLevels: logrus.AllLevels,
	})

	l.SetLevel(logrusLevel)
	
	e = logrus.NewEntry(l)

}
