package lite

import (
	"bufio"
	"strings"

	"github.com/sirupsen/logrus"
)

// decode image from yaml content
func DecodeImages(body string) []string {
	var list []string

	reader := strings.NewReader(body)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		l := decodeLine(scanner.Text())
		if l != "" {
			list = append(list, l)
		}
	}
	if err := scanner.Err(); err != nil {
		logrus.Errorf(err.Error())
		return list
	}

	return list
}

func decodeLine(line string) string {
	l := strings.Replace(line, `"`, "", -1)
	ss := strings.SplitN(l, ":", 2)
	if len(ss) != 2 {
		return ""
	}
	if !strings.HasSuffix(ss[0], "image") || strings.Contains(ss[0], "#") {
		return ""
	}

	return strings.Replace(ss[1], " ", "", -1)
}
