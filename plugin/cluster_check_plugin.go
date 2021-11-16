package plugin

import (
	goContext "context"
	"fmt"
	"time"

	"github.com/alibaba/sealer/client/k8s"
	"github.com/alibaba/sealer/logger"
	"golang.org/x/net/context"
)

type ClusterChecker struct {
	client *k8s.Client
}

func NewClusterCheckerPlugin() Interface {
	return &ClusterChecker{}
}

func (c *ClusterChecker) Run(context Context, phase Phase) error {
	if phase != PhasePreGuest || context.Plugin.Spec.Type != ClusterCheckPlugin {
		logger.Debug("check cluster is PreGuest!")
		return nil
	}
	if err := c.waitClusterReady(goContext.TODO()); err != nil {
		return err
	}
	return nil
}

func (c *ClusterChecker) waitClusterReady(ctx goContext.Context) error {
	var clusterStatusChan = make(chan string)
	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	go func(t *time.Ticker) {
		for {
			clusterStatus := c.getClusterStatus()
			clusterStatusChan <- clusterStatus
			<-t.C
		}
	}(ticker)
	for {
		select {
		case status := <-clusterStatusChan:
			if status == ClusterNotReady {
				logger.Info("wait for the cluster to ready ")
			} else if status == ClusterReady {
				logger.Info("cluster is ready now")
				return nil
			}
		case <-ctx.Done():
			return fmt.Errorf("cluster is not ready, please check")
		}
	}
}

func (c *ClusterChecker) getClusterStatus() string {
	k8sClient, err := k8s.Newk8sClient()
	c.client = k8sClient
	if err != nil {
		return ClusterNotReady
	}

	kubeSystemPodStatus, err := c.client.ListKubeSystemPodsStatus()
	if !kubeSystemPodStatus || err != nil {
		return ClusterNotReady
	}

	return ClusterReady
}
