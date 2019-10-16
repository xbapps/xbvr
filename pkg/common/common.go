package common

import (
	"os"
)

var (
	DEBUG    = os.Getenv("DEBUG")
	HttpAddr = "0.0.0.0:9999"
	WsAddr   = "0.0.0.0:9998"
)
