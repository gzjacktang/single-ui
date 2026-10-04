package util

import (
	"crypto/rand"
	"encoding/base64"
)

const Shadowsocks2022Method = "2022-blake3-aes-256-gcm"

func GenerateShadowsocks2022Key() (string, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

func ValidShadowsocks2022Key(value string) bool {
	key, err := base64.StdEncoding.DecodeString(value)
	return err == nil && len(key) == 32
}
