package infra

import (
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"gitlab.alibaba-inc.com/seadent/pkg/logger"
	"os"
	"time"
)

func Retry(tryTimes int, trySleepTime time.Duration, action func() error) error {
	var err error
	for i := 0; i < tryTimes; i++ {
		err = action()
		if err == nil {
			return nil
		}

		time.Sleep(trySleepTime * time.Duration(2*i+1))
	}
	return fmt.Errorf("retry action timeout: %v", err)
}

func (a *AliProvider) ReconcileResource(resourceKey string, action Alifunc) error {
	if a.Cluster.Annotations[resourceKey] == "" {
		err := action()
		if err != nil {
			return err
		}
		logger.Info("create resource success %s: %s", resourceKey, a.Cluster.Annotations[resourceKey])
	}
	return nil
}

func (a *AliProvider) DeleteResource(resourceKey string, action Alifunc) {
	if a.Cluster.Annotations[resourceKey] != "" {
		err := action()
		if err != nil {
			logger.Error("delete resource %s failed err: %s", resourceKey, err)
		} else {
			logger.Info("delete resource Success %s", a.Cluster.Annotations[resourceKey])
		}
	}
}

func CreateInstanceTag(tags map[string]string) (instanceTags []ecs.RunInstancesTag) {
	for k, v := range tags {
		instanceTags = append(instanceTags, ecs.RunInstancesTag{Key: k, Value: v})
	}
	return
}

func GetAKSKFromEnv(config *Config) error {
	config.AccessKey = os.Getenv(AccessKey)
	config.AccessSecret = os.Getenv(AccessSecret)
	config.RegionID = os.Getenv(RegionID)
	if config.RegionID == "" {
		config.RegionID = DefaultReigonID
	}
	if config.AccessKey == "" || config.AccessSecret == "" || config.RegionID == "" {
		return fmt.Errorf("please set accessKey and accessKeySecret ENV, example: export ACCESSKEYID=xxx export ACCESSKEYSECRET=xxx , how to get AK SK: https://ram.console.aliyun.com/manage/ak")
	}
	return nil
}
