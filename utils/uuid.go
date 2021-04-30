package utils

import (
	"crypto/rand"
	"fmt"
)

// GenUniqueId: gen uuid
func GenUniqueID(n int) string {
	randBytes := make([]byte, n/2)
	rand.Read(randBytes)
	return fmt.Sprintf("%x", randBytes)
}
