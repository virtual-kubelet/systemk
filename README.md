# Virtual Kubelet for systemd

This is an virtual kubelet provider that interacts with systemd. The aim is to make this to work
with [K3s](https://github.com/rancher/k3s/) and go from there.

Every Linux system has systemd nowadays. By utilizing K3s (just one Go binary) and this virtual
kubelet you can provision a system using the Kubernetes API. The networking is the host's network,
so it make sense to use this for more heavy weight (stateful?) applications.

It is hoped this setup allows you to use the Kubernetes API without the need to immerse yourself in
the (large) world of kubernetes (overlay networking, ingress objects, etc., etc.). However this
_does_ imply networking and discovery (i.e.) DNS is already working on the system you're
deploying this. How to get a system into a state it has, and can run, k3s and virtual-kubelet is an
open questions (ready made image, a tiny bit of config mgmt (but how to bootstrap that?)).

`vks` will start pods as plain processes, there are no cgroups (yet, maybe systemd will allow for
this easily), but generally there is no isolation. You basically use the k8s control plane to start
linux processes. There is also no address space allocated to the PODs specially, you are using the
host's networking.

"Images" are referencing (Debian) packages, these will be apt-get installed. Discovering that an
installed package is no longer used is hard, so this will not be done. `vks` will reuse the unit
file that comes from this install. However some other data is inject into it to make it work fully
for vks.

Each scheduled unit will adhere to a naming scheme so `vks` knows which ones are managed by it.

## Is This Useful?

Unclear. I personally consider this (once it fully works) useful enough to manage a small farm of
machines. Say you administer 200 machines and you need saner management and introspection than
config management can give you? I.e. with `kubectl` you can find which machines didn't run a DNS
server, with *deployments* you can more safely push out upgrades, "insert favorite Kubernetes
feature here".

Monitoring only requires prometheus to discover the pods via the Kubenetes API, vastly simplifying
that particular setup.

I currently manage 2 (Debian) machines and this is all manual - i.e.: login, apt-get upgrade, fiddle
with config files etc. It may turn of that k3s + vks is a better way of handling even *2* machines.

Note that getting to the stage where this all runs, is secured and everything has the correct TLS
certs (that are also rotated) is an open question.

## Current Status

Multiple containers in a pod can be run and they can see each others storage. Creating, deleing,
inspecting Pods all work. None of the higher level abstractions (replicaset, deployment) have been
tried though.

EmptyDir/ConfigMaps and Secrets are implemented, these are all backed my tmpfs, and `bind`-mounted
into the "container".

Getting logs also works, but the UI for it could be better - needs some extra setup.

## Building

Use `go build` in the top level directory, this should give you a `vks` binary which is the virtual
kubelet.

## Design

Pods can contain multiple containers; each container is a new unit and tracked by systemd. The named
image is assumed to be a *package* and will be installed via the normal means (`apt-get`, etc.). If
vks is installing the package the official system unit will be disabled; if the package already
exists we leave the existing unit alone. If the install doesn't come with a unit file (say you
install `zsh`) we will synthesize a small service unit file; for this to work the podSpec need to
(at) least define a command to run.

Now, while fiddling with this, I noticed setting up a Debian repository is an annoyingly amount of
work and the signing of packages requires GPG and manually inputting secrets. This is why a
short-cut for installing packages has been added. If the image name starts with `deb://` it is
assumed an URL and the package is fetched from there and installed. The image name is the first
string up until the `_` in the debian package name:
`deb://example.org/tmp/coredns_1.7.1-bla_amd64.deb` will download the package from that URL and
`coredns` will be the package name.

When we see a CreatePod call we call out to systemd to create a unit per container in the pod. Each
unit will be named `vks.<pod-namespace>.<pod-name>.<image-name>.service`. If a command is given it
will replace the first word of `ExecStart` and leave any options there. If `args` are also given
the entire `ExecStart` is replaced with those. If only `args` are given the command will remain and
only the options/args will be replaced.

We store a bunch of k8s meta data inside the unit in a `[X-kubernetes]` section. Whenever we want to
know a pod state vks will query systemd and read the unit file back. This way we know the status and
have access to all the meta data.

Storage is handled by systemd. Every mountpoint will get its own *tmpfs*, via bind mounts
(`BindPaths`) we mount secrets and config maps into the tmpfs. Each pod gets two tmpfss by default,
"/var" and "/run". All other paths are directly coming from the machine.

### Limitations

By using systemd and the hosts network we have weak isolation between pods, i.e. no more than
process isolation. Starting two pods that use the same port is guaranteed to fail for one.

## Questions

* Add a private repo for debian packages. I.e. I want to install latest CoreDNS which isn't in
  Debian. I need to add a repo for this... How?

## Use with K3S

Download k3s from it's releases on GitHub, you just need the `k3s` binary. Use the `k3s/k3s` shell
script to start it - this assumes `k3s` sits in "~/tmp/k3s". The script starts k3s with basically
*everything* disabled.

Compile `vks` and start it with.

~~~
sudo ./vks --kubeconfig ~/.rancher/k3s/server/cred/admin.kubeconfig --enable-node-lease --disable-taint
~~~

We need root to be allowed to install packages. Now a `k3s kubcetl get nodes` should show the
virtual kubelet as a node:

~~~
NAME    STATUS   ROLES   AGE   VERSION   INTERNAL-IP   EXTERNAL-IP   OS-IMAGE            KERNEL-VERSION     CONTAINER-RUNTIME
draak   Ready    agent   6s    v1.18.4   <none>        <none>        Ubuntu 20.04.1 LT   5.4.0-53-generic   systemd 245 (245.4-4ubuntu3.3)
~~~

`draak` is my machine's name. You can now try to schedule a pod: `k3s/kubelet apply -f
k3s/uptimed.yaml`.

Logging works, but due to TLS, is a bit fiddly, you need to start `vks` with --certfile and
--keyfile to make the HTTPS endpoint happy (enough). Once that's done you can get the logs with:

~~~
% ./kubectl logs --insecure-skip-tls-verify-backend=true uptimed
-- Logs begin at Mon 2020-08-24 09:00:18 CEST, end at Thu 2020-11-19 15:40:02 CET. --
nov 19 12:12:27 draak systemd[1]: Started uptime record daemon.
nov 19 12:14:44 draak uptimed[15245]: uptimed: no useable database found.
nov 19 12:14:44 draak systemd[1]: Stopping uptime record daemon...
nov 19 12:14:44 draak systemd[1]: vks.default.uptimed.uptimed.service: Succeeded.
nov 19 12:14:44 draak systemd[1]: Stopped uptime record daemon.
nov 19 13:38:54 draak systemd[1]: Started uptime record daemon.
nov 19 13:39:26 draak systemd[1]: Stopping uptime record daemon...
nov 19 13:39:26 draak systemd[1]: vks.default.uptimed.uptimed.service: Succeeded.
~~~

## Playing With It

### Debian

1. Install *k3s* and compile the virtual kubelet.
2. Install *policyrcd-script-zg2*: `apt-get install policyrcd-script-zg2` See
   To install daemons without starting them you will need the `policyrcd-script-zg2` package.
   'vks' will attempt to install this on startup.

I'm using `uptimed` as a very simple daemon that you (probably) haven't got installed, so we can
check the entire flow. My testing happens on Debian/Ubuntu.

3. `./k3s/kubectl apply -f uptimed.yaml`

The above *should* yield:

~~~
NAME      READY   STATUS    RESTARTS   AGE
uptimed   1/1     Running   0          7m42s
~~~

You can then delete the pod.
