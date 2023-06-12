package config

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
	log.Out = os.Stdout
	log.Formatter = &logrus.TextFormatter{
		CallerPrettyfier: func(frame *runtime.Frame) (function string, file string) {
			filename := path.Base(frame.File)
			return fmt.Sprintf("%s():%d", frame.Function, frame.Line), fmt.Sprintf("%s", filename)
		},
		DisableColors:  true,
		DisableSorting: false,
	}

	return log
}
