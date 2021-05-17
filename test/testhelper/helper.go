package testhelper

import (
	"errors"
	"fmt"
	"io"
	"os/exec"

	"github.com/alibaba/sealer/test/testhelper/settings"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

// Start sealer cmd and return *gexec.Session
func Start(cmdLine string) (*gexec.Session, error) {
	if cmdLine == "" {
		return nil, errors.New("failed to start cmd, line is empty")
	}
	execCmd := exec.Command("/bin/sh", "-c", cmdLine)
	_, err := io.WriteString(ginkgo.GinkgoWriter, fmt.Sprintf("%s\n", cmdLine))
	if err != nil {
		return nil, err
	}
	return gexec.Start(execCmd, ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)
}

// RunCmdAndCheckResult start cmd and check expectedCode
func RunCmdAndCheckResult(cmdLine string, expectedCode int) *gexec.Session {
	sess, err := Start(cmdLine)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Eventually(sess, settings.MaxWaiteTime).Should(gexec.Exit(expectedCode))
	return sess
}
