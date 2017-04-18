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

	"k8s.io/kubernetes/pkg/util/mount"
	volutil "k8s.io/kubernetes/pkg/volume/util"
)

type drbd struct {
	volName      string
	NodeName     string
	ResourceName string
	ReadOnly     bool
	mounter      *mount.SafeFormatAndMount
	// Utility interface that provides API calls to the provider to attach/detach disks.
	manager diskManager
}

type drbdMounter struct {
	*drbd
	fsType string
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

type drbdUnmounter struct {
	*drbdMounter
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
