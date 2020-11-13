
NOTE: Work In Progress

# Virtual Kubelet for systemd

This is an virtual kubelet provider that interacts with systemd. The aim is to make this to work
with K3S and go from there.

## Questions

* Pods can contain multiple containers. In systemd each container is a Unit (file). How can we keep
  track of these diff. Units and re-create the Pod when we need to?
* Pod storage, secret etc. Just something on disk? `/var/lib/<something>`?

## TODO

* make it work enough so we can try being a kubelet for K3S
* figure out how to connect to K3S control plane, use code from k3 agent which does this w/ a
  NODE_TOKEN
