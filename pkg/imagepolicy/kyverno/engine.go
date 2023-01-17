// Copyright © 2023 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kyverno

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"

	kyvernov1 "github.com/kyverno/kyverno/api/kyverno/v1"
	"github.com/sealerio/sealer/common"
	imagecommon "github.com/sealerio/sealer/pkg/define/options"
	"github.com/sealerio/sealer/pkg/imageengine"
	"github.com/sealerio/sealer/pkg/infradriver"
	"github.com/sealerio/sealer/pkg/rootfs"
	"github.com/sealerio/sealer/pkg/runtime"
	k "github.com/sealerio/sealer/pkg/runtime/kubernetes"
	apimachineryRuntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
)

type Engine struct {
	driver runtime.Driver
}

func NewKyvernoImagePolicyEngine() (*Engine, error) {
	driver, err := k.NewKubeDriver(k.AdminKubeConfPath)
	if err != nil {
		return nil, err
	}

	return &Engine{
		driver: driver,
	}, nil
}

// TODO: make it configurable
func (engine *Engine) IsImagePolicyApp(appName string) (bool, error) {
	return appName == "kyverno", nil
}

func (engine *Engine) CreateImagePolicyRule(infraDriver infradriver.InfraDriver, imageEngine imageengine.Interface, appName string) error {
	imagePolicyTemplateStr := common.ImagePolicyTemplate
	templatePath := filepath.Clean(filepath.Join(infraDriver.GetClusterRootfsPath(), rootfs.GlobalManager.App().Root(), appName, common.ImagePolicyTemplateYamlName))
	if _, err := os.Stat(templatePath); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	} else {
		yamlFile, err := ioutil.ReadFile(templatePath)
		if err != nil {
			return err
		}
		imagePolicyTemplateStr = string(yamlFile)
	}

	clusterImageName := infraDriver.GetClusterImageName()
	imageList, err := imageEngine.GetSealerContainerImageList(&imagecommon.GetImageAnnoOptions{ImageNameOrID: clusterImageName})
	clusterPolicy := &kyvernov1.ClusterPolicy{}

	var imageListArray []string
	for _, containerImage := range imageList {
		imageListArray = append(imageListArray, "\""+containerImage.Image+"\"")
	}

	ctx := map[string]string{
		"name":      clusterImageName,
		"imageList": "[" + strings.Join(imageListArray, ",") + "]",
		"registry":  infraDriver.GetClusterRegistry().LocalRegistry.Domain,
	}
	imagePolicyTemplate, err := SubsituteTemplate(imagePolicyTemplateStr, ctx)
	if err != nil {
		return err
	}

	if err := kyvernov1.AddToScheme(scheme.Scheme); err != nil {
		return err
	}
	decoder := serializer.NewCodecFactory(scheme.Scheme).UniversalDecoder()
	if err := apimachineryRuntime.DecodeInto(decoder, []byte(imagePolicyTemplate), clusterPolicy); err != nil {
		return err
	}

	if err := engine.driver.Create(context.Background(), clusterPolicy, &runtimeClient.CreateOptions{}); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("unable to create image policy: %v", err)
		}

		if err := engine.driver.Update(context.Background(), clusterPolicy, &runtimeClient.UpdateOptions{}); err != nil {
			return fmt.Errorf("unable to update image policy: %v", err)
		}
	}
	return nil
}
