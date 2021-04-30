package build

import (
	"fmt"

	"gitlab.alibaba-inc.com/seadent/pkg/apply"
	"gitlab.alibaba-inc.com/seadent/pkg/common"
	"gitlab.alibaba-inc.com/seadent/pkg/logger"
	"gitlab.alibaba-inc.com/seadent/pkg/utils"
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
