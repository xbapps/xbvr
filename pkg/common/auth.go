package common

import (
	"golang.org/x/crypto/bcrypt"
)

func IsUIAuthEnabled() bool {
	if UIPASSWORD != "" && UIUSER != "" {
		return true
	} else {
		return false
	}
}

func GetUISecret(user string, realm string) string {
	if user == UIUSER {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(UIPASSWORD), bcrypt.DefaultCost)
		if err == nil {
			return string(hashedPassword)
		}
	}
	return ""
}
