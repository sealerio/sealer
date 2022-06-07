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

package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"gotest.tools/skip"

	"github.com/sealerio/sealer/common"
	osUtils "github.com/sealerio/sealer/utils/os"
)

const fileContent = "fake file content"

type mockROLayer struct {
	roLayer    ROLayer
	tmpRelPath string
	files      []string
}

var layers = []mockROLayer{
	{
		tmpRelPath: "/tmp/layerstore-test/a",
		files:      []string{"a", "b"},
	},
	{
		tmpRelPath: "/tmp/layerstore-test/b",
		files:      []string{"b", "c"},
	},
	{
		tmpRelPath: "/tmp/layerstore-test/c",
		files:      []string{"d", "e"},
	},
	{
		tmpRelPath: "/tmp/layerstore-test/d",
		files:      []string{"f", "g"},
	},
}

func makeFakeLayer(layer mockROLayer) error {
	err := os.MkdirAll(layer.tmpRelPath, common.FileMode0755)
	if err != nil {
		return err
	}

	for _, file := range layer.files {
		if err = osUtils.NewAtomicWriter(filepath.Join(layer.tmpRelPath, file)).WriteFile([]byte(fileContent)); err != nil {
			return err
		}
	}

	return nil
}

func cleanTmpLayers(layer mockROLayer) error {
	_ = os.Remove(layer.tmpRelPath)
	backend, err := NewFSStoreBackend()
	if err != nil {
		return err
	}
	err = os.RemoveAll(backend.LayerDataDir(layer.roLayer.id.ToDigest()))
	if err != nil {
		logrus.Warn(err)
	}

	err = os.RemoveAll(backend.LayerDBDir(layer.roLayer.id.ToDigest()))
	if err != nil {
		logrus.Warn(err)
	}
	return nil
}

func TestLayerStore_RegisterLayerForBuilder(t *testing.T) {
	skip.If(t, os.Getuid() != 0, "skipping test that requires root")

	var err error
	for _, layer := range layers {
		err = makeFakeLayer(layer)
		if err != nil {
			t.Errorf("failed to make layers, err: %s", err)
		}
	}

	err = os.MkdirAll(layerDataRoot, common.FileMode0755)
	if err != nil {
		t.Error(err)
	}
	err = os.MkdirAll(layerDBRoot, common.FileMode0755)
	if err != nil {
		t.Error(err)
	}

	ls, err := NewDefaultLayerStore()
	if err != nil {
		t.Errorf("failed to get layer store, err: %s", err)
	}

	newLayers := []mockROLayer{}
	layerExists := map[string]bool{}
	for _, layer := range layers {
		layerID, err := ls.RegisterLayerForBuilder(layer.tmpRelPath)
		if err != nil {
			t.Errorf("failed to registry layer %s, err: %s", layer.tmpRelPath, err)
		}
		layer.roLayer = ROLayer{id: LayerID(layerID)}
		newLayers = append(newLayers, layer)
		layerExists[layerID.String()] = true
	}

	defer func() {
		for _, layer := range newLayers {
			err := cleanTmpLayers(layer)
			if err != nil {
				logrus.Error(err)
			}
		}
	}()

	lls, ok := ls.(*layerStore)
	if !ok {
		t.Errorf("failed to convert to layerStore")
	}

	err = lls.loadAllROLayers()
	if err != nil {
		t.Errorf("failed to load layers, err: %s", err)
	}

	for key := range lls.layers {
		if layerExists[key.String()] {
			delete(layerExists, key.String())
		}
	}

	if len(layerExists) > 0 {
		t.Errorf("there are still some layer not load correctly")
	}
}
