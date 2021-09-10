# Build a helm CloudImage

```shell script
sealer build -t sealer-apps/helm:v3.6.0 .
sealer push sealer-apps/helm:v3.6.0
```

## Using helm CloudImage as Base Image

```shell script
FROM sealer-apps/helm:v3.6.0
RUN helm repo add openebs https://openebs.github.io/charts \
    && helm repo update \
    && helm install --namespace openebs --name openebs openebs/openebs
```