package common

import (
	"os"
	"strconv"
)

var (
	DEBUG         = os.Getenv("DEBUG")
	SQL_DEBUG     = envToBool("SQL_DEBUG", false)
	DB_TYPE       = getEnv("DB_TYPE", "sqlite3")
	DB_HOST       = os.Getenv("DB_HOST")
	DB_NAME       = os.Getenv("DB_NAME")
	DB_USER       = os.Getenv("DB_USER")
	DB_PASSWORD   = os.Getenv("DB_PASSWORD")
	SQLITE_PARAMS = os.Getenv("SQLITE_PARAMS")
	WsAddr        = "0.0.0.0:9998"
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
