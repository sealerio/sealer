package utils

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

//DirMD5 count files md5
/*func DirMD5(dirName string) string {
	var md5Value []byte
	filepath.Walk(dirName, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("access path error %v", err)
		}

		if !info.IsDir() {
			data, err := ioutil.ReadFile(path)
			if err != nil {
				return fmt.Errorf("walk file error %v", err)
			}
			bytes := md5.Sum(data)
			md5Value = append(md5Value, bytes[:]...)
		}
		return nil
	})
	md5Values := md5.Sum(md5Value)
	return hex.EncodeToString(md5Values[:])
}*/

func MD5(body ...[]byte) string {
	md5Hash := md5.New()
	for _, b := range body {
		md5Hash.Write(b)
	}
	return hex.EncodeToString(md5Hash.Sum(nil))
}

//FileMD5 count file md5
func FileMD5(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}

	m := md5.New()
	if _, err := io.Copy(m, file); err != nil {
		return "", err
	}

	fileMd5 := fmt.Sprintf("%x", m.Sum(nil))
	return fileMd5, nil
}
