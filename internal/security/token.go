package security

import (
	"crypto/rand"
	"encoding/hex"
)

func RandomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func NewAccessToken() (string, error)  { return RandomHex(24) }
func NewSessionToken() (string, error) { return RandomHex(32) }
func NewCSRFToken() (string, error)    { return RandomHex(32) }

func NewUUIDLike() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80
	out := hex.EncodeToString(buf)
	return out[0:8] + "-" + out[8:12] + "-" + out[12:16] + "-" + out[16:20] + "-" + out[20:32], nil
}
