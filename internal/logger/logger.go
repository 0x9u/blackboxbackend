package logger

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	Warn  *log.Logger
	Info  *log.Logger
	Error *log.Logger
	Fatal *log.Logger
	Debug *log.Logger
)

//replace with Zap library later

func init() {
	//get current date for file
	t := time.Now()
	name := t.Format("2006-01-02") + ".log"
	dirPath := filepath.Join(".", "logs")
	err := os.MkdirAll(dirPath, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	logFilePath := filepath.Join(dirPath, name)
	file, err := os.OpenFile(logFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}
	//print to file and console
	output := io.MultiWriter(os.Stdout, file)
	gin.DefaultWriter = output
	errOutput := io.MultiWriter(os.Stderr, file)

	Info = log.New(output, "INFO:\t", log.Ldate|log.Ltime|log.Lshortfile)
	Warn = log.New(errOutput, "WARN:\t", log.Ldate|log.Ltime|log.Lshortfile)
	Error = log.New(errOutput, "ERROR:\t", log.Ldate|log.Ltime|log.Lshortfile)
	Fatal = log.New(errOutput, "FATAL:\t", log.Ldate|log.Ltime|log.Lshortfile)
	Debug = log.New(output, "DEBUG:\t", log.Ldate|log.Ltime|log.Lshortfile)
}
