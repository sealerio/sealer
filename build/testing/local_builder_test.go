package testing

import (
	"gitlab.alibaba-inc.com/seadent/pkg/build"
	"gitlab.alibaba-inc.com/seadent/pkg/common"
	"testing"
)

func TestLocalBuilder_Build(t *testing.T) {
	conf := &build.Config{
	}
	builder := build.NewBuilder(conf, common.LocalBuild)
	err := builder.Build("dashboard-test:latest", ".", "kubefile")
	if err != nil {
		t.Errorf("exec build error %v", err)
	}
}
