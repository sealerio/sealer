package store

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	"gotest.tools/skip"
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
		err = ioutil.WriteFile(filepath.Join(layer.tmpRelPath, file), []byte(fileContent), common.FileMode0644)
		if err != nil {
			return err
		}
	}

	return nil
}

func cleanTmpLayers(layer mockROLayer) {
	_ = os.Remove(layer.tmpRelPath)
	lsg := NewDefaultLayerStorage()
	err := os.RemoveAll(lsg.LayerDataDir(layer.roLayer.id.ToDigest()))
	if err != nil {
		logger.Warn(err)
	}

	err = os.RemoveAll(lsg.LayerDBDir(layer.roLayer.id.ToDigest()))
	if err != nil {
		logger.Warn(err)
	}
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
			cleanTmpLayers(layer)
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
