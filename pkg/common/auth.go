package common

import (
	"golang.org/x/crypto/bcrypt"
)

func IsUIAuthEnabled() bool {
	if EnvConfig.UIUsername != "" && EnvConfig.UIPassword != "" {
		return true
	} else {
		return false
	}
}

func GetUISecret(user string, realm string) string {
	if user == EnvConfig.UIUsername {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(EnvConfig.UIPassword), bcrypt.DefaultCost)
		if err == nil {
			return string(hashedPassword)
		}
	}
	return ""
}
