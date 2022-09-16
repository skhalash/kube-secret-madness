package rand

import (
	"encoding/base64"
	"math/rand"
	"time"
)

const charset = "abcdefghijklmnopqrstuvwxyz" +
	"0123456789"

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func StringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[Index(len(charset))]
	}
	return string(b)
}

func String(length int) string {
	return StringWithCharset(length, charset)
}

func SecretData() map[string][]byte {
	data := make(map[string][]byte)
	key := String(5)
	value := String(10)
	valueBase64 := base64.StdEncoding.EncodeToString([]byte(value))

	data[key] = []byte(valueBase64)

	return data
}

func Index(n int) int {
	return seededRand.Intn(n)
}
