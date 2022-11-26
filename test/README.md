# About

The code in this repository is a set of [Ginkgo](http://onsi.github.io/ginkgo)
and [Gomega](http://onsi.github.io/gomega) based integration tests that execute commands using the sealer CLI.

## Prerequisites

Before you run the tests, you'll need a sealer binary in your machine executable path and install docker. Follow the
instructions [here](https://github.com/sealerio/sealer#readme) to get one.

## Run the Tests

To run a single test or set of tests, you'll need the [Ginkgo](https://github.com/onsi/ginkgo) tool installed on your
machine:

```console
go get github.com/onsi/ginkgo/ginkgo
```

To install Sealer and prepare the test environment:

```console
#install Sealer binaries
wget https://github.com/sealerio/sealer/releases/download/v0.8.6/sealer-v0.8.6-linux-amd64.tar.gz && \
tar zxvf sealer-v0.8.6-linux-amd64.tar.gz && mv sealer /usr/bin

#or you can build sealer source code to binary for local e2e-test in container
make build-in-docker
cp _output/bin/sealer/linux_amd64/sealer /usr/local/bin

#prepare test environment
export REGISTRY_URL={your registry}
export REGISTRY_USERNAME={user name}
export REGISTRY_PASSWORD={password}
#default test image name: registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.22.4-test
export IMAGE_NAME={test image name}
```

To execute the entire test suite:

```console
cd sealer && ginkgo test
```

You can then use the `--focus` option to run subsets of the test suite:

```console
ginkgo --focus="sealer login" test
```

You can then use the `-v` option to print out default reporter as all specs begin:

```console
ginkgo -v test
```

More ginkgo helpful info see:

```console
ginkgo --help
```
