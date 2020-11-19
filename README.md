Current Status:

* CreatePod, GetPod, GetPodStatus work
* DeletePod might work
* Tested on Ubuntu with `uptimed`

# Virtual Kubelet for systemd

This is an virtual kubelet provider that interacts with systemd. The aim is to make this to work
with K3S and go from there.

Every Linux system has systemd nowadays. by utilzing k3s (just one Go binary) and this virtual
kubelet you can provision a system using the Kubernetes API. The networking is the host's network,
so it make sense to use this for more heavy weight (stateful?) applications.

It is hoped this setup allows you to use the Kubernetes API without the need to immerse yourself in
the (large) world of kubernetes (overlay networking, ingress objects, etc., etc.). However this
_does_ imply networking and discovery (i.e.) DNS is already working on the system you're
deploying this. How to get a system into a state it has, and can run, k3s and virtual-kubelet is an
open questions (ready made image, a tiny bit of config mgmt (but how to bootstrap that?)).

`vks` will start pods as plain processes, there are no cgroups (yet, maybe systemd will allow for
this easily), but generally there is no isolation. You basically use the k8s control plane to start
linux processes. There is also no address space allocated to the PODs specically, you are using the
host's networking.

"Images" are referencing (Debian) packages, these will be apt-get installed. Discoverying that an
installed package is no longer used is hard, so this will not be done.

Each scheduled unit will adhere to a nameing scheme so `vks` knows which ones are managed by it.

## Building

Use `go build` in the top level directory, this should give you a `vks` binary which is the virtual
kubelet.

## Design

Pods can contain multiple container; each container is a new unit and tracked by systemd.

When we see a CreatePod call we call out to systemd to create a unit per container in the pod. Each
unit will be named `vks.<pod-namespace>.<pod-name>.<image-name>.service`.

We store a bunch of k8s meta data inside the unit in a `[X-kubernetes]` section. Whenever we want to
know a pod state vks will query systemd and read the unit file back. This way we know the status and
have access to all the meta data.

### Limitations

By using systemd and the hosts network we have weak isolation between pods, i.e. no more than
process isolation. Starting two pods that use the same port is guaranteed to fail for one.

## Questions

* Pod storage, secret etc. Just something on disk? `/var/lib/<something>`?
* CPU and memory usage? I *think* systemd might now, but unsure how to fetch it.
* How to provision a Debian system to be able to join a k3s cluster? Something very minimal is
  needed here. _Maybe_ getting to k3s super early will help. We can then install extra things to
  configure?
* Add a private repo for debian packages. I.e. I want to install latest CoreDNS which isn't in
  Debian. I need to add a repo for this... How?
* If imagePullPolicy is set to Always, then apt-get install the binary ? If not check if the "image"
  can be found in $PATH and use it?
* namespaces?? Are they useful on a technical level for systemd? Right now they are used for naming
  only.

## Use with K3S

Download k3s from it's releases on GitHub, you just need the `k3s` binary. Use the `k3s/k3s` shell
script to start it - this assumes `k3s` sits in "~/tmp/k3s". The script starts k3s with basically
*everything* disabled.

Compile cmd/virtual-kubelet and start it with.

~~~
sudo ./vks --kubeconfig ~/.rancher/k3s/server/cred/admin.kubeconfig --enable-node-lease --disable-taint
~~~

We need root to be allowed to install packages. Now a `k3s kubcetl get nodes` should show the
virtual kubelet as a node:

~~~
NAME    STATUS   ROLES   AGE   VERSION   INTERNAL-IP   EXTERNAL-IP   OS-IMAGE            KERNEL-VERSION     CONTAINER-RUNTIME
draak   Ready    agent   6s    v1.18.4   <none>        <none>        Ubuntu 20.04.1 LT   5.4.0-53-generic   systemd 245 (245.4-4ubuntu3.3)
~~~

`draak` is my machine's name. Networking (or figuring out how to map it to the k8s API) is still on
the TODO list. You can now try to schedule a pod: `k3s/kubelet apply -f k3s/uptimed.yaml`.

## Playing With It

### Debian

1. Install *k3s* and compile the virtual kubelet.
2. Install *policyrcd-script-zg2*: `apt-get install policyrcd-script-zg2` See
   To install daemons without starting them you will need the `policyrcd-script-zg2` package.

I'm using `uptimed` as a very simple daemon that you (probably) haven't got installed, so we can
check the entire flow. My testing happens on Debian/Ubuntu.

3. `./k3s/kubectl apply -f uptimed.yaml`

The above *should* yield:

~~~
NAME      READY   STATUS    RESTARTS   AGE
uptimed   1/1     Running   0          7m42s
~~~

You can then delete the pod.
