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
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/golang/glog"
	"k8s.io/kubernetes/pkg/util/mount"
	"k8s.io/kubernetes/pkg/volume"
)

const fieldSep = ","

type DRBDUtil struct{}

func (util *DRBDUtil) MakeGlobalPDName(d drbd) string {
	return makePDNameInternal(d.plugin.host, d.ResourceName)
}

// Make a directory like /var/lib/kubelet/plugins/kubernetes.io/drbd/resource
func makePDNameInternal(host volume.VolumeHost, resourceName string) string {
	return path.Join(host.GetPluginDir(drbdPluginName), resourceName)
}

func (util *DRBDUtil) AttachDisk(disk drbdMounter) error {
	d := *disk.drbd

	hostName, err := getUname(d)
	if err != nil {
		return fmt.Errorf("DRBD: Unable to determine hostname: %v", err)
	}
	// Assign the resource to the Kubelet.
	ok, err := assignRes(d)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("DRBD: Unable to assign resouce %q on node %q", d.ResourceName, hostName)
	}

	// If we can't determine the device path, better to exit now before promotion occurs.
	devicePath, err := waitForDevPath(d, 10)
	if err != nil {
		return err
	}

	// Manually promoting is not strictly nessesary, but allows for less ambigous error
	// reporting than "device does not exist" on mount.
	// Sleep here to give the resource time to establish it's disk state.
	time.Sleep(time.Millisecond * 200)
	out, err := d.plugin.exe.Command("drbdadm", "primary", d.ResourceName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("DRBD: Unable to make resource %q primary on node %q: %s", d.ResourceName, hostName, out)
	}

	// mount it
	globalPDPath := d.manager.MakeGlobalPDName(d)
	notMnt, err := d.mounter.IsLikelyNotMountPoint(globalPDPath)
	// in the first time, the path shouldn't exist and IsLikelyNotMountPoint is expected to get NotExist
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("DRBD: %s failed to check mountpoint", globalPDPath)
	}
	if !notMnt {
		return nil
	}

	if err = os.MkdirAll(globalPDPath, 0750); err != nil {
		return fmt.Errorf("DRBD: failed to mkdir %s, error", globalPDPath)
	}

	if err = d.mounter.FormatAndMount(devicePath, globalPDPath, disk.fsType, nil); err != nil {
		err = fmt.Errorf("DRBD: failed to mount %s [%s] to %s, error %v", devicePath, disk.fsType, globalPDPath, err)
	}
	return err
}

func (util *DRBDUtil) DetachDisk(c drbdUnmounter, mntPath string) error {
	device, cnt, err := mount.GetDeviceNameFromMount(c.mounter, mntPath)
	if err != nil {
		return fmt.Errorf("DRBD detach disk: failed to get device from mnt: %s\nError: %v", mntPath, err)
	}
	if err = c.mounter.Unmount(mntPath); err != nil {
		return fmt.Errorf("DRBD detach disk: failed to umount: %s\nError: %v", mntPath, err)
	}

	res, err := getResFromDevice(*c.drbd, device)
	if err != nil {
		return err
	}
	c.ResourceName = res

	// If device is no longer used and is assigned as a client, see if we can unassign.
	// Client resources do not have local storage and are safe to unassign automatically.
	if cnt <= 1 && isClient(*c.drbd) {

		hostName, err := getUname(*c.drbd)
		if err != nil {
			return fmt.Errorf("DRBD: Unable to determine hostname for unassignment: %v", err)
		}

		glog.Infof("DRBD: Unassigning resource %q from node %q", c.ResourceName, hostName)
		// Demote resource to allow unmounting.
		out, err := c.plugin.exe.Command("drbdadm", "secondary", c.ResourceName).CombinedOutput()
		if err != nil {
			return fmt.Errorf("DRBD: failed to demote resource %q on node %q. Error: %s", c.ResourceName, hostName, out)
		}

		// Unassign resource from the kubelet.
		out, err = c.plugin.exe.Command("drbdmanage", "unassign-resource", c.ResourceName, hostName, "--quiet").CombinedOutput()
		if err != nil {
			return fmt.Errorf("DRBD: failed to unassign resource %q from node %q. Error: %s", c.ResourceName, hostName, out)
		}
		ok, err := waitForUnassignment(*c.drbd, 3)
		if err != nil {
			return fmt.Errorf("DRBD: failed to unassign resource %q from node %q. Error: %v", c.ResourceName, hostName, err)
		}
		if !ok {
			return fmt.Errorf("DRBD: failed to unassign resource %q from node %q. Error: Resource still assigned", c.ResourceName, hostName)
		}

		glog.Infof("DRBD: successfully unassigned resource %q from node %q", c.ResourceName, hostName)
	}

	return nil
}

