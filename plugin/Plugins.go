package plugin

import (
	"fmt"
	v1 "github.com/alibaba/sealer/types/api/v1"
)

type Plugins struct {
	pluginList v1.PluginList
}

func (p Plugins) Run(context Context, phase Phase) error {
	for _, aplugin := range p.pluginList.Items {
		if phase == Phase(aplugin.Spec.On[5:]) {
			switch aplugin.Name {
			case "LABEL":
				l := LabelsNodes{}
				err := l.Run(context, phase)
				if err != nil {
					return err
				}
				break
			default:
				return fmt.Errorf("not find plugib")
			}
		}
	}
	return nil
}
func (p Plugins) AddPluginConfigs(plugins []v1.Plugin) {
	p.pluginList = v1.PluginList{
		Items: plugins,
	}
}

func NewPlugins() Interface {
	return &Plugins{}
}
