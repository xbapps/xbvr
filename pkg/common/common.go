package common

import (
	"os"
	"strconv"
)

var (
	DEBUG    = os.Getenv("DEBUG")
	DLNA     = envToBool("ENABLE_DLNA", true)
	HttpAddr = "0.0.0.0:9999"
	WsAddr   = "0.0.0.0:9998"
)

func envToBool(envVar string, defaultVal bool) bool {
	v, s := os.LookupEnv(envVar)
	if s {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		} else {
			return defaultVal
		}
	} else {
		return defaultVal
	}
}
