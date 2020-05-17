package common

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

var (
	DEBUG        = os.Getenv("DEBUG")
	SQL_DEBUG    = envToBool("SQL_DEBUG", false)
	DATABASE_URL = getEnv("DATABASE_URL", fmt.Sprintf("sqlite:%v", filepath.Join(AppDir, "main.db")))
	WsAddr       = "0.0.0.0:9998"
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
