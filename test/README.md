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
go install github.com/onsi/ginkgo/ginkgo@v1.16.2
```

To install Sealer and prepare the test environment:

```console
#build sealer source code to binary for local e2e-test in containers
git clone https://github.com/sealerio/sealer.git
cd sealer/ && make build-in-docker
cp _output/bin/sealer/linux_amd64/sealer /usr/local/bin

#prepare test environment
export REGISTRY_URL={your registry}
export REGISTRY_USERNAME={user name}
export REGISTRY_PASSWORD={password}
#default test image name: docker.io/sealerio/kubernetes:v1-22-15-sealerio-2
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
