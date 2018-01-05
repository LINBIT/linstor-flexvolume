/*
* DRBD Flexvolume plugin for Kubernetes.
* Copyright © 2017 LINBIT USA LLC
*
* This program is free software; you can redistribute it and/or modify
* it under the terms of the GNU General Public License as published by
* the Free Software Foundation; either version 2 of the License, or
* (at your option) any later version.
*
* This program is distributed in the hope that it will be useful,
* but WITHOUT ANY WARRANTY; without even the implied warranty of
* MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
* GNU General Public License for more details.
*
* You should have received a copy of the GNU General Public License
* along with this program; if not, see <http://www.gnu.org/licenses/>.
 */

package api

import (
	"encoding/json"
	"fmt"

	"github.com/linbit/drbd-flexvolume/pkg/drbd"
)

// API status codes, used as exit codes in main.
const (
	EXITSUCCESS int = iota
	EXITDRBDFAILURE
	EXITBADAPICALL
)

type flexAPIErr struct {
	message string
}

func (e flexAPIErr) Error() string {
	return fmt.Sprintf("DRBD Flexvoume API: %s", e.message)
}

type response struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type attachResponse struct {
	response
	Device string `json:"device"`
}

type isAttachedResponse struct {
	response
	Attached string `json:"attached"`
}

type getVolNameResponse struct {
	response
	VolumeName string `json:"volumeName"`
}

type options struct {
	FsType      string `json:"kubernetes.io/fsType"`
	Readwrite   string `json:"kubernetes.io/readwrite"`
	Resource    string `json:"resource"`
	PVCResource string `json:"kubernetes.io/pvOrVolumeName"`
}

func (o *options) getResource() string {
	if o.Resource != "" {
		return o.Resource
	}
	return o.PVCResource
}

func parseOptions(s string) (options, error) {
	opts := options{}
	err := json.Unmarshal([]byte(s), &opts)
	if err != nil {
		return opts, flexAPIErr{fmt.Sprintf("couldn't parse options from %s", s)}
	}

	return opts, nil
}

type FlexVolumeApi struct{}

func (api FlexVolumeApi) Call(s []string) (string, int) {
	if len(s) < 1 {
		res, _ := json.Marshal(response{
			Status:  "Failure",
			Message: flexAPIErr{"No driver action! Valid actions are: init, attach, detach, mountdevice, unmountdevice, isattached"}.Error(),
		})
		return string(res), EXITBADAPICALL
	}
	switch s[0] {
	case "init":
		return api.init()
	case "attach":
		return api.attach(s)
	case "waitforattach":
		return api.waitForAttach(s)
	case "detach":
		return api.detach(s)
	case "mountdevice":
		return api.mountDevice(s)
	case "unmountdevice":
		return api.unmountDevice(s)
	case "unmount":
		return api.unmount(s)
	case "isattached":
		return api.isAttached(s)
	default:
		res, _ := json.Marshal(response{
			Status:  "Not supported",
			Message: flexAPIErr{fmt.Sprintf("Unsupported driver action: %s", s[0])}.Error(),
		})
		return string(res), EXITBADAPICALL
	}
}

func (api FlexVolumeApi) init() (string, int) {
	res, _ := json.Marshal(response{Status: "Success"})
	return string(res), EXITSUCCESS
}

func (api FlexVolumeApi) attach(s []string) (string, int) {
	if len(s) < 3 {
		return tooFewArgsResponse(s)
	}

	opts, err := parseOptions(s[1])
	if err != nil {
		res, _ := json.Marshal(response{
			Status:  "Failure",
			Message: flexAPIErr{fmt.Sprintf("%s: %v", s[0], err)}.Error(),
		})
		return string(res), EXITBADAPICALL
	}

	resource := drbd.Resource{Name: opts.getResource(), NodeName: s[2]}

	_, err = drbd.AssignRes(resource)
	if err != nil {
		res, _ := json.Marshal(response{
			Status:  "Failure",
			Message: flexAPIErr{fmt.Sprintf("%s: failed to assign resource %s: %v", s[0], resource.Name, err)}.Error(),
		})
		return string(res), EXITDRBDFAILURE
	}

	path, err := drbd.WaitForDevPath(resource, 4)
	if err != nil {
		res, _ := json.Marshal(response{
			Status:  "Failure",
			Message: flexAPIErr{fmt.Sprintf("%s: unable to find device path for resource %s: %v", s[0], resource.Name, err)}.Error(),
		})
		return string(res), EXITDRBDFAILURE
	}

	res, _ := json.Marshal(attachResponse{
		Device: path,
		response: response{
			Status: "Success",
		},
	})
	return string(res), EXITSUCCESS
}

func (api FlexVolumeApi) waitForAttach(s []string) (string, int) {
	res, _ := json.Marshal(response{Status: "Success"})
	return string(res), EXITSUCCESS
}

