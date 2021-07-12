# Build a kubernetes-withcalico CloudImage

```shell script
sealer build -t kubernetes-withcalico:v1.19.9 .
sealer push kubernetes-withcalico:v1.19.9
```

# Using kubernetes-withcalico CloudImage as Base Image

```shell script
FROM kubernetes-withcalico:v1.19.9
RUN wget https://raw.githubusercontent.com/kubernetes/dashboard/v2.2.0/aio/deploy/recommended.yaml
CMD kubectl apply -f recommended.yaml
```