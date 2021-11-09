# sealer-nvidia-plugin-pack

## preparation

1. centos 7.9 with nvidia driver installed
2. sealer latest version installed

## build

sealer build -f Kubefile -t registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes-nvidia:v1.19.8 -m lite .

## apply

1. Modify Clusterfile according to the environment
2. sealer apply -f ./Clusterfile

## results

`nvidia.com/gpu` shows on 'Allocated resources' with command 'kubectl describe node'.