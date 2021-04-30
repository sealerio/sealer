package build

import (
	"fmt"

	"github.com/alibaba/sealer/apply"
	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/utils"
)

func (l *LocalBuilder) applyCluster() error {
	if !utils.IsFileExist(common.TmpClusterfile) {
		return fmt.Errorf("%s not found", common.TmpClusterfile)
	}
	applier := apply.NewApplierFromFile(common.TmpClusterfile)
	if err := applier.Apply(); err != nil {
		return fmt.Errorf("failed to apply cluster:%v", err)
	}
	logger.Info("apply cluster success !")
	return nil
}