func waitForDevPath(d drbd, maxRetries int) (string, error) {
	var path string
	var err error

	for i := 0; i < maxRetries; i++ {
		path, err = getDevPath(d)
		if path != "" {
			return path, err
		}
		time.Sleep(time.Second * 2)
	}
	return path, err
}

func getDevPath(d drbd) (string, error) {
	out, err := d.plugin.exe.Command("drbdmanage", "list-volumes", "--resources", d.ResourceName, "--machine-readable").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("DRBD: Unable to get volume information: %s", out)
	}

	devicePath, err := doGetDevPath(string(out))
	if err != nil {
		return "", err
	}

	if _, err := os.Lstat(devicePath); err != nil {
		return "", fmt.Errorf("DRBD: Couldn't stat %s: %v", devicePath, err)
	}

	return devicePath, nil
}

func doGetDevPath(volInfo string) (string, error) {
	if volInfo == "" {
		return "", fmt.Errorf("DRBD: Resource is not configured")
	}

	s := strings.Split(volInfo, fieldSep)
	if len(s) != 7 {
		return "", fmt.Errorf("DRBD: Malformed volume string: %q", volInfo)
	}

	minor := s[5]
	ok, err := regexp.MatchString("\\d+", minor)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("DRBD: Bad device minor %q in volume string: %q", minor, volInfo)
	}

	return "/dev/drbd" + minor, nil
}

func assignRes(d drbd) (bool, error) {
	// Make sure the resource is defined before trying to assign it.
	if ok, err := resExists(d); err != nil || !ok {
		return ok, err
	}

	// If the resource is already assigned, we're done.
	if ok, err := resAssigned(d); err != nil || ok {
		return ok, err
	}

	hostName, err := getUname(d)
	if err != nil {
		return false, fmt.Errorf("DRBD: Unable to determine hostname: %v", err)
	}

	out, err := d.plugin.exe.Command("drbdmanage", "assign-resource", d.ResourceName, hostName, "--client").CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("DRBD: Unable to assign resource %q on node %q: %s", d.ResourceName, hostName, out)
	}
	return waitForAssignment(d, 5)
}

func resExists(d drbd) (bool, error) {
	out, err := d.plugin.exe.Command("drbdmanage", "list-resources", "--resources", d.ResourceName, "--machine-readable").CombinedOutput()
	if err != nil {
		return false, err
	}

	// Inject real implementations here, test through the internal function.
	return doResExists(d.ResourceName, string(out))
}

func doResExists(resource, resInfo string) (bool, error) {
	if resInfo == "" {
		return false, fmt.Errorf("DRBD: Resource %q not defined.", resource)
	}
	if strings.Split(resInfo, fieldSep)[0] != resource {
		return false, fmt.Errorf("DRBD: Error retriving resource information from the following output: %q", resInfo)
	}

	return true, nil
}

// Poll drbdmanage until resource assignment is complete.
func waitForAssignment(d drbd, maxRetries int) (bool, error) {
	for i := 0; i < maxRetries; i++ {
		// If there are no errors and the resource is assigned, we can exit early.
		if ok, err := resAssigned(d); err == nil && ok {
			return ok, nil
		}
		// See if we can recover from any errors or complete pending state changes.
		retryFailedActions(d)
	}
	// Return any errors that might have prevented resource assignment.
	return resAssigned(d)
}

