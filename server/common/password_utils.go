package common

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

func CheckPassword(hashedPassword, cleartextPassword string) error {
	if hashedPassword == "" {
		return errors.New("empty hashed password")
	}
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(cleartextPassword))
}

func HashPassword(cleartextPassword string) (string, error) {
	data, err := bcrypt.GenerateFromPassword([]byte(cleartextPassword), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
