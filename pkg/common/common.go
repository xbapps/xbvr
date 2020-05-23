package common

import (
	"os"
	"strconv"
)

var (
	DEBUG             = os.Getenv("DEBUG")
	DISABLE_ANALYTICS = os.Getenv("DISABLE_ANALYTICS")
	SQLITE_PARAMS     = os.Getenv("SQLITE_PARAMS")
	WsAddr            = "0.0.0.0:9998"
	CurrentVersion    = ""
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
