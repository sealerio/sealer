package collector

import "fmt"

type Collector interface {
	// Send git package;download common file work as wget or curl;copy local file to dst.
	Send(buildContext, src, savePath string) error
}

func NewCollector(src string) (Collector, error) {
	// if src is detected as remote context,will new different Collector via src type.
	switch {
	case src == "":
		return nil, fmt.Errorf("src can not be nil")
	case IsGitURL(src):
		// remote git context
		return NewGitCollector(), nil
	case IsURL(src):
		// remote web context
		return NewFileCollector(), nil
	default:
		//local context
		return NewLocalCollector(), nil
	}
}
