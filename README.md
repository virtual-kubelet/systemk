
NOTE: Work In Progress

# Virtual Kubelet for systemd

This is an virtual kubelet provider that interacts with systemd. The aim is to make this to work
with K3S and go from there.

## Questions

* Pods can contain multiple containers. In systemd each container is a Unit (file). How can we keep
  track of these diff. Units and re-create the Pod when we need to?
* Pod storage, secret etc. Just something on disk? `/var/lib/<something>`?

## Use with K3S

Download k3s from it's releases on GitHub, you just need the `k3s` binary. Use the `k3s/k3s` shell
script to start it - this assumes `k3s` sits in "~/tmp/k3s". The script basically starts k3s with
basically _everything_ disabled.

Compile cmd/virtual-kubelet and start it with:

~~~
./cmd/virtual-kubelet/virtual-kubelet --kubeconfig ~/.rancher/k3s/server/cred/admin.kubeconfig \
--enable-node-lease
~~~

(the node-lease sounded cool, don't know yet if it's really needed).

Now a `k3s kubcetl get nodes` should show the virtual kubelet as a node.

THIS IS AS FAR AS I AM RIGHT NOW.
