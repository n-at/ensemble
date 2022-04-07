package storage

import (
	"golang.org/x/crypto/bcrypt"
	"math/rand"
	"strings"
)

const PasswordEncryptionCost = 15

func GenerateRandomString(length int) string {
	vowels := []rune{'e', 'u', 'i', 'o', 'a'}
	consonants := []rune{'q', 'r', 't', 'p', 's', 'd', 'g', 'h', 'k', 'z', 'x', 'v', 'b', 'n', 'm'}

	str := strings.Builder{}

	for i := 0; i < length; i += 2 {
		str.WriteRune(consonants[rand.Intn(len(consonants))])
		if i != length-1 {
			str.WriteRune(vowels[rand.Intn(len(vowels))])
		}
	}

	return str.String()
}

func EncryptPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), PasswordEncryptionCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