func (api FlexVolumeApi) detach(s []string) (string, int) {
	if len(s) < 3 {
		return tooFewArgsResponse(s)
	}

	resource := drbd.Resource{Name: s[1], NodeName: s[2]}

	// Do not unassign resources that have local storage.
	if !drbd.IsClient(resource) {
		res, _ := json.Marshal(response{Status: "Success"})
		return string(res), EXITSUCCESS
	}

	err := drbd.UnassignRes(resource)
	if err != nil {
		res, _ := json.Marshal(response{
			Status:  "Failure",
			Message: flexAPIErr{fmt.Sprintf("%s: %v", s[0], err)}.Error(),
		})
		return string(res), EXITDRBDFAILURE
	}

	res, _ := json.Marshal(response{Status: "Success"})
	return string(res), EXITSUCCESS
}

func (api FlexVolumeApi) mountDevice(s []string) (string, int) {
	if len(s) < 4 {
		return tooFewArgsResponse(s)
	}

	opts, err := parseOptions(s[3])
	if err != nil {
		res, _ := json.Marshal(response{
			Status:  "Failure",
			Message: flexAPIErr{fmt.Sprintf("%s: %v", s[0], err)}.Error(),
		})
		return string(res), EXITBADAPICALL
	}

	mounter := drbd.Mounter{
		Resource: &drbd.Resource{
			Name: opts.getResource()},
		FSType: opts.FsType,
	}

	err = mounter.Mount(s[1])
	if err != nil {
		res, _ := json.Marshal(response{
			Status:  "Failure",
			Message: flexAPIErr{fmt.Sprintf("%s: %v", s[0], err)}.Error(),
		})
		return string(res), EXITDRBDFAILURE
	}

	res, _ := json.Marshal(response{Status: "Success"})
	return string(res), EXITSUCCESS
}

func (api FlexVolumeApi) unmountDevice(s []string) (string, int) {
	return api.unmount(s)
}

func (api FlexVolumeApi) unmount(s []string) (string, int) {
	if len(s) < 2 {
		return tooFewArgsResponse(s)
	}
	umounter := drbd.Mounter{}

	err := umounter.UnMount(s[1])
	if err != nil {
		res, _ := json.Marshal(response{
			Status:  "Failure",
			Message: flexAPIErr{fmt.Sprintf("%s: %v", s[0], err)}.Error(),
		})
		return string(res), EXITDRBDFAILURE
	}
	res, _ := json.Marshal(response{Status: "Success"})
	return string(res), EXITSUCCESS
}

func (api FlexVolumeApi) getVolumeName(s []string) (string, int) {
	if len(s) < 2 {
		return tooFewArgsResponse(s)
	}

	opts, err := parseOptions(s[1])
	if err != nil {
		res, _ := json.Marshal(response{
			Status:  "Failure",
			Message: flexAPIErr{fmt.Sprintf("%s: %v", s[0], err)}.Error(),
		})
		return string(res), EXITBADAPICALL
	}

	res, _ := json.Marshal(getVolNameResponse{
		VolumeName: opts.getResource(),
		response: response{
			Status: "Success",
		},
	})
	return string(res), EXITSUCCESS
}

func (api FlexVolumeApi) isAttached(s []string) (string, int) {
	if len(s) < 3 {
		return tooFewArgsResponse(s)
	}

	opts, err := parseOptions(s[1])
	if err != nil {
		res, _ := json.Marshal(response{
			Status:  "Failure",
			Message: flexAPIErr{fmt.Sprintf("%s: %v", s[0], err)}.Error(),
		})
		return string(res), EXITBADAPICALL
	}

	resource := drbd.Resource{Name: opts.getResource(), NodeName: s[2]}

	ok, err := drbd.WaitForAssignment(resource, 4)
	if err != nil {
		res, _ := json.Marshal(response{
			Status:  "Failure",
			Message: flexAPIErr{fmt.Sprintf("%s: %v", s[0], err)}.Error(),
		})
		return string(res), EXITDRBDFAILURE
	}

	if !ok {
		res, _ := json.Marshal(response{
			Status:  "Failure",
			Message: flexAPIErr{fmt.Sprintf("%s: resource %s not attached", s[0], resource.Name)}.Error(),
		})
		return string(res), EXITDRBDFAILURE
	}

	res, _ := json.Marshal(isAttachedResponse{
		Attached: "true",
		response: response{Status: "Success"},
	})
	return string(res), EXITSUCCESS
}

func tooFewArgsResponse(s []string) (string, int) {
	res, _ := json.Marshal(response{
		Status:  "Failure",
		Message: flexAPIErr{fmt.Sprintf("%s: too few arguments passed: %s", s[0], s)}.Error(),
	})
	return string(res), EXITBADAPICALL
}
