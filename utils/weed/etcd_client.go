// Copyright © 2021 Alibaba Group Holding Ltd.
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

package weed

import (
	"context"
	"fmt"
	"time"

	etcd3 "go.etcd.io/etcd/client/v3"
)

// Client is the interface for etcd client.
// It provides the basic operations for etcd cluster.
// Like put, get, delete, register, unregister, get service.
type Client interface {

	// RegisterService register service to etcd cluster.
	RegisterService(serviceName string, endpoints string) error

	// UnRegisterService unregister service from etcd cluster.
	UnRegisterService(serviceName string, endpoints string) error

	// GetService get service from etcd cluster.
	GetService(serviceName string) ([]string, error)

	// Put put key-value to etcd cluster.
	Put(key, value string) error

	// Get get key-value from etcd cluster.
	Get(key string) (string, error)

	// Delete delete key-value from etcd cluster.
	Delete(key string) error
}

type client struct {
	peers  []string
	client *etcd3.Client
	ctx    context.Context
	lease  etcd3.Lease
}

func (c *client) Put(key, value string) error {
	_, err := c.client.Put(c.ctx, key, value)
	if err != nil {
		return err
	}
	return nil
}

func (c *client) Get(key string) (string, error) {
	resp, err := c.client.Get(c.ctx, key)
	if err != nil {
		return "", err
	}
	return string(resp.Kvs[0].Value), nil
}

func (c *client) Delete(key string) error {
	_, err := c.client.Delete(c.ctx, key)
	if err != nil {
		return err
	}
	return nil
}

func (c *client) RegisterService(serviceName string, endpoint string) error {
	c.lease = etcd3.NewLease(c.client)
	grant, err := c.lease.Grant(c.ctx, int64(10*time.Second))
	if err != nil {
		return err
	}
	key := fmt.Sprintf("/services/%s/%s", serviceName, endpoint)
	_, err = c.client.Put(context.Background(), key, endpoint, etcd3.WithLease(grant.ID))
	keepAliceCh, err := c.client.KeepAlive(context.Background(), grant.ID)
	if err != nil {
		return err
	}
	go c.doKeepAlive(keepAliceCh)
	return err
}

func (c *client) UnRegisterService(serviceName string, endpoint string) error {
	key := fmt.Sprintf("/services/%s/%s", serviceName, endpoint)
	_, err := c.client.Delete(c.ctx, key)
	if c.lease != nil {
		_ = c.lease.Close()
	}
	return err
}

// doKeepAlive continuously keeps alive the lease from ETCD.
func (c *client) doKeepAlive(keepAliceCh <-chan *etcd3.LeaseKeepAliveResponse) {
	for {
		select {
		case <-c.client.Ctx().Done():
			return

		case res, ok := <-keepAliceCh:
			if res != nil {
			}
			if !ok {
				return
			}
		}
	}
}

func (c *client) GetService(serviceName string) ([]string, error) {
	key := fmt.Sprintf("/services/%s", serviceName)
	response, err := c.client.Get(c.ctx, key, etcd3.WithPrefix())
	if err != nil {
		return nil, err
	}
	res := make([]string, 0)
	for _, v := range response.Kvs {
		res = append(res, string(v.Value))
	}
	return res, nil
}

func NewClient(peers []string) (Client, error) {
	c, err := etcd3.New(etcd3.Config{
		Endpoints: peers,
	})
	if err != nil {
		return nil, err
	}
	return &client{
		peers:  peers,
		client: c,
		ctx:    context.Background(),
	}, nil
}
