package testing

import (
	"testing"

	"github.com/alibaba/sealer/build"
	"github.com/alibaba/sealer/common"
)

func TestLocalBuilder_Build(t *testing.T) {
	conf := &build.Config{}
	builder := build.NewBuilder(conf, common.LocalBuild)
	err := builder.Build("dashboard-test:latest", ".", "kubefile")
	if err != nil {
		t.Errorf("exec build error %v", err)
	}
}
