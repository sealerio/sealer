package apply

import (
	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
	"strconv"
	"strings"
)

func StrToInt(str string) int {
	num, err := strconv.Atoi(str)
	if err != nil {
		logger.Error("String to digit conversion failed:", err)
		return 0
	}
	return num
}

func JoinApplierFromArgs(clusterfile string, joinArgs *common.RunArgs) Interface {
	cluster := &v1.Cluster{}
	if err := utils.UnmarshalYamlFile(clusterfile, cluster); err != nil {
		logger.Error("clusterfile parsing failed, please check:", err)
		return nil
	}
	if joinArgs.Nodes == "" && joinArgs.Masters == "" {
		logger.Error("The node or master parameter was not committed")
		return nil
	}
	if cluster.Spec.Provider == "" {
		if IsIPList(joinArgs.Nodes) || IsIPList(joinArgs.Masters) {
			cluster.Spec.Masters.IPList = append(cluster.Spec.Masters.IPList, strings.Split(joinArgs.Masters, ",")...)
			cluster.Spec.Nodes.IPList = append(cluster.Spec.Masters.IPList, strings.Split(joinArgs.Nodes, ",")...)
		} else {
			logger.Error("Parameter error:", "provider cannot be empty when using cloud service！")
			return nil
		}
	} else {
		if IsNumber(joinArgs.Nodes) || IsNumber(joinArgs.Masters) {
			cluster.Spec.Masters.Count = strconv.Itoa(StrToInt(cluster.Spec.Masters.Count) + StrToInt(joinArgs.Masters))
			cluster.Spec.Nodes.Count = strconv.Itoa(StrToInt(cluster.Spec.Nodes.Count) + StrToInt(joinArgs.Nodes))
		} else {
			logger.Error("Parameter error:", "The current mode should submit iplist！")
			return nil
		}
	}
	if err := utils.MarshalYamlToFile(clusterfile, cluster); err != nil {
		logger.Error("clusterfile save failed, please check:", err)
		return nil
	}
	return NewApplier(cluster)
}
