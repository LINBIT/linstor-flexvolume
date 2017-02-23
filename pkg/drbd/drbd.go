/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package drbd

import (
	"fmt"

	"github.com/golang/glog"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/pkg/api/v1"
	"k8s.io/kubernetes/pkg/util/exec"
	"k8s.io/kubernetes/pkg/util/mount"
	utilstrings "k8s.io/kubernetes/pkg/util/strings"
	"k8s.io/kubernetes/pkg/volume"
	volutil "k8s.io/kubernetes/pkg/volume/util"
)

// This is the primary entrypoint for volume plugins.
func ProbeVolumePlugins() []volume.VolumePlugin {
	return []volume.VolumePlugin{&drbdPlugin{nil, exec.New()}}
}

type drbdPlugin struct {
	host volume.VolumeHost
	exe  exec.Interface
}

var _ volume.VolumePlugin = &drbdPlugin{}
var _ volume.PersistentVolumePlugin = &drbdPlugin{}

const (
	drbdPluginName = "kubernetes.io/drbd"
)

func (plugin *drbdPlugin) Init(host volume.VolumeHost) error {
	plugin.host = host
	return nil
}

func (plugin *drbdPlugin) GetPluginName() string {
	return drbdPluginName
}

func (plugin *drbdPlugin) GetVolumeName(spec *volume.Spec) (string, error) {
	volumeSource, _, err := getVolumeSource(spec)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%v", volumeSource.ResourceName), nil
}

func (plugin *drbdPlugin) CanSupport(spec *volume.Spec) bool {
	return (spec.Volume != nil && spec.Volume.DRBD != nil) || (spec.PersistentVolume != nil && spec.PersistentVolume.Spec.DRBD != nil)
}

func (plugin *drbdPlugin) RequiresRemount() bool {
	return false
}

func (plugin *drbdPlugin) NewMounter(spec *volume.Spec, pod *v1.Pod, _ volume.VolumeOptions) (volume.Mounter, error) {
	return plugin.newMounterInternal(spec, pod.UID, &DRBDUtil{}, plugin.host.GetMounter())
}

func (plugin *drbdPlugin) newMounterInternal(spec *volume.Spec, podUID types.UID, manager diskManager, mounter mount.Interface) (volume.Mounter, error) {
	source, readOnly, err := getVolumeSource(spec)
	if err != nil {
		return nil, err
	}
	return &drbdMounter{
		drbd: &drbd{
			volName:      spec.Name(),
			podUID:       podUID,
			ResourceName: source.ResourceName,
			ReadOnly:     readOnly,
			plugin:       plugin,
			mounter:      &mount.SafeFormatAndMount{Interface: mounter, Runner: exec.New()},
			manager:      manager,
		},
		fsType: source.FSType,
	}, nil
}

func getVolumeSource(spec *volume.Spec) (*v1.DRBDVolumeSource, bool, error) {
	if spec.Volume != nil && spec.Volume.DRBD != nil {
		return spec.Volume.DRBD, spec.Volume.DRBD.ReadOnly, nil
	} else if spec.PersistentVolume != nil &&
		spec.PersistentVolume.Spec.DRBD != nil {
		return spec.PersistentVolume.Spec.DRBD, spec.ReadOnly, nil
	}

	return nil, false, fmt.Errorf("Spec does not reference a DRBD volume type")
}

func (plugin *drbdPlugin) NewUnmounter(volName string, podUID types.UID) (volume.Unmounter, error) {
	// Inject real implementations here, test through the internal function.
	return plugin.newUnmounterInternal(volName, podUID, &DRBDUtil{}, plugin.host.GetMounter())
}

func (plugin *drbdPlugin) newUnmounterInternal(volName string, podUID types.UID, manager diskManager, mounter mount.Interface) (volume.Unmounter, error) {
	return &drbdUnmounter{
		&drbdMounter{
			drbd: &drbd{
				volName: volName,
				podUID:  podUID,
				plugin:  plugin,
				mounter: &mount.SafeFormatAndMount{Interface: mounter, Runner: exec.New()},
				manager: manager,
			},
		},
	}, nil
}

func (plugin *drbdPlugin) ConstructVolumeSpec(volumeName string, mountPath string) (*volume.Spec, error) {
	drbdVolume := &v1.Volume{
		Name: volumeName,
		VolumeSource: v1.VolumeSource{
			DRBD: &v1.DRBDVolumeSource{
				ResourceName: volumeName,
			},
		},
	}
	return volume.NewSpecFromVolume(drbdVolume), nil
}

func (plugin *drbdPlugin) GetAccessModes() []v1.PersistentVolumeAccessMode {
	return []v1.PersistentVolumeAccessMode{
		v1.ReadWriteOnce,
		v1.ReadOnlyMany,
	}
}

type drbd struct {
	volName      string
	podUID       types.UID
	ResourceName string
	ReadOnly     bool
	plugin       *drbdPlugin
	mounter      *mount.SafeFormatAndMount
	// Utility interface that provides API calls to the provider to attach/detach disks.
	manager diskManager
	volume.MetricsNil
}

func (d *drbd) GetPath() string {
	name := drbdPluginName
	// safe to use PodVolumeDir now: volume teardown occurs before pod is cleaned up
	return d.plugin.host.GetPodVolumeDir(d.podUID, utilstrings.EscapeQualifiedNameForDisk(name), d.volName)
}

type drbdMounter struct {
	*drbd
	fsType string
}

var _ volume.Mounter = &drbdMounter{}

func (b *drbdMounter) CanMount() error {
	return nil
}

func (b *drbdMounter) SetUp(fsGroup *int64) error {
	return b.SetUpAt(b.GetPath(), fsGroup)
}

func (b *drbdMounter) SetUpAt(dir string, fsGroup *int64) error {
	// diskSetUp checks mountpoints and prevent repeated calls
	glog.V(4).Infof("DRBD: attempting to SetUp and mount %s", dir)
	err := diskSetUp(b.manager, *b, dir, b.mounter, fsGroup)
	if err != nil {
		glog.Errorf("DRBD: failed to setup mount %s %v", dir, err)
	}
	return err
}

func (b *drbdMounter) GetAttributes() volume.Attributes {
	return volume.Attributes{
		ReadOnly:        b.ReadOnly,
		Managed:         !b.ReadOnly,
		SupportsSELinux: true,
	}
}

type drbdUnmounter struct {
	*drbdMounter
}

var _ volume.Unmounter = &drbdUnmounter{}

// Unmounts the bind mount, and detaches the disk only if the disk
// resource was the last reference to that disk on the kubelet.
func (c *drbdUnmounter) TearDown() error {
	return c.TearDownAt(c.GetPath())
}

func (c *drbdUnmounter) TearDownAt(dir string) error {
	if pathExists, pathErr := volutil.PathExists(dir); pathErr != nil {
		return fmt.Errorf("Error checking if path exists: %v", pathErr)
	} else if !pathExists {
		glog.Warningf("Warning: Unmount skipped because path does not exist: %v", dir)
		return nil
	}
	return diskTearDown(c.manager, *c, dir, c.mounter)
}
