package guest

import (
	"bytes"
	"fmt"
	"text/template"
)

const (
	CALICO           = "calico"
	defaultCIDR      = "192.168.0.0/16"
	defaultInterface = "interface=eth.*|en.*"
)

type MetaData struct {
	//set IP auto-detection method
	Interface string
	CIDR      string
	//ipip mode for calico.yml,cannot be set at the same time as `vxlan`
	IPIP bool
	/*vxlan mode for calico.yml,cannot be set at the same time as `ipip`
	VXLAN bool*/
	//MTU size
	MTU string
}

type Net interface {
	// if template is "" using default template
	Manifests(template string) (string, error)
	// return cni template file
	Template() string
}

func render(data MetaData, templ string) (string, error) {
	var b bytes.Buffer
	t := template.Must(template.New("net").Parse(templ))
	if err := t.Execute(&b, &data); err != nil {
		return "", fmt.Errorf("render execute failed, %s", err)
	}
	return b.String(), nil
}

func NewNetWork(netName string, metaData MetaData) Net {
	switch netName {
	case CALICO:
		return &Calico{metaData}
	default:
		return &Calico{metaData}
	}
}
