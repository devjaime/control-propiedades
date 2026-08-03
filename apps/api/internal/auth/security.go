package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

const passwordCost = 12

func HashPassword(password string) (string, error) {
	if err := ValidatePassword(password); err != nil {
		return "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), passwordCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func ComparePassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func HashAccessCode(code string) (string, error) {
	if len(code) != 4 {
		return "", errors.New("el código de acceso debe contener cuatro dígitos")
	}
	for _, character := range code {
		if character < '0' || character > '9' {
			return "", errors.New("el código de acceso sólo puede contener dígitos")
		}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(code), passwordCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func ValidatePassword(password string) error {
	length := utf8.RuneCountInString(password)
	if length < 10 {
		return errors.New("la contraseña debe tener al menos 10 caracteres")
	}
	if len([]byte(password)) > 72 {
		return errors.New("la contraseña no puede superar 72 bytes")
	}
	if strings.TrimSpace(password) != password {
		return errors.New("la contraseña no puede comenzar o terminar con espacios")
	}
	return nil
}

func NewSessionToken() (plain string, hash []byte, err error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", nil, err
	}
	plain = base64.RawURLEncoding.EncodeToString(buffer)
	digest := sha256.Sum256([]byte(plain))
	return plain, digest[:], nil
}

func HashSessionToken(token string) []byte {
	digest := sha256.Sum256([]byte(token))
	return digest[:]
}
