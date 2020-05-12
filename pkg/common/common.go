package common

import (
	"os"
	"strconv"
)

var (
	DEBUG          = os.Getenv("DEBUG")
	MYSQL_HOST     = os.Getenv("MYSQL_HOST")
	MYSQL_DB       = os.Getenv("MYSQL_DB")
	MYSQL_USER     = os.Getenv("MYSQL_USER")
	MYSQL_PASSWORD = os.Getenv("MYSQL_PASSWORD")
	SQLITE_PARAMS  = os.Getenv("SQLITE_PARAMS")
	WsAddr         = "0.0.0.0:9998"
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
