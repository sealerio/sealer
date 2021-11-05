# Overview

This image will deploy ingress-nginx as DaemonSet and create an ingress controller class named `k8s.io/ingress-nginx` by
default.

Components included in this image:

* 1 Job for ingress nginx admission patch
* 1 Job for ingress nginx admission create
* 1 DaemonSet for ingress nginx controller
* 1 Service for ingress nginx controller with LoadBalancer

## How to use it

Create test web service named myapp which exposed 80 port:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: myapp
  namespace: default
spec:
  selector:
    app: myapp
    release: canary
  ports:
    - name: http
      targetPort: 80
      port: 80
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp-deploy
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: myapp
      release: canary
  template:
    metadata:
      labels:
        app: myapp
        release: canary
    spec:
      containers:
        - name: myapp
          image: ikubernetes/myapp:v2
          ports:
            - name: http
              containerPort: 80
---
```

Create ingress for myapp web service:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  namespace: default
  name: ingress-myapp
spec:
  rules:
    - host: myapp.foo.org
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: myapp
                port:
                  number: 80
  ingressClassName: nginx

```

Add node ip and domain name map to hosts file:

'172.19.0.7' is the node ip of myapp deployment pod.
'myapp.foo.org' is the ingress domain name.

```shell
cat << EOF >>/etc/hosts
172.19.0.71 myapp.foo.org
EOF
```

Access myapp service via domain name:

```shell
curl myapp.foo.org
```

## How to rebuild it use helm

Kubefile:

```shell
FROM registry.cn-qingdao.aliyuncs.com/sealer-apps/helm:v3.6.0
# add helm repo and run helm install
RUN helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
CMD helm install ingress-nginx --create-namespace --namespace ingress-system ingress-nginx/ingress-nginx
```

run below command to build it

```shell
sealer build -t {Your Image Name} -f Kubefile -m cloud .
```

More parameters see [official document here](https://kubernetes.github.io/ingress-nginx/deploy).