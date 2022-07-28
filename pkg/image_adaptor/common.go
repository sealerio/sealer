package image_adaptor

import (
	buildahcli "github.com/containers/buildah/pkg/cli"
	"github.com/sealerio/sealer/pkg/image_adaptor/buildah"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

// TODO do we have an util or unified local storage accessing pattern?
func writeFileIfNotExist(path string, content []byte) error {
	_, err := os.Stat(path)
	if err != nil {
		err = os.MkdirAll(filepath.Dir(path), 0755)
		if err != nil {
			return err
		}

		err = os.WriteFile(path, []byte(content), 0644)
		if err != nil {
			return err
		}
	}
	return nil
}

func NewAdaptor() (Interface, error) {
	err := initBuildah()
	if err != nil {
		return nil, err
	}
	return &buildah.Adaptor{
		BudResults:        &buildahcli.BudResults{},
		LayerResults:      &buildahcli.LayerResults{},
		FromAndBudResults: &buildahcli.FromAndBudResults{},
		NameSpaceResults:  &buildahcli.NameSpaceResults{},
		UserNSResults:     &buildahcli.UserNSResults{},
		Command:           &cobra.Command{},
	}, nil
}
