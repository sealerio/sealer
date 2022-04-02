module github.com/alibaba/sealer

go 1.14

require (
	github.com/Masterminds/semver/v3 v3.1.1
	github.com/Masterminds/sprig/v3 v3.2.2
	github.com/aliyun/alibaba-cloud-sdk-go v1.61.985
	github.com/cavaliergopher/grab/v3 v3.0.1
	github.com/distribution/distribution/v3 v3.0.0-20211125133600-cc4627fc6e5f
	github.com/docker/cli v20.10.7+incompatible
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v20.10.7+incompatible
	github.com/docker/go-connections v0.4.0
	github.com/docker/libtrust v0.0.0-20160708172513-aabc10ec26b7 // indirect
	github.com/go-git/go-git/v5 v5.4.2
	github.com/imdario/mergo v0.3.12
	github.com/mitchellh/go-homedir v1.1.0
	github.com/moby/buildkit v0.9.3
	github.com/moby/term v0.0.0-20201216013528-df9cb8a40635
	github.com/olekukonko/tablewriter v0.0.4
	github.com/onsi/ginkgo v1.16.2
	github.com/onsi/gomega v1.12.0
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.0.2
	github.com/pkg/errors v0.9.1
	github.com/pkg/sftp v1.13.0
	github.com/sealyun/lvscare v1.1.2-alpha.2
	github.com/shirou/gopsutil v3.21.11+incompatible
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.1.3
	github.com/spf13/viper v1.7.0
	github.com/tonistiigi/fsutil v0.0.0-20211208191308-f95797418e48
	github.com/vbatts/tar-split v0.11.1
	github.com/vishvananda/netlink v1.1.1-0.20201029203352-d40f9887b852
	github.com/vishvananda/netns v0.0.0-20211101163701-50045581ed74 // indirect
	github.com/wonderivan/logger v1.0.0
	github.com/yusufpapurcu/wmi v1.2.2 // indirect
	go.etcd.io/etcd/client/v3 v3.5.0
	go.uber.org/zap v1.17.0
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5
	golang.org/x/net v0.0.0-20210510120150-4163338589ed
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20220114195835-da31bd327af9
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
	gotest.tools v2.2.0+incompatible
	helm.sh/helm/v3 v3.6.2
	k8s.io/api v0.21.0
	k8s.io/apimachinery v0.21.0
	k8s.io/cli-runtime v0.21.0
	k8s.io/client-go v0.21.0
	k8s.io/kube-proxy v0.21.0
	k8s.io/kubelet v0.21.0
	sigs.k8s.io/controller-runtime v0.8.1
	sigs.k8s.io/yaml v1.2.0
)

replace github.com/docker/docker => github.com/docker/docker v20.10.3-0.20211208011758-87521affb077+incompatible
