package api

import (
	"context"
	"os"
)

type OSCall interface {
	Mkdir(ctx context.Context, dir string, mode os.FileMode) error

	CPFiles(ctx context.Context, dir string, files ...string) error

	WriteFile(ctx context.Context, file string, content []byte) error

	RMFiles(ctx context.Context, files ...string) error
}
