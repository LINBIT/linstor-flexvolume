# DRBD flexvolume plugin for Kubernetes

## Building

Requires Go 1.8 or higher and a configured GOPATH

`mkdir -p $GOPATH/src/github.com/linbit/`

`cd $GOPATH/src/github.com/linbit/`

`git clone https://github.com/linbit/drbd-flexvolume`

`cd drbd-flexvolume`

`go get ./...`

`make`

This will compile a binary targeting the local machine's architecture and
place it into the `_build` directory.

## Installing

Place the generated binary named `drbd` under the following path on the kubelet and
kube-controller-manager nodes: 

```bash
/usr/libexec/kubernetes/kubelet-plugins/volume/exec/linbit~drbd/
```
After installation, restarting kubelet process is required on each node
for Kubernetes versions older than 1.8.

## Usage

Resources must be created before attachment with DRBD Manage or
[drbd-flex-provision](https://github.com/LINBIT/drbd-flex-provision) before
they are available for this plugin to use.

The kube-controller-manager and all kubelets eligible to run containers must be
part of the same DRBD Manage cluster. Volumes will be attached to the kubelet
across the network via the DRBD Transport protocol, so they do not require local
storage.

Kubelet nodes names must match the output of `uname -n` exactly. If they do not,
this may be overridden via the kubelet `--hostname-override` parameter

Please note that the Kubernetes PV name and the associated DRBD Manage resource
name **must** match exactly in order for the volume to remain attached to the
kubelet due to https://github.com/kubernetes/kubernetes/issues/44737

`example.yaml`, located in the root of this project, contains an example
configuration that attaches a resource named `r0` to the container under the path
`/data`. Note that that the PV name is also named `r0`.
