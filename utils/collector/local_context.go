package collector

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/alibaba/sealer/logger"
	fsutil "github.com/tonistiigi/fsutil/copy"
)

type localCollector struct {
}

func (l localCollector) Send(buildContext, src, savePath string) error {
	xattrErrorHandler := func(dst, src, key string, err error) error {
		logger.Warn(err)
		return nil
	}
	opt := []fsutil.Opt{
		fsutil.WithXAttrErrorHandler(xattrErrorHandler),
	}

	m, err := fsutil.ResolveWildcards(buildContext, src, true)
	if err != nil {
		return err
	}

	if len(m) == 0 {
		return fmt.Errorf("%s not found", src)
	}
	for _, s := range m {
		if err := fsutil.Copy(context.TODO(), buildContext, s, savePath, filepath.Base(s), opt...); err != nil {
			return err
		}
	}
	return nil
}

func NewLocalCollector() Collector {
	return localCollector{}
}
