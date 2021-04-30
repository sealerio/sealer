package hash

import (
	"github.com/opencontainers/go-digest"
	"io"
	"os"
)

type Interface interface {
	CheckSum(reader io.Reader) (*digest.Digest, error)
	TarCheckSum(src string) (*os.File, string, error)
}
