package logger

import (
	"io"
	"log"
	"os"
	"time"
)

type LogLvl int

type Logger struct {
	filename string
	logInfo  *log.Logger
	logWarn  *log.Logger
	logError *log.Logger
	logFatal *log.Logger
}

const (
	INFO LogLvl = iota
	WARN
	ERROR
	FATAL
)

func NewLogger() (logger *Logger, file *os.File) {
	t := time.Now()
	file, err := os.OpenFile("log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}
	output := io.MultiWriter(file, os.Stdout)
	logger = &Logger{
		filename: t.Format("2006-Jan-2"),
		logInfo:  log.New(output, "INFO ", log.Ldate|log.Ltime),
		logWarn:  log.New(output, "WARNING ", log.Ldate|log.Ltime),
		logError: log.New(output, "ERROR ", log.Ldate|log.Ltime),
		logFatal: log.New(output, "FATAL ", log.Ldate|log.Ltime),
	}
	return
}

func (l *Logger) WriteLog(level LogLvl, msg string) {
	switch level {
	case INFO:
		l.logInfo.Println(msg)
	case WARN:
		l.logWarn.Println(msg)
	case ERROR:
		l.logError.Println(msg)
	case FATAL:
		l.logFatal.Fatalln(msg)
	}
}
