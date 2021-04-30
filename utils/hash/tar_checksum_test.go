package hash

import (
	"testing"

	"github.com/alibaba/sealer/logger"
)

func TestCheckSumAndPlaceLayer(t *testing.T) {
	dst := "/Users/jim/Workspace/kubeb"
	dig, err := CheckSumAndPlaceLayer(dst)
	if err != nil {
		t.Error(err)
	}
	logger.Info("the digest hex is:" + dig)
}
