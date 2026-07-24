package emailing

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
)

type SecretBox struct{ key [32]byte }

func NewSecretBox(secret string) (*SecretBox, error) {
	if secret == "" {
		return nil, errors.New("EMAIL_ENCRYPTION_KEY is required")
	}
	return &SecretBox{key: sha256.Sum256([]byte(secret))}, nil
}
func (b *SecretBox) Encrypt(value string) ([]byte, error) {
	block, err := aes.NewCipher(b.key[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, []byte(value), nil), nil
}
func (b *SecretBox) Decrypt(value []byte) (string, error) {
	block, err := aes.NewCipher(b.key[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(value) < gcm.NonceSize() {
		return "", errors.New("invalid encrypted secret")
	}
	plain, err := gcm.Open(nil, value[:gcm.NonceSize()], value[gcm.NonceSize():], nil)
	return string(plain), err
}
func NewToken() (plain string, hash []byte, err error) {
	raw := make([]byte, 32)
	if _, err = io.ReadFull(rand.Reader, raw); err != nil {
		return "", nil, err
	}
	plain = base64.RawURLEncoding.EncodeToString(raw)
	sum := sha256.Sum256([]byte(plain))
	return plain, sum[:], nil
}
func TokenHash(plain string) []byte { sum := sha256.Sum256([]byte(plain)); return sum[:] }
