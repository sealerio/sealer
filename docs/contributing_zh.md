# 贡献指南

## 仓库设置

sealer与其它的开源项目一样，任何开发者都可以通过fork & pull request的方式进行代码贡献。

1. FORK 点击项目右上角的fork按钮，把alibaba/sealer fork到自己的仓库中如 fanux/sealer
2. CLONE 把fork后的项目clone到自己本地，如`git clone https://github.com/fanux/sealer`
3. Set Remote upstream, 方便把alibaba/sealer的代码更新到自己的仓库中
```shell script
git remote add upstream https://github.com/alibaba/sealer.git
git remote set-url --push upstream no-pushing
```
更新主分支代码到自己的仓库可以：
```shell script
git pull upstream main # git pull <remote name> <branch name>
git push
```

建议main分支只做代码同步，所有功能切一个新分支开发，如修复一个bug:
```shell script
git checkout -b bugfix/calico-interface
# 开发完成后
git push --set-upstream origin bugfix/calico-interface
```

代码先提交到自己的仓库中，然后在自己的仓库里点击pull request申请合并代码。
然后在MR中就可以看到一个黄色的 "signed the CLA" 按钮，点一下签署CLA直至按钮变绿.

## 代码开发

代码的commit信息请尽量描述清楚, 一个PR功能也尽可能单一一些方便review

### 需求开发

可以到issue中去寻找已经贴了[kind/featue](https://github.com/alibaba/sealer/issues?q=is%3Aissue+is%3Aopen+label%3Akind%2Ffeature)标签的任务，注意有的需求
没有放到里程碑里面说明正在讨论还未决定是否开发，建议认领已经放到里程碑内的需求。

如果你有一些新的需求，建议先开issue讨论，再进行编码开发。

### bug修复以及优化

任何优化的点都可以PR，如文档不全，发现bug，排版问题，多余空格，错别字，健壮性处理，冗长代码重复代码，命名不规范, 丑陋代码等等