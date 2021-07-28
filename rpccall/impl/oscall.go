package impl

import (
	"context"
	"os"

	"github.com/alibaba/sealer/rpccall/api"
	"github.com/alibaba/sealer/rpccall/baseapi/oscall"
)

type remoteOSCall struct {
	client oscall.OSCallClient
}

func NewRemoteOSCall(client oscall.OSCallClient) api.OSCall {
	return &remoteOSCall{client: client}
}

func (r *remoteOSCall) Mkdir(ctx context.Context, dir string, mode os.FileMode) error {
	_, err := r.client.Mkdir(ctx, &oscall.Dir{
		Name:     dir,
		FileMode: uint32(mode),
	})

	return err
}

func (r *remoteOSCall) CPFiles(ctx context.Context, dir string, files ...string) error {
	fs := []*oscall.File{}
	for _, f := range files {
		fs = append(fs, &oscall.File{Name: f})
	}

	_, err := r.client.CPFiles(ctx, &oscall.FilesToCopy{Files: fs, Dir: dir})
	return err
}

func (r *remoteOSCall) WriteFile(ctx context.Context, file string, content []byte) error {
	_, err := r.client.WriteFile(ctx,
		&oscall.FileWithContent{
			File:    &oscall.File{Name: file},
			Content: content,
		})
	return err
}

func (r *remoteOSCall) RMFiles(ctx context.Context, files ...string) error {
	pbFiles := []*oscall.File{}
	for _, f := range files {
		pbFiles = append(pbFiles, &oscall.File{Name: f})
	}

	_, err := r.client.RMFiles(ctx, &oscall.FilesToDelete{Files: pbFiles})
	return err
}
