package hash

import (
	"testing"

	"gitlab.alibaba-inc.com/seadent/pkg/logger"
)

func TestCheckSumAndPlaceLayer(t *testing.T) {
	dst := "/Users/jim/Workspace/kubeb"
	dig, err := CheckSumAndPlaceLayer(dst)
	if err != nil {
		t.Error(err)
	}
	logger.Info("the digest hex is:" + dig)
}
