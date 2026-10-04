package util

import (
	"crypto/rand"
	"encoding/base64"
)

const Shadowsocks2022Method = "2022-blake3-aes-256-gcm"

func ShadowsocksMethod(method string) string {
	if method == "" {
		return Shadowsocks2022Method
	}
	return method
}

func SupportedShadowsocksMethod(method string) bool {
	switch method {
	case Shadowsocks2022Method, "aes-256-gcm", "chacha20-ietf-poly1305", "xchacha20-ietf-poly1305":
		return true
	}
	return false
}

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
