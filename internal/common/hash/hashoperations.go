// Package hash Общие процедуры взаимодейсивя с хэшем
package hash

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

type HashFunc func(secretKey string, data []byte) string

func СomputeHexadecimalSha256Hash(secretKey string, data []byte) string {

	hash := СomputeSha256Hash(secretKey, data)
	return hex.EncodeToString(hash)
}

func СomputeSha256Hash(secretKey string, data []byte) []byte {
	h := hmac.New(sha256.New, []byte(secretKey))
	return h.Sum(data)
}
