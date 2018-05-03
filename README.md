# Linstor flexvolume plugin for Kubernetes

## Building

Requires Go 1.8 or higher and a configured GOPATH

`mkdir -p $GOPATH/src/github.com/LINBIT/`

`cd $GOPATH/src/github.com/LINBIT/`

`git clone https://github.com/LINBIT/linstor-flexvolume`

`cd linstor-flexvolume`

`make`

This will compile a binary targeting the local machine's architecture and
place it into the root of the project.

## Installing

Place the generated binary named `linstor-flexvolume` under the following path 
on the kubelet and kube-controller-manager nodes: 

```bash
/usr/libexec/kubernetes/kubelet-plugins/volume/exec/linbit~linstor-flexvolume/
```
After installation, restarting kubelet process is required on each node
for Kubernetes versions older than 1.8.

You must set the `--enable-controller-attach-detach=false` option on all
kubelets. For systemd managed kubelets this can be set in
`/etc/systemd/system/kubelet.service.d/10-kubeadm.conf`

## Usage

Resources must be created before attachment with Linstor or
[linstor-external-provisioner](https://github.com/LINBIT/linstor-external-provisioner) before
they are available for this plugin to use.

The kube-controller-manager and all kubelets eligible to run containers must be
part of the same Linstor cluster. Volumes will be attached to the kubelet
across the network via the DRBD Transport protocol, so they do not require local
storage. You will need to install the Linstor Client on each kubelet and you
must configure a list of controllers via configuration files such as
`/etc/linstor/linstor-client.conf`, rather than using the `LS_CONTROLLERS`
environment variable.

Kubelet nodes names must match the output of `uname -n` exactly. If they do not,
this may be overridden via the kubelet `--hostname-override` parameter

Please note that the Kubernetes PV name and the associated Linstor resource
name **must** match exactly in order for the volume to remain attached to the
kubelet due to https://github.com/kubernetes/kubernetes/issues/44737

`example.yaml`, located in the root of this project, contains an example
configuration that attaches a resource named `r0` to the container under the path
`/data`. Note that that the PV name is also named `r0`.
