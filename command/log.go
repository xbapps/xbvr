package command

import (
	"os"
	"runtime"

	"github.com/sirupsen/logrus"
	"github.com/x-cray/logrus-prefixed-formatter"
)

var log = logrus.New()

func init() {
	log.Out = os.Stdout
	log.SetLevel(logrus.InfoLevel)

	if runtime.GOOS == "windows" {
		log.Formatter = &prefixed.TextFormatter{
			DisableColors: true,
		}
	} else {
		log.Formatter = &prefixed.TextFormatter{}
	}
}
