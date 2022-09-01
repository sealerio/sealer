// Copyright © 2022 Alibaba Group Holding Ltd.
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

package kubernetes

import (
	"context"
	"fmt"
	"github.com/sealerio/sealer/pkg/runtime"
	"k8s.io/apimachinery/pkg/api/meta"
	k8sRuntime "k8s.io/apimachinery/pkg/runtime"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
)

type kubeDriver struct {
	client     runtimeClient.Client
	kubeConfig string
}

func NewKubeDriver(kubeConfig string) (runtime.Driver, error) {
	client, err := GetClientFromConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to new k8s runtime client via adminconf: %v", err)
	}

	return kubeDriver{
		kubeConfig: kubeConfig,
		client:     client,
	}, nil
}

// Get retrieves an obj for the given object key from the Kubernetes Cluster.
// obj must be a struct pointer so that obj can be updated with the response
// returned by the Server.
func (k kubeDriver) Get(ctx context.Context, key runtimeClient.ObjectKey, obj runtimeClient.Object) error {
	return k.client.Get(ctx, key, obj)
}

// List retrieves list of objects for a given namespace and list options. On a
// successful call, Items field in the list will be populated with the
// result returned from the server.
func (k kubeDriver) List(ctx context.Context, list runtimeClient.ObjectList, opts ...runtimeClient.ListOption) error {
	return k.client.List(ctx, list, opts...)
}

// Create saves the object obj in the Kubernetes cluster.
func (k kubeDriver) Create(ctx context.Context, obj runtimeClient.Object, opts ...runtimeClient.CreateOption) error {
	return k.client.Create(ctx, obj, opts...)
}

// Delete deletes the given obj from Kubernetes cluster.
func (k kubeDriver) Delete(ctx context.Context, obj runtimeClient.Object, opts ...runtimeClient.DeleteOption) error {
	return k.client.Delete(ctx, obj, opts...)
}

// Update updates the given obj in the Kubernetes cluster. obj must be a
// struct pointer so that obj can be updated with the content returned by the Server.
func (k kubeDriver) Update(ctx context.Context, obj runtimeClient.Object, opts ...runtimeClient.UpdateOption) error {
	return k.client.Update(ctx, obj, opts...)
}

// Patch patches the given obj in the Kubernetes cluster. obj must be a
// struct pointer so that obj can be updated with the content returned by the Server.
func (k kubeDriver) Patch(ctx context.Context, obj runtimeClient.Object, patch runtimeClient.Patch, opts ...runtimeClient.PatchOption) error {
	return k.client.Patch(ctx, obj, patch, opts...)
}

// DeleteAllOf deletes all objects of the given type matching the given options.
func (k kubeDriver) DeleteAllOf(ctx context.Context, obj runtimeClient.Object, opts ...runtimeClient.DeleteAllOfOption) error {
	return k.client.DeleteAllOf(ctx, obj, opts...)
}

// Status knows how to update status subresource of a Kubernetes object.
func (k kubeDriver) Status() runtimeClient.StatusWriter {
	return k.client.Status()
}

// Scheme returns the scheme this client is using.
func (k kubeDriver) Scheme() *k8sRuntime.Scheme {
	return k.client.Scheme()
}

// RESTMapper returns the rest this client is using.
func (k kubeDriver) RESTMapper() meta.RESTMapper {
	return k.client.RESTMapper()
}

// GetAdminKubeconfig returns the file path of admin kubeconfig is using.
func (k kubeDriver) GetAdminKubeconfig() string {
	return k.kubeConfig
}
