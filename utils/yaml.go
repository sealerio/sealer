package utils

import (
	"io/ioutil"

	"sigs.k8s.io/yaml"
)

func UnmarshalYamlFile(file string, obj interface{}) error {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(data, obj)
	return err
}

func MarshalYamlToFile(file string, obj interface{}) error {
	data, err := yaml.Marshal(obj)
	if err != nil {
		return err
	}

	if err = WriteFile(file, data); err != nil {
		return err
	}
	return nil
}
