
NOTE: Work In Progress

# Virtual Kubelet wth systemd

This is an virtual kubelet provider that interacts with systemd. The aim is to make this to work
with K3S.

## Questions

* Pods can contain multiple containers. In systemd each container is a Unit (file). How can we keep
  track of these diff. Units and re-create the Pod when we need to?
* Pod storage, secret etc. Just something on disk? `/var/lib/<something>`?
