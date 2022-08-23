# implementation of sealer build

## Abstract

Generally, the image generated from sealer build has no differences with container images. The image is compatible with OCI. Let's call it cluster image.
Sealer has some special operations based on the usual build of container images. Like (1) Adding a layer for storing containers
images automatically; (2) Saving cluster image information to the annotations from manifest of OCI v1 images. Sealer doesn't implement
the concrete building procedure. Sealer implements build over mature tools (we choose `buildah` currently). We will have an introduction for how the
sealer build implements next.

## Implement

### Engine

### Store Container Images