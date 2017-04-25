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
)

type Resource struct {
	ResourceName string
	NodeName     string
	ReadOnly     bool
	mounter      drbdMounter
}

type drbdMounter struct {
	fsType string
}

const fieldSep = ","

func waitForDevPath(d Resource, maxRetries int) (string, error) {
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

func getDevPath(d Resource) (string, error) {
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

func AssignRes(d Resource) (bool, error) {
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

func resExists(d Resource) (bool, error) {
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
func WaitForAssignment(d Resource, maxRetries int) (bool, error) {
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
func waitForUnassignment(d Resource, maxRetries int) (bool, error) {
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

func resAssigned(d Resource) (bool, error) {
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

func retryFailedActions(d Resource) {
	exec.Command("drbdmanage", "resume-all").CombinedOutput()
	time.Sleep(time.Second * 2)
}

func isClient(d Resource) bool {
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

func getResFromDevice(d Resource, device string) (string, error) {
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
