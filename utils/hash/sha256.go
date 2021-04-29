package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/opencontainers/go-digest"
	"gitlab.alibaba-inc.com/seadent/pkg/common"
	"gitlab.alibaba-inc.com/seadent/pkg/utils"
	"gitlab.alibaba-inc.com/seadent/pkg/utils/compress"
	"io"
	"os"
	"path/filepath"
)

type SHA256 struct {
}

func (sha SHA256) CheckSum(reader io.Reader) (*digest.Digest, error) {
	hash := sha256.New()
	if _, err := io.Copy(hash, reader); err != nil {
		return nil, err
	}
	dig := digest.NewDigestFromEncoded(digest.SHA256, hex.EncodeToString(hash.Sum(nil)))
	return &dig, nil
}

func (sha SHA256) TarCheckSum(src string) (*os.File, string, error) {
	file, err := compress.Compress(src, "", nil)
	if err != nil {
		return nil, "", err
	}

	_, err = file.Seek(0, 0)
	if err != nil {
		return nil, "", err
	}

	dig, err := sha.CheckSum(file)
	if err != nil {
		return nil, "", err
	}

	_, err = file.Seek(0, 0)
	if err != nil {
		return nil, "", err
	}
	return file, dig.Hex(), nil
}

func CheckSumAndPlaceLayer(dir string) (string, error) {
	sha := SHA256{}
	file, dig, err := sha.TarCheckSum(dir)
	if err != nil {
		return "", err
	}

	defer utils.CleanFile(file)
	err = compress.Uncompress(file, filepath.Join(common.DefaultLayerDir, dig))
	if err != nil {
		return "", err
	}

	return dig, nil
}