// Poll drbdmanage until resource unassignment is complete.
func waitForUnassignment(d drbd, maxRetries int) (bool, error) {
	for i := 0; i < maxRetries; i++ {
		// If there are no errors and the resource is unassigned, we can exit early.
		if ok, err := resAssigned(d); err == nil && !ok {
			return !ok, nil
		}
		// See if we can recover from any errors or complete pending state changes.
		retryFailedActions(d)
	}
	// Return any errors that might have prevented resource unassignment.
	ok, err := resAssigned(d)
	return !ok, err
}

func resAssigned(d drbd) (bool, error) {
	hostName, err := getUname(d)
	if err != nil {
		return false, fmt.Errorf("DRBD: Unable to determine hostname: %v", err)
	}

	out, err := d.plugin.exe.Command("drbdmanage", "list-assignments", "--resources", d.ResourceName, "--nodes", hostName, "--machine-readable").CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("%s", out)
	}
	return doResAssigned(string(out))
}

func doResAssigned(assignmentInfo string) (bool, error) {
	if assignmentInfo == "" {
		return false, nil
	}

	fields := strings.Split(assignmentInfo, fieldSep)
	if len(fields) != 5 {
		return false, fmt.Errorf("DRBD: Malformed assignmentInfo: %q", assignmentInfo)
	}

	// Target state differs from current state.
	// The assignment exists, but is in a transient state or unhealthy.
	currentState := strings.TrimSpace(fields[3])
	targetState := strings.TrimSpace(fields[4])
	if currentState != targetState {
		return false, fmt.Errorf("DRBD: Assignment targetState %q differs from currentState %q", targetState, currentState)
	}

	return true, nil
}

func retryFailedActions(d drbd) {
	d.plugin.exe.Command("drbdmanage", "resume-all").CombinedOutput()
	time.Sleep(time.Second * 2)
}

func isClient(d drbd) bool {
	hostName, err := getUname(d)
	if err != nil {
		return false
	}

	out, err := d.plugin.exe.Command("drbdmanage", "list-assignments", "--resources", d.ResourceName, "--nodes", hostName, "--machine-readable").CombinedOutput()
	if err != nil {
		return false
	}
	return doIsClient(string(out))
}

func doIsClient(assignmentInfo string) bool {
	// No assignment for the resource on the host.
	if assignmentInfo == "" {
		return false
	}
	fields := strings.Split(assignmentInfo, fieldSep)
	if len(fields) != 5 {
		return false
	}

	targetState := strings.TrimSpace(fields[4])

	if targetState != "connect|deploy|diskless" {
		return false
	}

	return true
}

// DRBD Manage node names must match the output of `uname -n`.
func getUname(d drbd) (string, error) {
	out, err := d.plugin.exe.Command("uname", "-n").CombinedOutput()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

func getResFromDevice(d drbd, device string) (string, error) {
	minor, err := getMinorFromDevice(device)
	if err != nil {
		return "", err
	}

	out, err := d.plugin.exe.Command("drbdmanage", "list-volumes", "--machine-readable").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("DRBD: Unable to get volume information: %s", out)
	}

	res, err := getResFromVolumes(string(out), minor)
	if err != nil {
		return "", err
	}

	return res, nil
}

func getMinorFromDevice(device string) (string, error) {
	if ok, _ := regexp.MatchString("/dev/drbd\\d+", device); !ok {
		return "", fmt.Errorf("DRBD: Tried to get minor from non-DRBD device: %q", device)
	}

	return device[9:], nil
}

func getResFromVolumes(volumes, minor string) (string, error) {
	vols := strings.Split(volumes, "\n")
	for _, v := range vols {
		fields := strings.Split(v, fieldSep)

		// If we get badly formatted volume info, skip it: the next one might be ok.
		if len(fields) != 7 {
			continue
		}
		if fields[5] == minor {
			return fields[0], nil
		}
	}
	return "", nil
}
