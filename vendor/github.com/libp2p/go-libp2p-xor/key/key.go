package key

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	kbucket "github.com/libp2p/go-libp2p-kbucket"
)

func KbucketIDToKey(id kbucket.ID) Key {
	return Key(id)
}

func ByteKey(b byte) Key {
	return Key{b}
}

func BytesKey(b []byte) Key {
	return Key(b)
}

// Key is a vector of bits backed by a Go byte slice.
// First byte is most significant.
// First bit (in each byte) is least significant.
type Key []byte

func (k Key) BitAt(offset int) byte {
	if k[offset/8]&(byte(1)<<(7-offset%8)) == 0 {
		return 0
	} else {
		return 1
	}
}

func (k Key) NormInt() *big.Int {
	return big.NewInt(0).SetBytes(k)
}

func (k Key) BitLen() int {
	return 8 * len(k)
}

func (k Key) String() string {
	b, _ := json.Marshal(k)
	return string(b)
}

// BitString returns a bit representation of the key, in descending order of significance.
func (k Key) BitString() string {
	s := make([]string, len(k))
	for i, b := range k {
		s[i] = fmt.Sprintf("%08b", b)
	}
	return strings.Join(s, "")
}

func Equal(x, y Key) bool {
	return bytes.Equal(x, y)
}

func Xor(x, y Key) Key {
	z := make(Key, len(x))
	for i := range x {
		z[i] = x[i] ^ y[i]
	}
	return z
}

func DistInt(x, y Key) *big.Int {
	return Xor(x, y).NormInt()
}
