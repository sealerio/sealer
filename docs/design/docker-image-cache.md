# docker image cache

## docker daemon config
/etc/docker/daemon.json

we bring some changes on dockerd, there is a new filed in daemon.json—"mirror-registries".

Over the standard docker version. `docker pull a.test.com/test/test:v1` will go to a.test.com directly, even though the
"registry-mirrors" was configured.

With "mirror-registries", we can make the `docker pull a.test.com/test/test:v1` to some mirror endpoints. There are some
examples following:

Step 1:

`docker pull reg.test1.com/library/nginx:latest` from `mirror.test1.com`, `/mirror.test2.com` first.

```json
{
"mirror-registries":[
{
	"domain": "reg.test1.com",
	"mirrors": ["http://mirror.test1.com", "https://mirror.test2.com"]
}
]
}
```

Step 2:

docker pull anything from `http://sea.hub:5000`, `https://mirror.test2.com` first

```json
{
  "mirror-registries":[
    {
      "domain": "*",
      "mirrors": ["http://sea.hub:5000", "https://mirror.test2.com"]
    }
  ],
  "insecure-registries": ["sea.hub:5000", "mirror.test1.com"]
}
```

### registry config

Config with registry auth info

```yaml
version: 0.1
log:
  fields:
    service: registry
storage:
  cache:
    blobdescriptor: inmemory
  filesystem:
    rootdirectory: /var/lib/registry
http:
  addr: :5000
  headers:
    X-Content-Type-Options: [nosniff]
proxy:
  remoteregistries:
  # will cache image from docker pull docker.io/library/nginx:latest or docker pull nginx
  - url: https://registry-1.docker.io #dockerhub default registry
    username:
    password:
    # will cache image from docker pull reg.test1.com/library/nginx:latest
  - url: https://reg.test1.com
    username: username
    password: password
  - url: http://reg.test2.com
    username: username
    password: password
health:
  storagedriver:
    enabled: true
    interval: 10s
    threshold: 3
```

Or config with nothing remote registry info, we can get this info dynamically.

```yaml
version: 0.1
log:
  fields:
    service: registry
storage:
  cache:
    blobdescriptor: inmemory
  filesystem:
    rootdirectory: /var/lib/registry
http:
  addr: :5000
  headers:
    X-Content-Type-Options: [nosniff]
proxy:
  #turn on the proxy ability, but with noting registry auth info.
  on: true
health:
  storagedriver:
    enabled: true
    interval: 10s
    threshold: 3
```

registry config should be mounted as /etc/docker/registry/config.yml, and mount host /var/lib/registry using -v /var/lib/registry/:/var/lib/registry/ to store image cache

### Describe what feature you want

### Additional context
remote registry could be added dynamically, but I do not store the dynamical remote registry info, because there would be many pair of username and password for same url probably, and maybe some image from different namespace has different auth info. Thus, it's costly for adding remote registries dynamically, every docker pull request will generate request to real registry from local registry to get real auth endpoint.
And for making cache registry work, there must be one remote registry item, so I take the following config as default registry config.yml.

```yaml
version: 0.1
log:
  fields:
    service: registry
storage:
  cache:
    blobdescriptor: inmemory
  filesystem:
    rootdirectory: /var/lib/registry
http:
  addr: :5000
  headers:
    X-Content-Type-Options: [nosniff]
proxy:
  remoteregistries:
  - url: https://registry-1.docker.io
    username:
    password:
health:
  storagedriver:
    enabled: true
    interval: 10s
    threshold: 3
```

at the runtime, I guess not everyone needs the cache ability, So I recommend turn the cache off, leave the choice to users.
the following config will turn off cache ability, and the registry will behave like the community version.

```yaml
version: 0.1
log:
  fields:
    service: registry
storage:
  cache:
    blobdescriptor: inmemory
  filesystem:
    rootdirectory: /var/lib/registry
http:
  addr: :5000
  headers:
    X-Content-Type-Options: [nosniff]
health:
  storagedriver:
    enabled: true
    interval: 10s
    threshold: 3
```

docker run -v  {pathToTheConfigAbove}:/etc/docker/registry/config.yml

if you do not want to provide any remote url, depend on request to config auth info dynamically. should config registry by following way:

```yaml
version: 0.1
log:
  fields:
    service: registry
storage:
  cache:
    blobdescriptor: inmemory
  filesystem:
    rootdirectory: /var/lib/registry
proxy:
  on: true
http:
  addr: :5000
  headers:
    X-Content-Type-Options: [nosniff]
health:
  storagedriver:
    enabled: true
    interval: 10s
    threshold: 3
```