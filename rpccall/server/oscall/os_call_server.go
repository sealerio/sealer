package oscall

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/alibaba/sealer/logger"

	pb "github.com/alibaba/sealer/rpccall/baseapi/oscall"
	"google.golang.org/protobuf/types/known/emptypb"
)

type server struct {
	pb.UnimplementedOSCallServer
}

func (*server) Mkdir(ctx context.Context, dir *pb.Dir) (*emptypb.Empty, error) {
	var resp emptypb.Empty
	fm := dir.FileMode
	if fm == 0 {
		fm = 0755
	}

	err := os.MkdirAll(dir.Name, os.FileMode(fm))
	if err != nil {
		return nil, fmt.Errorf("failed to mkdir %s, err: %s", dir.Name, err)
	}
	return &resp, nil
}

func (*server) CPFiles(ctx context.Context, filesToCopy *pb.FilesToCopy) (*emptypb.Empty, error) {
	var (
		files  = filesToCopy.Files
		tarDir = filesToCopy.Dir
		resp   emptypb.Empty
	)

	for _, file := range files {
		fileBaseName := filepath.Base(file.Name)
		err := copyFile(file.Name, filepath.Join(tarDir, fileBaseName))
		if err != nil {
			return nil, fmt.Errorf("failed to copy file %s to %s, err: %s", file.Name, filepath.Join(tarDir, fileBaseName), err)
		}
	}
	return &resp, nil
}

func (*server) WriteFile(ctx context.Context, fileWithContent *pb.FileWithContent) (*emptypb.Empty, error) {
	var (
		file    = fileWithContent.File.Name
		content = fileWithContent.Content
		resp    emptypb.Empty
	)

	theFile, err := os.Create(file)
	if err != nil {
		return nil, err
	}
	defer theFile.Close()

	_, err = io.Copy(theFile, bytes.NewBuffer(content))
	if err != nil {
		return nil, err
	}

	return &resp, theFile.Sync()
}

func (*server) RMFiles(ctx context.Context, toDelete *pb.FilesToDelete) (*emptypb.Empty, error) {
	var (
		files = toDelete.Files
		resp  emptypb.Empty
		err   error
	)

	for _, f := range files {
		_, err = os.Stat(f.Name)
		if err != nil {
			logger.Warn(err)
			continue
		}

		err = os.Remove(f.Name)
		if err != nil {
			logger.Warn(err)
		}
	}

	return &resp, nil
}

func copyFile(src, tar string) error {
	srcFI, err := os.Stat(src)
	if err != nil {
		return err
	}

	if srcFI.IsDir() {
		return fmt.Errorf("source file %s is not a regular file, cannot copy it", srcFI.Name())
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open src file %s, err: %s", src, err)
	}
	defer srcFile.Close()

	err = os.MkdirAll(filepath.Dir(tar), 0755)
	if err != nil {
		return fmt.Errorf("failed to mkdir %s, err: %s", filepath.Dir(tar), err)
	}

	tarFile, err := os.Create(tar)
	if err != nil {
		return fmt.Errorf("failed to create target file %s, err: %s", tar, err)
	}
	defer tarFile.Close()

	_, err = io.Copy(tarFile, srcFile)
	if err != nil {
		return err
	}

	return tarFile.Sync()
}

func NewServer() pb.OSCallServer {
	return &server{}
}
