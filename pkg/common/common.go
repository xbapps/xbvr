package common

import (
	"os"
	"strconv"
)

var (
	DEBUG             = os.Getenv("DEBUG")
	UIPASSWORD        = os.Getenv("UI_PASSWORD")
	UIUSER            = os.Getenv("UI_USERNAME")
	DISABLE_ANALYTICS = os.Getenv("DISABLE_ANALYTICS")
	SQL_DEBUG         = envToBool("SQL_DEBUG", false)
	DATABASE_URL      = ""
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

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
