package test

import (
	"os/exec"
	"testing"

	"github.com/alibaba/sealer/test/testhelper/settings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestSealerTests(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "run sealer suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	output, err := exec.LookPath("sealer")
	Expect(err).NotTo(HaveOccurred(), output)
	SetDefaultEventuallyTimeout(settings.DefaultWaiteTime)
	return nil
}, func(data []byte) {
	SetDefaultEventuallyTimeout(settings.DefaultWaiteTime)
})
