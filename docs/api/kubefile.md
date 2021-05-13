# kubefile

```shell script
FROM kubernetes:1.18.0 # Base Image
COPY wordpress-chart . # copy files to CloudImage
COPY helm /usr/bin 
ENV DOMAIN aliyun.com # cluster global config
RUN wget https://get.helm.sh/helm-v3.5.2-linux-amd64.tar.gz # run command on building a CloudImage
CMD ./helm install wordpress wordpress-chart  # run command on creating a cluster
CMD ./helm list # multi CMD is valid
```