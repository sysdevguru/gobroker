package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/scrypt"
)

func SaltEncrypt(value, salt []byte) ([]byte, error) {
	return scrypt.Key(value, salt, 16384, 8, 1, 32)
}

func EncryptWithKey(value, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	b := base64.StdEncoding.EncodeToString(value)
	cipherText := make([]byte, aes.BlockSize+len(b))
	iv := cipherText[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	cfb := cipher.NewCFBEncrypter(block, iv)
	cfb.XORKeyStream(cipherText[aes.BlockSize:], []byte(b))
	return cipherText, nil
}

func DecryptWithkey(value, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(value) < aes.BlockSize {
		return nil, errors.New("Cipher text too short")
	}
	iv := value[:aes.BlockSize]
	value = value[aes.BlockSize:]
	cfb := cipher.NewCFBDecrypter(block, iv)
	cfb.XORKeyStream(value, value)
	data, err := base64.StdEncoding.DecodeString(string(value))
	if err != nil {
		return nil, err
	}
	return data, nil
}

func GenRandomKey(length int) string {
	b := make([]byte, 2*length)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return fmt.Sprintf("%X", b)[0:length]
}

func Mask(value string, start, end int) string {
	out := make([]rune, len(value))
	in := []rune(value)
	for i := 0; i < len(value); i++ {
		if i >= start && i <= end {
			out[i] = rune('*')
		} else {
			out[i] = in[i]
		}
	}
	return string(out)
}
