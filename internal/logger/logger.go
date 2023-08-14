package logger

import (
	"fmt"
	"os"
	"path"
	"runtime"

	"github.com/sirupsen/logrus"
)

func InitLog() *logrus.Logger {
	log := logrus.New()
	log.SetReportCaller(true)
	//file, err := os.OpenFile("logrus.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0640)
	/* 	if err == nil {
	   		log.Out = file
	   	} else {
	   		log.Info("Failed to log to file, using default stdout") */
	log.Out = os.Stdout
	//}
	log.Formatter = &logrus.TextFormatter{
		CallerPrettyfier: func(frame *runtime.Frame) (function string, file string) {
			filename := path.Base(frame.File)
			return fmt.Sprintf("%s():%d", frame.Function, frame.Line), filename
		},
		DisableColors:  true,
		DisableSorting: false,
	}

	return log
}
