package storage

import (
	"crypto/aes"
	"crypto/cipher"
	cryptoRand "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"io"
	"math/rand"
	"strings"
)

const PasswordEncryptionCost = 15

///////////////////////////////////////////////////////////////////////////////

func NewId() string {
	return uuid.New().String()
}

///////////////////////////////////////////////////////////////////////////////

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

///////////////////////////////////////////////////////////////////////////////

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

///////////////////////////////////////////////////////////////////////////////

func EncryptString(key, text string) (string, error) {
	if len(key) == 0 {
		return text, nil
	}

	plaintext := []byte(text)

	block, err := aes.NewCipher(Sha256(key))
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(cryptoRand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)

	return fmt.Sprintf("%x", ciphertext), nil
}

func DecryptString(key, text string) (string, error) {
	if len(key) == 0 {
		return text, nil
	}

	enc, _ := hex.DecodeString(text)

	block, err := aes.NewCipher(Sha256(key))
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := aesGCM.NonceSize()
	if len(enc) < nonceSize {
		return "", errors.New("encrypted text too short")
	}

	nonce, ciphertext := enc[:nonceSize], enc[nonceSize:]

	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s", plaintext), nil
}

func Sha256(text string) []byte {
	hash := sha256.Sum256([]byte(text))
	return hash[:]
}
