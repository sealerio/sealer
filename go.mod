module github.com/sealerio/sealer

go 1.14

require (
	github.com/BurntSushi/toml v1.0.0
	github.com/Masterminds/semver/v3 v3.1.1
	github.com/aliyun/alibaba-cloud-sdk-go v1.61.985
	github.com/cavaliergopher/grab/v3 v3.0.1
	github.com/containers/buildah v1.25.0
	github.com/containers/common v0.47.5
	github.com/containers/image/v5 v5.20.0
	github.com/containers/ocicrypt v1.1.4
	github.com/containers/storage v1.39.0
	github.com/distribution/distribution/v3 v3.0.0-20211125133600-cc4627fc6e5f
	github.com/docker/cli v20.10.7+incompatible
	github.com/docker/distribution v2.8.1+incompatible
	github.com/docker/docker v20.10.17+incompatible
	github.com/docker/go-connections v0.4.1-0.20210727194412-58542c764a11
	github.com/docker/go-units v0.4.0
	github.com/go-git/go-git/v5 v5.4.2
	github.com/google/uuid v1.3.0
	github.com/hashicorp/go-multierror v1.1.1
	github.com/imdario/mergo v0.3.12
	github.com/lestrrat-go/file-rotatelogs v2.4.0+incompatible
	github.com/lestrrat-go/strftime v1.0.6 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/moby/buildkit v0.9.3
	github.com/moby/term v0.0.0-20210619224110-3f7ff695adc6
	github.com/olekukonko/tablewriter v0.0.4
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.19.0
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.0.3-0.20211202193544-a5463b7f9c84
	github.com/pkg/errors v0.9.1
	github.com/pkg/sftp v1.13.0
	github.com/rifflock/lfshook v0.0.0-20180920164130-b9218ef580f5
	github.com/sealyun/lvscare v1.1.2-alpha.2
	github.com/shirou/gopsutil v3.21.11+incompatible
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.4.0
	github.com/spf13/viper v1.10.0
	github.com/stretchr/testify v1.7.1
	github.com/tonistiigi/fsutil v0.0.0-20210609172227-d72af97c0eaf
	github.com/vishvananda/netlink v1.1.1-0.20210330154013-f5de75959ad5
	github.com/vishvananda/netns v0.0.0-20211101163701-50045581ed74 // indirect
	github.com/yusufpapurcu/wmi v1.2.2 // indirect
	go.etcd.io/etcd/client/v3 v3.5.0
	go.uber.org/zap v1.17.0
	golang.org/x/crypto v0.0.0-20220411220226-7b82a4e95df4
	golang.org/x/net v0.0.0-20220225172249-27dd8689420f
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20220422013727-9388b58f7150
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
	helm.sh/helm/v3 v3.6.2
	k8s.io/api v0.22.5
	k8s.io/apimachinery v0.22.5
	k8s.io/cli-runtime v0.21.0
	k8s.io/client-go v0.22.5
	k8s.io/kube-proxy v0.21.0
	k8s.io/kubelet v0.21.0
	k8s.io/utils v0.0.0-20210930125809-cb0fa318a74b
	sigs.k8s.io/controller-runtime v0.8.1
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/containerd/containerd => github.com/containerd/containerd v1.6.6
	github.com/opencontainers/runc => github.com/opencontainers/runc v1.1.2
	golang.org/x/crypto => golang.org/x/crypto v0.0.0-20210817164053-32db794688a5
	golang.org/x/net => golang.org/x/net v0.0.0-20210510120150-4163338589ed
	golang.org/x/sys => golang.org/x/sys v0.0.0-20220114195835-da31bd327af9
	k8s.io/api => k8s.io/api v0.21.0
	k8s.io/apimachinery => k8s.io/apimachinery v0.21.0
	k8s.io/client-go => k8s.io/client-go v0.21.0
	k8s.io/utils => k8s.io/utils v0.0.0-20210111153108-fddb29f9d009
)
