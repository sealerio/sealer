package utils

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func Cmd(name string, args ...string) error {
	cmd := exec.Command(name, args[:]...)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

func CmdOutput(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args[:]...)
	return cmd.CombinedOutput()
}

func RunSimpleCmd(cmd string) (string, error) {
	result, err := exec.Command("/bin/sh", "-c", cmd).Output()
	if err != nil {
		return "", err
	}
	return string(result), nil
}

func CheckCmdIsExist(cmd string) (string, bool) {
	cmd = fmt.Sprintf("type %s", cmd)
	out, err := RunSimpleCmd(cmd)
	if err != nil {
		return "", false
	}

	outSlice := strings.Split(out, "is")
	last := outSlice[len(outSlice)-1]

	if last != "" && !strings.Contains(last, "not found") {
		return strings.TrimSpace(last), true
	}
	return "", false
}
