package command

import (
	"github.com/alibaba/sealer/utils"
)

type Interface interface {
	Exec() (string, error)
}

type SimpleCommand struct {
	Command string
}

func (c *SimpleCommand) Exec() (string, error) {
	output, err := utils.RunSimpleCmd(c.Command)
	if err != nil {
		return "", err
	}
	return output, err
}

func NewSimpleCommand(cmd string) Interface {
	return &SimpleCommand{
		Command: cmd,
	}
}

type CopyCommand struct {
	Src string
	Dst string
}

// COPY dashboard-chart .
// COPY [src] [dst]
// copy files to /var/lib/seadent/[image hash]/[layer hash]
func (c *CopyCommand) Exec() (string, error) {
	output, err := utils.CmdOutput("cp", "-r", c.Src, c.Dst)
	if err != nil {
		return "", err
	}
	return string(output), err
}

func NewCopyCommand(src, dst string) Interface {
	return &CopyCommand{
		Src: src,
		Dst: dst,
	}
}
