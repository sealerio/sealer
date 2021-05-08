package hash

import (
	"io"
	"os"

	"github.com/opencontainers/go-digest"
)

type Interface interface {
	CheckSum(reader io.Reader) (*digest.Digest, error)
	TarCheckSum(src string) (*os.File, string, error)
}
