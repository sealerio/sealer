package parser

import (
	"bufio"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"
)

var validLayer = []string{"FROM", "COPY", "RUN", "CMD"}

type Interface interface {
	Parse(kubefile []byte, name string) *v1.Image
}

type Parser struct{}

func NewParse() Interface {
	return &Parser{}
}

func (p *Parser) Parse(kubefile []byte, name string) *v1.Image {
	image := &v1.Image{
		TypeMeta:   metav1.TypeMeta{APIVersion: "", Kind: "Image"},
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec:       v1.ImageSpec{},
		Status:     v1.ImageStatus{},
	}
	scanner := bufio.NewScanner(strings.NewReader(string(kubefile)))
	for scanner.Scan() {
		text := scanner.Text()
		layerType, layerValue, err := decodeLine(text)
		if err != nil || layerType == "" {
			logger.Warn("decode kubefile line failed %v", err)
			continue
		}

		//TODO count layer hash
		image.Spec.Layers = append(image.Spec.Layers, v1.Layer{
			Hash:  "",
			Type:  layerType,
			Value: layerValue,
		})
	}
	return image
}

func decodeLine(line string) (string, string, error) {
	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", nil
	}
	//line = strings.TrimPrefix(line, " ")
	ss := strings.SplitN(line, " ", 2)
	if len(ss) != 2 {
		return "", "", fmt.Errorf("unknown line %s", line)
	}
	var flag bool
	for _, v := range validLayer {
		if ss[0] == v {
			flag = true
		}
	}
	if !flag {
		return "", "", fmt.Errorf("invalid command %s %s", ss[0], line)
	}

	return ss[0], strings.TrimSpace(ss[1]), nil
}
