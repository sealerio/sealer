# Contributing to Sealer

It is warmly welcomed if you have interest to hack on Sealer. First, we encourage this kind of willing very much. And here is a list of contributing guide for you.

## Topics

* [Reporting security issues](#reporting-security-issues)
* [Reporting general issues](#reporting-general-issues)
* [Code and doc contribution](#code-and-doc-contribution)
* [Engage to help anything](#engage-to-help-anything)

## Reporting security issues

Security issues are always treated seriously. As our usual principle, we discourage anyone to spread security issues. If you find a security issue of Sealer, please do not discuss it in public and even do not open a public issue. Instead, we encourage you to send us a private email to [sealer@list.alibaba-inc.com](mailto:sealer@list.alibaba-inc.com) to report this.

## Reporting general issues

To be honest, we regard every user of Sealer as a very kind contributor. After experiencing Sealer, you may have some feedback for the project. Then feel free to open an issue via [NEW ISSUE](https://github.com/sealerio/sealer/issues/new/choose).

Since we collaborate project Sealer in a distributed way, we appreciate **WELL-WRITTEN**, **DETAILED**, **EXPLICIT** issue reports. To make the communication more efficient, we wish everyone could search if your issue is an existing one in the searching list. If you find it existing, please add your details in comments under the existing issue instead of opening a brand new one.

To make the issue details as standard as possible, we set up an [ISSUE TEMPLATE](./.github/ISSUE_TEMPLATE) for issue reporters. You can find three kinds of issue templates there: question, bug report and feature request. Please **BE SURE** to follow the instructions to fill fields in template.

There are a lot of cases when you could open an issue:

* bug report
* feature request
* performance issues
* feature proposal
* feature design
* help wanted
* doc incomplete
* test improvement
* any questions on project
* and so on

Also, we must remind that when filing a new issue, please remember to remove the sensitive data from your post. Sensitive data could be password, secret key, network locations, private business data and so on.

## Code and doc contribution

Every action to make project Sealer better is encouraged. On GitHub, every improvement for Sealer could be via a PR (short for pull request).

* If you find a typo, try to fix it!
* If you find a bug, try to fix it!
* If you find some redundant codes, try to remove them!
* If you find some test cases missing, try to add them!
* If you could enhance a feature, please **DO NOT** hesitate!
* If you find code implicit, try to add comments to make it clear!
* If you find code ugly, try to refactor that!
* If you can help to improve documents, it could not be better!
* If you find document incorrect, just do it and fix that!
* ...

Actually it is impossible to list them completely. Just remember one principle:

> WE ARE LOOKING FORWARD TO ANY PR FROM YOU.

Since you are ready to improve Sealer with a PR, we suggest you could take a look at the PR rules here.

* [Workspace Preparation](#workspace-preparation)
* [Branch Definition](#branch-definition)
* [Commit Rules](#commit-rules)
* [PR Description](#pr-description)
* [Developing Environment](#developing-environment)
* [Run E2E Test or Write E2E Test Cases](#Run-E2E-Test-or-Write-E2E-Test-Cases)

### Workspace Preparation

To put forward a PR, we assume you have registered a GitHub ID. Then you could finish the preparation in the following steps:

1. **FORK** Sealer to your repository. To make this work, you just need to click the button Fork in right-left of [sealerio/sealer](https://github.com/sealerio/sealer) main page. Then you will end up with your repository in `https://github.com/<your-username>/sealer`, in which `your-username` is your GitHub username.

1. **CLONE** your own repository to develop locally. Use `git clone https://github.com/<your-username>/sealer.git` to clone repository to your local machine. Then you can create new branches to finish the change you wish to make.

1. **Set Remote** upstream to be `https://github.com/sealerio/sealer.git` using the following two commands:

```
	git remote add upstream https://github.com/sealerio/sealer.git
	git remote set-url --push upstream no-pushing
```

	With this remote setting, you can check your git remote configuration like this:

```
	$ git remote -v
	origin     https://github.com/<your-username>/sealer.git (fetch)
	origin     https://github.com/<your-username>/sealer.git (push)
	upstream   https://github.com/sealerio/sealer.git (fetch)
	upstream   no-pushing (push)
```

	Adding this, we can easily synchronize local branches with upstream branches.

1. **Create a branch** to add a new feature or fix issues

    Update local working directory and remote forked repository:

   ```
   cd sealer
   git fetch upstream
   git checkout main
   git rebase upstream/main
   git push	// default origin, update your forked repository
   ```

   Create a new branch:

   ```
   git checkout -b <new-branch>
   ```

   Make any change on the `new-branch` then build and test your codes.

1. **Push your branch** to your forked repository, try not to generate multiple commit message within a pr.

   ```
   golangci-lint run -c .golangci.yml	// lint
   git commit -a -m "message for your changes" --signoff	// -a is git add .
   git rebase -i	<commit-id>// do this if your pr has multiple commits
   git push	// push to your forked repository after rebase done
   ```

1. **File a pull request** to sealerio/sealer:main

### Branch Definition

Right now we assume every contribution via pull request is for [branch master](https://github.com/sealerio/sealer/tree/main) in Sealer. Before contributing, be aware of branch definition would help a lot.

As a contributor, keep in mind again that every contribution via pull request is for branch master. While in project sealer, there are several other branches, we generally call them rc branches, release branches and backport branches.

Before officially releasing a version, we will check out a rc(release candidate) branch. In this branch, we will test more than branch main.

When officially releasing a version, there will be a release branch before tagging. After tagging, we will delete the release branch.

When backport some fixes to existing released version, we will check out backport branches. After backporting, the backporting effects will be in PATCH number in MAJOR.MINOR.PATCH of [SemVer](http://semver.org/).

### Commit Rules

Actually in Sealer, we take two rules serious when committing:

* [Commit Message](#commit-message)
* [Commit Content](#commit-content)

#### Commit Message

Commit message could help reviewers better understand what the purpose of submitted PR is. It could help accelerate the code review procedure as well. We encourage contributors to use **EXPLICIT** commit message rather than ambiguous message. In general, we advocate the following commit message type:

* docs: xxxx. For example, "docs: add docs about storage installation".
* feature: xxxx.For example, "feature: make result show in sorted order".
* bugfix: xxxx. For example, "bugfix: fix panic when input nil parameter".
* style: xxxx. For example, "style: format the code style of Constants.java".
* refactor: xxxx. For example, "refactor: simplify to make codes more readable".
* test: xxx. For example, "test: add unit test case for func InsertIntoArray".
* chore: xxx. For example, "chore: integrate travis-ci". It's the type of mantainance change.
* other readable and explicit expression ways.

On the other side, we discourage contributors from committing message like the following ways:

* ~~fix bug~~
* ~~update~~
* ~~add doc~~

#### Commit Content

Commit content represents all content changes included in one commit. We had better include things in one single commit which could support reviewer's complete review without any other commits' help. In another word, contents in one single commit can pass the CI to avoid code mess. In brief, there are two minor rules for us to keep in mind:

* avoid very large change in a commit;
* complete and reviewable for each commit.

No matter what the commit message, or commit content is, we do take more emphasis on code review.

### PR Description

PR is the only way to make change to Sealer project files. To help reviewers better get your purpose, PR description could not be too detailed. We encourage contributors to follow the [PR template](./.github/PULL_REQUEST_TEMPLATE.md) to finish the pull request.

### Developing Environment

As a contributor, if you want to make any contribution to Sealer project, we should reach an agreement on the version of tools used in the development environment.
Here are some dependents with specific version:

* golang : v1.14
* golangci-lint: 1.39.0
* gpgme(brew install gpgme)

When you develop the Sealer project at the local environment, you should use subcommands of Makefile to help yourself to check and build the latest version of Sealer. For the convenience of developers, we use the docker to build Sealer. It can reduce problems of the developing environment.

### Run E2E Test or Write E2E Test Cases
Before you commit a pull request, you need to run E2E test first. If you have modified the code or added features, you need to write corresponding E2E test cases, or you want to contribute E2E test cases for sealer, please read this section.

#### Principle of sealer k8s in docker

Sealer E2E test is inspired by kind, uses a container to emulate a node by running systemd inside the container and hosting other processes with systemd.
Sealer uses [pkg/infra/container/imagecontext/base/Dockerfile](https://github.com/sealerio/sealer/blob/main/pkg/infra/container/imagecontext/base/Dockerfile) to build a basic image: `sealerio/sealer-base-image:v1`，and use this image to start the container to emulate node。

After starting the container, the sealer binary is transferred to the container through ssh, and the sealer commands such as: sealer run and sealer apply are invoked through ssh to create the k8s cluster (k8s in docker), as shown in the figure below:

![sealer E2E drawio (1)](https://user-images.githubusercontent.com/56665618/217173635-3f27033d-3cf5-47e2-9b4c-66173eb89821.png)

#### How to run E2E test locally

The code in test repository is a set of [Ginkgo](http://onsi.github.io/ginkgo) and [Gomega](http://onsi.github.io/gomega) based integration tests that execute commands using the sealer CLI.

* Prerequisites

Before you run the tests, you'll need a sealer binary in your machine executable path and install docker.

* Run the Tests

To run a single test or set of tests, you'll need the [Ginkgo](https://github.com/onsi/ginkgo) tool installed on your machine:

```bash
go install github.com/onsi/ginkgo/ginkgo@v1.16.2
```

To install Sealer and prepare the test environment:

```bash
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

```bash
cd sealer && ginkgo test
```

You can then use the `--focus` option to run subsets of the test suite:

```bash
ginkgo --focus="sealer login" test
```

You can then use the `-v` option to print out default reporter as all specs begin:

```bash
ginkgo -v test
```

More ginkgo helpful info see:

```bash
ginkgo --help
```

#### How to run E2E Tests using github action

Before we run the CI by using github aciton, please make sure that these four variables: `REGISTRY_URL={your registry}`, `REGISTRY_USERNAME={user name}`, `REGISTRY_PASSWORD={password}`, `IMAGE_NAME={test image name}` have been setted in github's secret (for sealer login and sealer image tests).

CI is triggered when we push a branch with name starts with release or comment `/test all` in PR，you can also comment `/test {test to run}` in PR to run subsets of the test suite.

#### How to write E2E test cases for sealer

E2E tests required by sealer can be divided into those that need cluster construction and those that do not. For tests that need cluster construction, container needs to be started to simulate node (such as sealer run and sealer apply). If you don't need to build a cluster, you just need to execute it on the machine (e.g., sealer pull, sealer tag, sealer build).

* E2E test entry

Sealer all E2E test files are in the `test` directory, where `e2e_test.go` is the entry to E2E test.

The function： `func TestSealerTests(t *testing.T)`is the entry point for Ginkgo - the go test runner will run this function when you run `go test` or `ginkgo`. `RegisterFailHandler(Fail)`is the single line of glue code connecting Ginkgo to Gomega. If we were to avoid dot-imports this would read as `gomega.RegisterFailHandler(ginkgo.Fail)`- what we're doing here is telling our matcher library (Gomega) which function to call (Ginkgo's `Fail`) in the event a failure is detected.

* Writing Specs

You can use `ginkgo generate` to generate a E2E test file (eg. `ginkgo generate sealer_alpha`)

Ginkgo allows you to hierarchically organize the specs in your suite using container nodes. Ginkgo provides three synonymous nouns for creating container nodes: `Describe`, `Context`, and `When`. These three are functionally identical and are provided to help the spec narrative flow. You usually `Describe`different capabilities of your code and explore the behavior of each capability across different `Context`s.

For example:

```go
var _ = Describe("sealer login", func() {
	Context("login docker registry", func() {
		AfterEach(func() {
			registry.Logout()
		})
		It("with correct name and password", func() {
			image.CheckLoginResult(
				settings.RegistryURL,
				settings.RegistryUsername,
				settings.RegistryPasswd,
				true)
		})
		It("with incorrect name and password", func() {
			image.CheckLoginResult(
				settings.RegistryURL,
				settings.RegistryPasswd,
				settings.RegistryUsername,
				false)
		})
		It("with only name", func() {
			image.CheckLoginResult(
				settings.RegistryURL,
				settings.RegistryUsername,
				"",
				false)
		})
		It("with only password", func() {
			image.CheckLoginResult(
				settings.RegistryURL,
				"",
				settings.RegistryPasswd,
				false)
		})
		It("with only registryURL", func() {
			image.CheckLoginResult(
				settings.RegistryURL,
				"",
				"",
				false)
		})
	})
})
```

The configuration of e2e test is in `test/testhelper/settings`, The test data and methods for e2e test are here: `test/suites`.

More information about ginkgo and gomega can be found in the documentation:[Ginkgo](http://onsi.github.io/ginkgo) and [Gomega](http://onsi.github.io/gomega).

#### Configure github action file

The github action file for e2e test is under the `.github/workflows` directory. You need to configure the file here for CI.

For example:

```yaml
name: {Test name}

on:
  push:
    branches: "release*"
  issue_comment:
    types:
      - created
  workflow_dispatch: {}
  pull_request_target:
    types: [opened, synchronize, reopened]
    branches: "*"
    paths-ignore:
      - 'docs/**'
      - '*.md'
      - '*.yml'
      - '.github'

permissions:
  statuses: write

jobs:
  build:
    name: test
    runs-on: ubuntu-latest
    if: ${{ (github.event.issue.pull_request && (github.event.comment.body == '/test all' || github.event.comment.body == '/test {name}')) || github.event_name == 'push' || github.event_name == 'pull_request_target' }}
    env:
      GO111MODULE: on
    steps:
      - name: Get PR details
        if: ${{ github.event_name == 'issue_comment'}}
        uses: xt0rted/pull-request-comment-branch@v1
        id: comment-branch

      - name: Set commit status as pending
        if: ${{ github.event_name == 'issue_comment'}}
        uses: myrotvorets/set-commit-status-action@master
        with:
          sha: ${{ steps.comment-branch.outputs.head_sha }}
          token: ${{ secrets.GITHUB_TOKEN }}
          status: pending

      - name: Github API Request
        id: request
        uses: octokit/request-action@v2.1.7
        with:
          route: ${{ github.event.issue.pull_request.url }}
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      - name: Get PR informations
        id: pr_data
        run: |
          echo "repo_name=${{ fromJson(steps.request.outputs.data).head.repo.full_name }}" >> $GITHUB_STATE
          echo "repo_clone_url=${{ fromJson(steps.request.outputs.data).head.repo.clone_url }}" >> $GITHUB_STATE
          echo "repo_ssh_url=${{ fromJson(steps.request.outputs.data).head.repo.ssh_url }}" >> $GITHUB_STATE
      - name: Check out code into the Go module directory
        uses: actions/checkout@v3
        with:
          token: ${{ secrets.GITHUB_TOKEN }}
          repository: ${{fromJson(steps.request.outputs.data).head.repo.full_name}}
          ref: ${{fromJson(steps.request.outputs.data).head.ref}}
          path: src/github.com/sealerio/sealer
      - name: Install deps
        run: |
          sudo su
          sudo apt-get update
          sudo apt-get install -y libgpgme-dev libbtrfs-dev libdevmapper-dev
          sudo mkdir /var/lib/sealer
      - name: Set up Go 1.17
        uses: actions/setup-go@v3
        with:
          go-version: 1.17
        id: go

      - name: Install sealer and ginkgo
        shell: bash
        run: |
          docker run --rm -v ${PWD}:/usr/src/sealer -w /usr/src/sealer registry.cn-qingdao.aliyuncs.com/sealer-io/sealer-build:v1 make linux
          export SEALER_DIR=${PWD}/_output/bin/sealer/linux_amd64
          echo "$SEALER_DIR" >> $GITHUB_PATH
          go install github.com/onsi/ginkgo/ginkgo@v1.16.2
          go install github.com/onsi/gomega/...@v1.12.0
          GOPATH=`go env GOPATH`
          echo "$GOPATH/bin" >> $GITHUB_PATH
        working-directory: src/github.com/sealerio/sealer

      - name: Run **{Test name}** test and generate coverage
        shell: bash
        working-directory: src/github.com/sealerio/sealer
        env:
          REGISTRY_USERNAME: ${{ secrets.REGISTRY_USERNAME }}
          REGISTRY_PASSWORD: ${{ secrets.REGISTRY_PASSWORD }}
          REGISTRY_URL: ${{ secrets.REGISTRY_URL }}
          IMAGE_NAME: ${{ secrets.IMAGE_NAME}}
          ACCESSKEYID: ${{ secrets.ACCESSKEYID }}
          ACCESSKEYSECRET: ${{ secrets.ACCESSKEYSECRET }}
          RegionID: ${{ secrets.RegionID }}
        run: | # Your main focus is here
          ginkgo -v -focus="{your test}" -cover -covermode=atomic -coverpkg=./... -coverprofile=/tmp/coverage.out -trace test

      - name: Upload coverage to Codecov
        uses: codecov/codecov-action@v3
        with:
          token: ${{ secrets.CODECOV_TOKEN }}
          files: /tmp/coverage.out
          flags: e2e-tests
          name: codecov-umbrella

      - name: Set final commit status
        uses: myrotvorets/set-commit-status-action@master
        if: contains(github.event.comment.body, '/test') && always()
        with:
          sha: ${{ steps.comment-branch.outputs.head_sha }}
          token: ${{ secrets.GITHUB_TOKEN }}
          status: ${{ job.status }}
```

More information about github action can be found in the documentation:[Github action](https://docs.github.com/en/actions/quickstart).

## Engage to help anything

We choose GitHub as the primary place for Sealer to collaborate. So the latest updates of Sealer are always here. Although contributions via PR is an explicit way to help, we still call for any other ways.

* reply to other's issues if you could;
* help solve other user's problems;
* help review other's PR design;
* help review other's codes in PR;
* discuss Sealer to make things clearer;
* advocate Sealer technology beyond GitHub;
* write blogs on Sealer and so on.

In a word, **ANY HELP IS CONTRIBUTION.**