[简体中文](./docs/README_zh.md)

[![Go](https://github.com/alibaba/sealer/actions/workflows/go.yml/badge.svg)](https://github.com/alibaba/sealer/actions/workflows/go.yml)
[![Release](https://github.com/alibaba/sealer/actions/workflows/release.yml/badge.svg)](https://github.com/alibaba/sealer/actions/workflows/release.yml)

# What is sealer

**Build distributed application, share to anyone and run anywhere!!!**

![image](https://user-images.githubusercontent.com/8912557/117263291-b88b8700-ae84-11eb-8b46-838292e85c5c.png)

sealer[ˈsiːlər] provides the way for distributed application package and delivery based on kubernetes. 

It solves the delivery problem of complex applications by packaging distributed applications and dependencies(like database,middleware) together.

> Concept

* CloudImage : like Dockerimage, but the rootfs is kubernetes, and contains all the dependencies(docker images,yaml files or helm chart...) your application needs.
* Kubefile : the file describe how to build a CloudImage.
* Clusterfile : the config of using CloudImage to run a cluster.

![image](https://user-images.githubusercontent.com/8912557/117400612-97cf3a00-af35-11eb-90b9-f5dc8e8117b5.png)


We can write a Kubefile, and build a CloudImage, then using a Clusterfile to run a cluster.

sealer[ˈsiːlər] provides the way for distributed application package and delivery based on kubernetes. 

It solves the delivery problem of complex applications by packaging distributed applications and dependencies(like database,middleware) together.

For example, build a dashboard CloudImage:

Kubefile:

```shell script
# base CloudImage contains all the files that run a kubernetes cluster needed.
#    1. kubernetes components like kubectl kubeadm kubelet and apiserver images ...
#    2. docker engine, and a private registry
#    3. config files, yaml, static files, scripts ...
FROM registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.9
# generate kubernetes dashboard yaml file
RUN echo "IyBDb3B5cmlnaHQgMjAxNyBUaGUgS3ViZXJuZXRlcyBBdXRob3JzLgojCiMgTGljZW5zZWQgdW5kZXIgdGhlIEFwYWNoZSBMaWNlbnNlLCBWZXJzaW9uIDIuMCAodGhlICJMaWNlbnNlIik7CiMgeW91IG1heSBub3QgdXNlIHRoaXMgZmlsZSBleGNlcHQgaW4gY29tcGxpYW5jZSB3aXRoIHRoZSBMaWNlbnNlLgojIFlvdSBtYXkgb2J0YWluIGEgY29weSBvZiB0aGUgTGljZW5zZSBhdAojCiMgICAgIGh0dHA6Ly93d3cuYXBhY2hlLm9yZy9saWNlbnNlcy9MSUNFTlNFLTIuMAojCiMgVW5sZXNzIHJlcXVpcmVkIGJ5IGFwcGxpY2FibGUgbGF3IG9yIGFncmVlZCB0byBpbiB3cml0aW5nLCBzb2Z0d2FyZQojIGRpc3RyaWJ1dGVkIHVuZGVyIHRoZSBMaWNlbnNlIGlzIGRpc3RyaWJ1dGVkIG9uIGFuICJBUyBJUyIgQkFTSVMsCiMgV0lUSE9VVCBXQVJSQU5USUVTIE9SIENPTkRJVElPTlMgT0YgQU5ZIEtJTkQsIGVpdGhlciBleHByZXNzIG9yIGltcGxpZWQuCiMgU2VlIHRoZSBMaWNlbnNlIGZvciB0aGUgc3BlY2lmaWMgbGFuZ3VhZ2UgZ292ZXJuaW5nIHBlcm1pc3Npb25zIGFuZAojIGxpbWl0YXRpb25zIHVuZGVyIHRoZSBMaWNlbnNlLgoKYXBpVmVyc2lvbjogdjEKa2luZDogTmFtZXNwYWNlCm1ldGFkYXRhOgogIG5hbWU6IGt1YmVybmV0ZXMtZGFzaGJvYXJkCgotLS0KCmFwaVZlcnNpb246IHYxCmtpbmQ6IFNlcnZpY2VBY2NvdW50Cm1ldGFkYXRhOgogIGxhYmVsczoKICAgIGs4cy1hcHA6IGt1YmVybmV0ZXMtZGFzaGJvYXJkCiAgbmFtZToga3ViZXJuZXRlcy1kYXNoYm9hcmQKICBuYW1lc3BhY2U6IGt1YmVybmV0ZXMtZGFzaGJvYXJkCgotLS0KCmtpbmQ6IFNlcnZpY2UKYXBpVmVyc2lvbjogdjEKbWV0YWRhdGE6CiAgbGFiZWxzOgogICAgazhzLWFwcDoga3ViZXJuZXRlcy1kYXNoYm9hcmQKICBuYW1lOiBrdWJlcm5ldGVzLWRhc2hib2FyZAogIG5hbWVzcGFjZToga3ViZXJuZXRlcy1kYXNoYm9hcmQKc3BlYzoKICBwb3J0czoKICAgIC0gcG9ydDogNDQzCiAgICAgIHRhcmdldFBvcnQ6IDg0NDMKICBzZWxlY3RvcjoKICAgIGs4cy1hcHA6IGt1YmVybmV0ZXMtZGFzaGJvYXJkCgotLS0KCmFwaVZlcnNpb246IHYxCmtpbmQ6IFNlY3JldAptZXRhZGF0YToKICBsYWJlbHM6CiAgICBrOHMtYXBwOiBrdWJlcm5ldGVzLWRhc2hib2FyZAogIG5hbWU6IGt1YmVybmV0ZXMtZGFzaGJvYXJkLWNlcnRzCiAgbmFtZXNwYWNlOiBrdWJlcm5ldGVzLWRhc2hib2FyZAp0eXBlOiBPcGFxdWUKCi0tLQoKYXBpVmVyc2lvbjogdjEKa2luZDogU2VjcmV0Cm1ldGFkYXRhOgogIGxhYmVsczoKICAgIGs4cy1hcHA6IGt1YmVybmV0ZXMtZGFzaGJvYXJkCiAgbmFtZToga3ViZXJuZXRlcy1kYXNoYm9hcmQtY3NyZgogIG5hbWVzcGFjZToga3ViZXJuZXRlcy1kYXNoYm9hcmQKdHlwZTogT3BhcXVlCmRhdGE6CiAgY3NyZjogIiIKCi0tLQoKYXBpVmVyc2lvbjogdjEKa2luZDogU2VjcmV0Cm1ldGFkYXRhOgogIGxhYmVsczoKICAgIGs4cy1hcHA6IGt1YmVybmV0ZXMtZGFzaGJvYXJkCiAgbmFtZToga3ViZXJuZXRlcy1kYXNoYm9hcmQta2V5LWhvbGRlcgogIG5hbWVzcGFjZToga3ViZXJuZXRlcy1kYXNoYm9hcmQKdHlwZTogT3BhcXVlCgotLS0KCmtpbmQ6IENvbmZpZ01hcAphcGlWZXJzaW9uOiB2MQptZXRhZGF0YToKICBsYWJlbHM6CiAgICBrOHMtYXBwOiBrdWJlcm5ldGVzLWRhc2hib2FyZAogIG5hbWU6IGt1YmVybmV0ZXMtZGFzaGJvYXJkLXNldHRpbmdzCiAgbmFtZXNwYWNlOiBrdWJlcm5ldGVzLWRhc2hib2FyZAoKLS0tCgpraW5kOiBSb2xlCmFwaVZlcnNpb246IHJiYWMuYXV0aG9yaXphdGlvbi5rOHMuaW8vdjEKbWV0YWRhdGE6CiAgbGFiZWxzOgogICAgazhzLWFwcDoga3ViZXJuZXRlcy1kYXNoYm9hcmQKICBuYW1lOiBrdWJlcm5ldGVzLWRhc2hib2FyZAogIG5hbWVzcGFjZToga3ViZXJuZXRlcy1kYXNoYm9hcmQKcnVsZXM6CiAgIyBBbGxvdyBEYXNoYm9hcmQgdG8gZ2V0LCB1cGRhdGUgYW5kIGRlbGV0ZSBEYXNoYm9hcmQgZXhjbHVzaXZlIHNlY3JldHMuCiAgLSBhcGlHcm91cHM6IFsiIl0KICAgIHJlc291cmNlczogWyJzZWNyZXRzIl0KICAgIHJlc291cmNlTmFtZXM6IFsia3ViZXJuZXRlcy1kYXNoYm9hcmQta2V5LWhvbGRlciIsICJrdWJlcm5ldGVzLWRhc2hib2FyZC1jZXJ0cyIsICJrdWJlcm5ldGVzLWRhc2hib2FyZC1jc3JmIl0KICAgIHZlcmJzOiBbImdldCIsICJ1cGRhdGUiLCAiZGVsZXRlIl0KICAgICMgQWxsb3cgRGFzaGJvYXJkIHRvIGdldCBhbmQgdXBkYXRlICdrdWJlcm5ldGVzLWRhc2hib2FyZC1zZXR0aW5ncycgY29uZmlnIG1hcC4KICAtIGFwaUdyb3VwczogWyIiXQogICAgcmVzb3VyY2VzOiBbImNvbmZpZ21hcHMiXQogICAgcmVzb3VyY2VOYW1lczogWyJrdWJlcm5ldGVzLWRhc2hib2FyZC1zZXR0aW5ncyJdCiAgICB2ZXJiczogWyJnZXQiLCAidXBkYXRlIl0KICAgICMgQWxsb3cgRGFzaGJvYXJkIHRvIGdldCBtZXRyaWNzLgogIC0gYXBpR3JvdXBzOiBbIiJdCiAgICByZXNvdXJjZXM6IFsic2VydmljZXMiXQogICAgcmVzb3VyY2VOYW1lczogWyJoZWFwc3RlciIsICJkYXNoYm9hcmQtbWV0cmljcy1zY3JhcGVyIl0KICAgIHZlcmJzOiBbInByb3h5Il0KICAtIGFwaUdyb3VwczogWyIiXQogICAgcmVzb3VyY2VzOiBbInNlcnZpY2VzL3Byb3h5Il0KICAgIHJlc291cmNlTmFtZXM6IFsiaGVhcHN0ZXIiLCAiaHR0cDpoZWFwc3RlcjoiLCAiaHR0cHM6aGVhcHN0ZXI6IiwgImRhc2hib2FyZC1tZXRyaWNzLXNjcmFwZXIiLCAiaHR0cDpkYXNoYm9hcmQtbWV0cmljcy1zY3JhcGVyIl0KICAgIHZlcmJzOiBbImdldCJdCgotLS0KCmtpbmQ6IENsdXN0ZXJSb2xlCmFwaVZlcnNpb246IHJiYWMuYXV0aG9yaXphdGlvbi5rOHMuaW8vdjEKbWV0YWRhdGE6CiAgbGFiZWxzOgogICAgazhzLWFwcDoga3ViZXJuZXRlcy1kYXNoYm9hcmQKICBuYW1lOiBrdWJlcm5ldGVzLWRhc2hib2FyZApydWxlczoKICAjIEFsbG93IE1ldHJpY3MgU2NyYXBlciB0byBnZXQgbWV0cmljcyBmcm9tIHRoZSBNZXRyaWNzIHNlcnZlcgogIC0gYXBpR3JvdXBzOiBbIm1ldHJpY3MuazhzLmlvIl0KICAgIHJlc291cmNlczogWyJwb2RzIiwgIm5vZGVzIl0KICAgIHZlcmJzOiBbImdldCIsICJsaXN0IiwgIndhdGNoIl0KCi0tLQoKYXBpVmVyc2lvbjogcmJhYy5hdXRob3JpemF0aW9uLms4cy5pby92MQpraW5kOiBSb2xlQmluZGluZwptZXRhZGF0YToKICBsYWJlbHM6CiAgICBrOHMtYXBwOiBrdWJlcm5ldGVzLWRhc2hib2FyZAogIG5hbWU6IGt1YmVybmV0ZXMtZGFzaGJvYXJkCiAgbmFtZXNwYWNlOiBrdWJlcm5ldGVzLWRhc2hib2FyZApyb2xlUmVmOgogIGFwaUdyb3VwOiByYmFjLmF1dGhvcml6YXRpb24uazhzLmlvCiAga2luZDogUm9sZQogIG5hbWU6IGt1YmVybmV0ZXMtZGFzaGJvYXJkCnN1YmplY3RzOgogIC0ga2luZDogU2VydmljZUFjY291bnQKICAgIG5hbWU6IGt1YmVybmV0ZXMtZGFzaGJvYXJkCiAgICBuYW1lc3BhY2U6IGt1YmVybmV0ZXMtZGFzaGJvYXJkCgotLS0KCmFwaVZlcnNpb246IHJiYWMuYXV0aG9yaXphdGlvbi5rOHMuaW8vdjEKa2luZDogQ2x1c3RlclJvbGVCaW5kaW5nCm1ldGFkYXRhOgogIG5hbWU6IGt1YmVybmV0ZXMtZGFzaGJvYXJkCnJvbGVSZWY6CiAgYXBpR3JvdXA6IHJiYWMuYXV0aG9yaXphdGlvbi5rOHMuaW8KICBraW5kOiBDbHVzdGVyUm9sZQogIG5hbWU6IGt1YmVybmV0ZXMtZGFzaGJvYXJkCnN1YmplY3RzOgogIC0ga2luZDogU2VydmljZUFjY291bnQKICAgIG5hbWU6IGt1YmVybmV0ZXMtZGFzaGJvYXJkCiAgICBuYW1lc3BhY2U6IGt1YmVybmV0ZXMtZGFzaGJvYXJkCgotLS0KCmtpbmQ6IERlcGxveW1lbnQKYXBpVmVyc2lvbjogYXBwcy92MQptZXRhZGF0YToKICBsYWJlbHM6CiAgICBrOHMtYXBwOiBrdWJlcm5ldGVzLWRhc2hib2FyZAogIG5hbWU6IGt1YmVybmV0ZXMtZGFzaGJvYXJkCiAgbmFtZXNwYWNlOiBrdWJlcm5ldGVzLWRhc2hib2FyZApzcGVjOgogIHJlcGxpY2FzOiAxCiAgcmV2aXNpb25IaXN0b3J5TGltaXQ6IDEwCiAgc2VsZWN0b3I6CiAgICBtYXRjaExhYmVsczoKICAgICAgazhzLWFwcDoga3ViZXJuZXRlcy1kYXNoYm9hcmQKICB0ZW1wbGF0ZToKICAgIG1ldGFkYXRhOgogICAgICBsYWJlbHM6CiAgICAgICAgazhzLWFwcDoga3ViZXJuZXRlcy1kYXNoYm9hcmQKICAgIHNwZWM6CiAgICAgIGNvbnRhaW5lcnM6CiAgICAgICAgLSBuYW1lOiBrdWJlcm5ldGVzLWRhc2hib2FyZAogICAgICAgICAgaW1hZ2U6IGt1YmVybmV0ZXN1aS9kYXNoYm9hcmQ6djIuMi4wCiAgICAgICAgICBpbWFnZVB1bGxQb2xpY3k6IEFsd2F5cwogICAgICAgICAgcG9ydHM6CiAgICAgICAgICAgIC0gY29udGFpbmVyUG9ydDogODQ0MwogICAgICAgICAgICAgIHByb3RvY29sOiBUQ1AKICAgICAgICAgIGFyZ3M6CiAgICAgICAgICAgIC0gLS1hdXRvLWdlbmVyYXRlLWNlcnRpZmljYXRlcwogICAgICAgICAgICAtIC0tbmFtZXNwYWNlPWt1YmVybmV0ZXMtZGFzaGJvYXJkCiAgICAgICAgICAgICMgVW5jb21tZW50IHRoZSBmb2xsb3dpbmcgbGluZSB0byBtYW51YWxseSBzcGVjaWZ5IEt1YmVybmV0ZXMgQVBJIHNlcnZlciBIb3N0CiAgICAgICAgICAgICMgSWYgbm90IHNwZWNpZmllZCwgRGFzaGJvYXJkIHdpbGwgYXR0ZW1wdCB0byBhdXRvIGRpc2NvdmVyIHRoZSBBUEkgc2VydmVyIGFuZCBjb25uZWN0CiAgICAgICAgICAgICMgdG8gaXQuIFVuY29tbWVudCBvbmx5IGlmIHRoZSBkZWZhdWx0IGRvZXMgbm90IHdvcmsuCiAgICAgICAgICAgICMgLSAtLWFwaXNlcnZlci1ob3N0PWh0dHA6Ly9teS1hZGRyZXNzOnBvcnQKICAgICAgICAgIHZvbHVtZU1vdW50czoKICAgICAgICAgICAgLSBuYW1lOiBrdWJlcm5ldGVzLWRhc2hib2FyZC1jZXJ0cwogICAgICAgICAgICAgIG1vdW50UGF0aDogL2NlcnRzCiAgICAgICAgICAgICAgIyBDcmVhdGUgb24tZGlzayB2b2x1bWUgdG8gc3RvcmUgZXhlYyBsb2dzCiAgICAgICAgICAgIC0gbW91bnRQYXRoOiAvdG1wCiAgICAgICAgICAgICAgbmFtZTogdG1wLXZvbHVtZQogICAgICAgICAgbGl2ZW5lc3NQcm9iZToKICAgICAgICAgICAgaHR0cEdldDoKICAgICAgICAgICAgICBzY2hlbWU6IEhUVFBTCiAgICAgICAgICAgICAgcGF0aDogLwogICAgICAgICAgICAgIHBvcnQ6IDg0NDMKICAgICAgICAgICAgaW5pdGlhbERlbGF5U2Vjb25kczogMzAKICAgICAgICAgICAgdGltZW91dFNlY29uZHM6IDMwCiAgICAgICAgICBzZWN1cml0eUNvbnRleHQ6CiAgICAgICAgICAgIGFsbG93UHJpdmlsZWdlRXNjYWxhdGlvbjogZmFsc2UKICAgICAgICAgICAgcmVhZE9ubHlSb290RmlsZXN5c3RlbTogdHJ1ZQogICAgICAgICAgICBydW5Bc1VzZXI6IDEwMDEKICAgICAgICAgICAgcnVuQXNHcm91cDogMjAwMQogICAgICB2b2x1bWVzOgogICAgICAgIC0gbmFtZToga3ViZXJuZXRlcy1kYXNoYm9hcmQtY2VydHMKICAgICAgICAgIHNlY3JldDoKICAgICAgICAgICAgc2VjcmV0TmFtZToga3ViZXJuZXRlcy1kYXNoYm9hcmQtY2VydHMKICAgICAgICAtIG5hbWU6IHRtcC12b2x1bWUKICAgICAgICAgIGVtcHR5RGlyOiB7fQogICAgICBzZXJ2aWNlQWNjb3VudE5hbWU6IGt1YmVybmV0ZXMtZGFzaGJvYXJkCiAgICAgIG5vZGVTZWxlY3RvcjoKICAgICAgICAia3ViZXJuZXRlcy5pby9vcyI6IGxpbnV4CiAgICAgICMgQ29tbWVudCB0aGUgZm9sbG93aW5nIHRvbGVyYXRpb25zIGlmIERhc2hib2FyZCBtdXN0IG5vdCBiZSBkZXBsb3llZCBvbiBtYXN0ZXIKICAgICAgdG9sZXJhdGlvbnM6CiAgICAgICAgLSBrZXk6IG5vZGUtcm9sZS5rdWJlcm5ldGVzLmlvL21hc3RlcgogICAgICAgICAgZWZmZWN0OiBOb1NjaGVkdWxlCgotLS0KCmtpbmQ6IFNlcnZpY2UKYXBpVmVyc2lvbjogdjEKbWV0YWRhdGE6CiAgbGFiZWxzOgogICAgazhzLWFwcDogZGFzaGJvYXJkLW1ldHJpY3Mtc2NyYXBlcgogIG5hbWU6IGRhc2hib2FyZC1tZXRyaWNzLXNjcmFwZXIKICBuYW1lc3BhY2U6IGt1YmVybmV0ZXMtZGFzaGJvYXJkCnNwZWM6CiAgcG9ydHM6CiAgICAtIHBvcnQ6IDgwMDAKICAgICAgdGFyZ2V0UG9ydDogODAwMAogIHNlbGVjdG9yOgogICAgazhzLWFwcDogZGFzaGJvYXJkLW1ldHJpY3Mtc2NyYXBlcgoKLS0tCgpraW5kOiBEZXBsb3ltZW50CmFwaVZlcnNpb246IGFwcHMvdjEKbWV0YWRhdGE6CiAgbGFiZWxzOgogICAgazhzLWFwcDogZGFzaGJvYXJkLW1ldHJpY3Mtc2NyYXBlcgogIG5hbWU6IGRhc2hib2FyZC1tZXRyaWNzLXNjcmFwZXIKICBuYW1lc3BhY2U6IGt1YmVybmV0ZXMtZGFzaGJvYXJkCnNwZWM6CiAgcmVwbGljYXM6IDEKICByZXZpc2lvbkhpc3RvcnlMaW1pdDogMTAKICBzZWxlY3RvcjoKICAgIG1hdGNoTGFiZWxzOgogICAgICBrOHMtYXBwOiBkYXNoYm9hcmQtbWV0cmljcy1zY3JhcGVyCiAgdGVtcGxhdGU6CiAgICBtZXRhZGF0YToKICAgICAgbGFiZWxzOgogICAgICAgIGs4cy1hcHA6IGRhc2hib2FyZC1tZXRyaWNzLXNjcmFwZXIKICAgICAgYW5ub3RhdGlvbnM6CiAgICAgICAgc2VjY29tcC5zZWN1cml0eS5hbHBoYS5rdWJlcm5ldGVzLmlvL3BvZDogJ3J1bnRpbWUvZGVmYXVsdCcKICAgIHNwZWM6CiAgICAgIGNvbnRhaW5lcnM6CiAgICAgICAgLSBuYW1lOiBkYXNoYm9hcmQtbWV0cmljcy1zY3JhcGVyCiAgICAgICAgICBpbWFnZToga3ViZXJuZXRlc3VpL21ldHJpY3Mtc2NyYXBlcjp2MS4wLjYKICAgICAgICAgIHBvcnRzOgogICAgICAgICAgICAtIGNvbnRhaW5lclBvcnQ6IDgwMDAKICAgICAgICAgICAgICBwcm90b2NvbDogVENQCiAgICAgICAgICBsaXZlbmVzc1Byb2JlOgogICAgICAgICAgICBodHRwR2V0OgogICAgICAgICAgICAgIHNjaGVtZTogSFRUUAogICAgICAgICAgICAgIHBhdGg6IC8KICAgICAgICAgICAgICBwb3J0OiA4MDAwCiAgICAgICAgICAgIGluaXRpYWxEZWxheVNlY29uZHM6IDMwCiAgICAgICAgICAgIHRpbWVvdXRTZWNvbmRzOiAzMAogICAgICAgICAgdm9sdW1lTW91bnRzOgogICAgICAgICAgLSBtb3VudFBhdGg6IC90bXAKICAgICAgICAgICAgbmFtZTogdG1wLXZvbHVtZQogICAgICAgICAgc2VjdXJpdHlDb250ZXh0OgogICAgICAgICAgICBhbGxvd1ByaXZpbGVnZUVzY2FsYXRpb246IGZhbHNlCiAgICAgICAgICAgIHJlYWRPbmx5Um9vdEZpbGVzeXN0ZW06IHRydWUKICAgICAgICAgICAgcnVuQXNVc2VyOiAxMDAxCiAgICAgICAgICAgIHJ1bkFzR3JvdXA6IDIwMDEKICAgICAgc2VydmljZUFjY291bnROYW1lOiBrdWJlcm5ldGVzLWRhc2hib2FyZAogICAgICBub2RlU2VsZWN0b3I6CiAgICAgICAgImt1YmVybmV0ZXMuaW8vb3MiOiBsaW51eAogICAgICAjIENvbW1lbnQgdGhlIGZvbGxvd2luZyB0b2xlcmF0aW9ucyBpZiBEYXNoYm9hcmQgbXVzdCBub3QgYmUgZGVwbG95ZWQgb24gbWFzdGVyCiAgICAgIHRvbGVyYXRpb25zOgogICAgICAgIC0ga2V5OiBub2RlLXJvbGUua3ViZXJuZXRlcy5pby9tYXN0ZXIKICAgICAgICAgIGVmZmVjdDogTm9TY2hlZHVsZQogICAgICB2b2x1bWVzOgogICAgICAgIC0gbmFtZTogdG1wLXZvbHVtZQogICAgICAgICAgZW1wdHlEaXI6IHt9Cg==" |base64  -d  > recommended.yaml
# when run this CloudImage, will apply a dashboard manifests
CMD kubectl apply -f recommended.yaml
```

Build dashobard CloudImage:

```shell script
sealer build -t registry.cn-qingdao.aliyuncs.com/sealer-io/dashboard:latest .
```

Run a kubernetes cluster with dashboard:

```shell script
# sealer will install a kubernetes on host 192.168.0.2 then apply the dashboard manifests
sealer run registry.cn-qingdao.aliyuncs.com/sealer-io/dashboard:latest --masters 192.168.0.2 --passwd xxx
# check the pod
kubectl get pod -A|grep dashboard
```

Push the CloudImage to the registry

```shell script
# you can push the CloudImage to docker hub, Ali ACR, or Harbor
sealer push registry.cn-qingdao.aliyuncs.com/sealer-io/dashboard:latest
```

# Usage scenarios & features

- [x] An extremely simple way to install kubernetes and other software in the kubernetes ecosystem in a production or offline environment. 
- [x] Through Kubefile, you can easily customize the kubernetes CloudImage to package the cluster and applications, and submit them to the registry.  
- [x] Powerful life cycle management capabilities, to perform operations such as cluster upgrade, cluster backup and recovery, node expansion and contraction in unimaginable simple ways 
- [x] Very fast, complete cluster installation within 3 minutes 
- [x] Support ARM x86, v1.20 and above versions support containerd, almost compatible with all Linux operating systems that support systemd 
- [x] Does not rely on ansible haproxy keepalived, high availability is achieved through ipvs, takes up less resources, is stable and reliable 
- [x] There are very few in the official warehouse. Many ecological software images can be used directly, including all dependencies, one-click installation

# Quick start

Install a kubernetes cluster

```shell script
#install Sealer binaries
wget https://github.com/alibaba/sealer/releases/download/v0.1.4/sealer-0.1.4-linux-amd64.tar.gz && \
tar zxvf sealer-0.1.4-linux-amd64.tar.gz && mv sealer /usr/bin
#run a kubernetes cluster 
sealer run kubernetes:v1.19.9 --masters 192.168.0.2 --passwd xxx
```

Install a cluster on public cloud(now support alicloud):

```shell script
export ACCESSKEYID=xxx
export ACCESSKEYSECRET=xxx
sealer run registry.cn-qingdao.aliyuncs.com/sealer-io/dashboard:latest
```

Or specify the number of nodes to run the cluster

```shell script
sealer run registry.cn-qingdao.aliyuncs.com/sealer-io/dashboard:latest \
  --masters 3 --nodes 3
```

```shell script
[root@iZm5e42unzb79kod55hehvZ ~]# kubectl get node
NAME                    STATUS ROLES AGE VERSION
izm5e42unzb79kod55hehvz Ready master 18h v1.16.9
izm5ehdjw3kru84f0kq7r7z Ready master 18h v1.16.9
izm5ehdjw3kru84f0kq7r8z Ready master 18h v1.16.9
izm5ehdjw3kru84f0kq7r9z Ready <none> 18h v1.16.9
izm5ehdjw3kru84f0kq7raz Ready <none> 18h v1.16.9
izm5ehdjw3kru84f0kq7rbz Ready <none> 18h v1.16.9
```

View the default startup configuration of the CloudImage:

```shell script
sealer inspect -c registry.cn-qingdao.aliyuncs.com/sealer-io/dashboard:latest
```

Use Clusterfile to set up a k8s cluster

## Scenario 1. Install on an existing server, the provider type is BAREMETAL

Clusterfile content:

```
apiVersion: sealer.aliyun.com/v1alpha1
kind: Cluster
metadata:
  name: my-cluster
spec:
  image: registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.9
  provider: BAREMETAL
  ssh:
    # SSH login password, if you use the key to log in, you don’t need to set it
    passwd:
    ## The absolute path of the ssh private key file, for example /root/.ssh/id_rsa
    pk: xxx
    #  The password of the ssh private key file, if there is none, set it to ""
    pkPasswd: xxx
    # ssh login user
    user: root
  network:
    # in use NIC name
    interface: eth0
    # Network plug-in name
    cniName: calico
    podCIDR: 100.64.0.0/10
    svcCIDR: 10.96.0.0/22
    withoutCNI: false
  certSANS:
    - aliyun-inc.com
    - 10.0.0.2
    
  masters:
    ipList:
     - 172.20.125.234
     - 172.20.126.5
     - 172.20.126.6
  nodes:
    ipList:
     - 172.20.126.8
     - 172.20.126.9
     - 172.20.126.10
```

```shell script
[root@iZm5e42unzb79kod55hehvZ ~]# sealer apply -f Clusterfile
[root@iZm5e42unzb79kod55hehvZ ~]# kubectl get node
```

## Scenario 2. Automatically apply for Alibaba Cloud server for installation, provider: ALI_CLOUD Clusterfile:

```
apiVersion: sealer.aliyun.com/v1alpha1
kind: Cluster
metadata:
  name: my-cluster
spec:
  image: registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.9
  provider: ALI_CLOUD
  ssh:
    # SSH login password, if you use the key to log in, you don’t need to set it
    passwd:
    ## The absolute path of the ssh private key file, for example /root/.ssh/id_rsa
    pk: xxx
    #  The password of the ssh private key file, if there is none, set it to ""
    pkPasswd: xxx
    # ssh login user
    user: root
  network:
    # in use NIC name
    interface: eth0
    # Network plug-in name
    cniName: calico
    podCIDR: 100.64.0.0/10
    svcCIDR: 10.96.0.0/22
    withoutCNI: false
  certSANS:
    - aliyun-inc.com
    - 10.0.0.2
    
  masters:
    cpu: 4
    memory: 4
    count: 3
    systemDisk: 100
    dataDisks:
    - 100
  nodes:
    cpu: 4
    memory: 4
    count: 3
    systemDisk: 100
    dataDisks:
    - 100
```

## clean the cluster

Some information of the basic settings will be written to the Clusterfile and stored in /root/.sealer/[cluster-name]/Clusterfile.

```shell script
sealer delete -f /root/.sealer/my-cluster/Clusterfile
```

# Developing Sealer

* [contributing guide](./CONTRIBUTING.md)
* [贡献文档](./docs/contributing_zh.md)
