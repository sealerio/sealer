package image

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/alibaba/sealer/image/reference"
	"github.com/alibaba/sealer/image/store"
	"github.com/opencontainers/go-digest"

	"github.com/alibaba/sealer/common"
	utils2 "github.com/alibaba/sealer/image/utils"
	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
)

type DefaultImageFileService struct {
}

func (d DefaultImageFileService) Load(imageSrc string) error {
	return Decompress(imageSrc)
}

func (d DefaultImageFileService) Save(imageName string, imageTar string) error {
	return Compress(imageName, imageTar)
}

func (d DefaultImageFileService) Merge(image *v1.Image) error {
	panic("implement me")
}

func Compress(imageName string, imageTar string) error {
	named, err := reference.ParseToNamed(imageName)
	if err != nil {
		return err
	}

	ima, err := utils2.GetImage(named.Raw())
	if err != nil {
		return err
	}
	if imageTar == "" {
		imageTar = ima.Spec.ID + common.TarGzipSuffix
	}
	if _, err := os.Stat(imageTar); err == nil {
		return fmt.Errorf("file %s exist", imageTar)
	}
	var layerDBCompress []string
	var layerCompress []string
	for _, layer := range ima.Spec.Layers {
		if layer.Hash == "" {
			continue
		}
		layerID := store.LayerID(digest.NewDigestFromEncoded(digest.SHA256, layer.Hash))
		layerDir := filepath.Join(common.DefaultLayerDir, digest.Digest(layerID).Hex())
		layerCompress = append(layerCompress, layerDir)
		digs := digest.Digest(layerID)
		subDir := filepath.Join(common.DefaultLayerDBDir, digs.Algorithm().String(), digs.Hex())
		layerDBCompress = append(layerDBCompress, subDir)
	}
	compressFiles := append(append(layerDBCompress, layerCompress...), filepath.Join(common.DefaultImageMetaRootDir, ima.Spec.ID+common.YamlSuffix))

	if filepath.IsAbs(imageTar) {
		err = os.MkdirAll(filepath.Dir(imageTar), common.FileMode0644)
		if err != nil {
			return err
		}
	}
	file, err := os.Create(imageTar)
	defer func() {
		if err != nil {
			utils.CleanFile(file)
		}
		_ = file.Close()
	}()

	zr := gzip.NewWriter(file)
	tw := tar.NewWriter(zr)
	defer func() {
		_ = tw.Close()
		_ = zr.Close()
	}()
	for _, src := range compressFiles {
		if len(src) == 0 {
			return errors.New("[compress] source must be provided")
		}

		if !filepath.IsAbs(src) {
			return errors.New("src should be absolute path")
		}
		var f os.FileInfo
		var newFolder string
		f, err = os.Stat(src)
		if err != nil {
			return err
		}
		if f.IsDir() {
			newFolder = strings.TrimPrefix(src, common.DefaultImageDir+"/")
		} else {
			newFolder = strings.TrimPrefix(filepath.Dir(src), common.DefaultImageDir+"/")
		}

		//use existing file
		src = strings.TrimSuffix(src, "/")
		srcPrefix := filepath.ToSlash(src + "/")
		err = filepath.Walk(src, func(file string, fi os.FileInfo, err error) error {
			// generate tar header
			header, walkErr := tar.FileInfoHeader(fi, file)
			if walkErr != nil {
				return err
			}
			if file != src {
				absPath := filepath.ToSlash(file)
				header.Name = filepath.Join(newFolder, strings.TrimPrefix(absPath, srcPrefix))
			} else {
				// do not contain root dir
				if fi.IsDir() {
					return nil
				}
				// for supporting tar single file
				header.Name = filepath.Join(newFolder, filepath.Base(src))
			}
			fmt.Println(header.Name)
			// write header
			if err = tw.WriteHeader(header); err != nil {
				return err
			}
			// if not a dir, write file content
			if !fi.IsDir() {
				data, err := os.Open(file)
				if err != nil {
					return err
				}
				if _, err = io.Copy(tw, data); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	logger.Info("save image %s success", ima.Name)
	return nil
}

func Decompress(imageSrc string) error {
	src, err := os.Open(imageSrc)
	if err != nil {
		return fmt.Errorf("failed to open %s,%v", imageSrc, err)
	}
	defer src.Close()
	var dst = common.DefaultImageDir
	err = os.MkdirAll(dst, common.FileMode0755)
	if err != nil {
		return err
	}

	zr, err := gzip.NewReader(src)
	if err != nil {
		return err
	}

	tr := tar.NewReader(zr)
	type DirStruct struct {
		header     *tar.Header
		dir        string
		next, prev *DirStruct
	}

	prefixes := make(map[string]*DirStruct)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// validate name against path traversal
		if !validRelPath(header.Name) {
			return fmt.Errorf("tar contained invalid name error %q", header.Name)
		}

		target := filepath.Join(dst, header.Name)
		//var readImageMetadata bool
		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err = os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
					return err
				}

				// building a double linked list
				prefix := filepath.Dir(target)
				prev := prefixes[prefix]
				//an root dir
				if prev == nil {
					prefixes[target] = &DirStruct{header: header, dir: target, next: nil, prev: nil}
				} else {
					newHead := &DirStruct{header: header, dir: target, next: nil, prev: prev}
					prev.next = newHead
					prefixes[target] = newHead
				}
			}

		case tar.TypeReg:
			err = func() error {
				fmt.Println(header.Name)
				err := utils.MkDirIfNotExists(filepath.Dir(target))
				if err != nil {
					return err
				}
				fileToWrite, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_RDWR, os.FileMode(header.Mode))
				if err != nil {
					return err
				}

				defer fileToWrite.Close()
				if _, err := io.Copy(fileToWrite, tr); err != nil {
					return err
				}
				if filepath.Dir(header.Name) == filepath.Base(common.DefaultImageMetaRootDir) {
					image, err := utils2.GetImageByID(strings.TrimSuffix(filepath.Base(target), common.YamlSuffix))
					if err != nil {
						return fmt.Errorf("failed to read image.yaml,%v", err)
					}
					if err := utils2.SetImageMetadata(utils2.ImageMetadata{Name: image.Name, ID: image.Spec.ID}); err != nil {
						return fmt.Errorf("failed to set image metadata, %v", err)
					}
				}
				// for not changing
				if err = os.Chtimes(target, header.AccessTime, header.ModTime); err != nil {
					return err
				}
				return nil
			}()

			if err != nil {
				return err
			}
		}
	}

	for _, v := range prefixes {
		// for taking the last one
		if v.next != nil {
			continue
		}

		// every change in dir, will change the metadata of that dir
		// change times from the last one
		// do this is for not changing metadata of parent dir
		for dirStr := v; dirStr != nil; dirStr = dirStr.prev {
			if err = os.Chtimes(dirStr.dir, dirStr.header.AccessTime, dirStr.header.ModTime); err != nil {
				return err
			}
		}
	}

	return nil
}

// check for path traversal and correct forward slashes
func validRelPath(p string) bool {
	if p == "" || strings.Contains(p, `\`) || strings.HasPrefix(p, "/") || strings.Contains(p, "../") {
		return false
	}
	return true
}
