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
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/golang/glog"
)

const fieldSep = ","

type DRBDUtil struct{}

func (util *DRBDUtil) AttachDisk(disk drbdMounter) error {
	d := *disk.drbd

	// Assign the resource to the Kubelet.
	ok, err := AssignRes(d)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("DRBD: Unable to assign resouce %q on node %q", d.ResourceName, d.NodeName)
	}

	// Manually promoting is not strictly nessesary, but allows for less ambigous error
	// reporting than "device does not exist" on mount.
	// Sleep here to give the resource time to establish it's disk state.
	time.Sleep(time.Millisecond * 200)
	out, err := exec.Command("drbdadm", "primary", d.ResourceName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("DRBD: Unable to make resource %q primary on node %q: %s", d.ResourceName, d.NodeName, out)
	}
	return nil
}

func (util *DRBDUtil) DetachDisk(c drbdUnmounter, mntPath string) error {
	// Temp values
	device := "/dev/drbd100"
	cnt := 4

	res, err := getResFromDevice(*c.drbd, device)
	if err != nil {
		return err
	}
	c.ResourceName = res

	// If device is no longer used and is assigned as a client, see if we can unassign.
	// Client resources do not have local storage and are safe to unassign automatically.
	if cnt <= 1 && isClient(*c.drbd) {

		glog.Infof("DRBD: Unassigning resource %q from node %q", c.ResourceName, c.NodeName)
		// Demote resource to allow unmounting.
		out, err := exec.Command("drbdadm", "secondary", c.ResourceName).CombinedOutput()
		if err != nil {
			return fmt.Errorf("DRBD: failed to demote resource %q on node %q. Error: %s", c.ResourceName, c.NodeName, out)
		}

		// Unassign resource from the kubelet.
		out, err = exec.Command("drbdmanage", "unassign-resource", c.ResourceName, c.NodeName, "--quiet").CombinedOutput()
		if err != nil {
			return fmt.Errorf("DRBD: failed to unassign resource %q from node %q. Error: %s", c.ResourceName, c.NodeName, out)
		}
		ok, err := waitForUnassignment(*c.drbd, 3)
		if err != nil {
			return fmt.Errorf("DRBD: failed to unassign resource %q from node %q. Error: %v", c.ResourceName, c.NodeName, err)
		}
		if !ok {
			return fmt.Errorf("DRBD: failed to unassign resource %q from node %q. Error: Resource still assigned", c.ResourceName, c.NodeName)
		}

		glog.Infof("DRBD: successfully unassigned resource %q from node %q", c.ResourceName, c.NodeName)
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
	out, err := exec.Command("drbdmanage", "list-volumes", "--resources", d.ResourceName, "--machine-readable").CombinedOutput()
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

func AssignRes(d drbd) (bool, error) {
	// Make sure the resource is defined before trying to assign it.
	if ok, err := resExists(d); err != nil || !ok {
		return ok, err
	}

	// If the resource is already assigned, we're done.
	if ok, err := resAssigned(d); err != nil || ok {
		return ok, err
	}

	out, err := exec.Command("drbdmanage", "assign-resource", d.ResourceName, d.NodeName, "--client").CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("DRBD: Unable to assign resource %q on node %q: %s", d.ResourceName, d.NodeName, out)
	}
	return WaitForAssignment(d, 5)
}

func resExists(d drbd) (bool, error) {
	out, err := exec.Command("drbdmanage", "list-resources", "--resources", d.ResourceName, "--machine-readable").CombinedOutput()
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
func WaitForAssignment(d drbd, maxRetries int) (bool, error) {
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
	out, err := exec.Command("drbdmanage", "list-assignments", "--resources", d.ResourceName, "--nodes", d.NodeName, "--machine-readable").CombinedOutput()
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
	exec.Command("drbdmanage", "resume-all").CombinedOutput()
	time.Sleep(time.Second * 2)
}

func isClient(d drbd) bool {
	out, err := exec.Command("drbdmanage", "list-assignments", "--resources", d.ResourceName, "--nodes", d.NodeName, "--machine-readable").CombinedOutput()
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

func getResFromDevice(d drbd, device string) (string, error) {
	minor, err := getMinorFromDevice(device)
	if err != nil {
		return "", err
	}

	out, err := exec.Command("drbdmanage", "list-volumes", "--machine-readable").CombinedOutput()
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
