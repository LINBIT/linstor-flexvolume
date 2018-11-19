/*
* Linstor Flexvolume plugin for Kubernetes.
* Copyright © 2018 LINBIT USA LLC
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
	"io"
	"log"
	"log/syslog"
	"os"
	"strconv"

	linstor "github.com/LINBIT/golinstor"
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
	return fmt.Sprintf("Linstor Flexvoume API: %s", e.message)
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
	Attached bool `json:"attached"`
}

type getVolNameResponse struct {
	response
	VolumeName string `json:"volumeName"`
}

type options struct {
	// K8s volume options.
	FsType      string `json:"kubernetes.io/fsType"`
	Readwrite   string `json:"kubernetes.io/readwrite"`
	PVCResource string `json:"kubernetes.io/pvOrVolumeName"`

	// Homegrown volume options.
	Resource            string `json:"resource"`
	BlockSize           string `json:"blockSize"`
	Force               string `json:"force"`
	XFSDiscardBlocks    string `json:"xfsDiscardBlocks"`
	XFSDataSU           string `json:"xfsDataSu"`
	XFSDataSW           string `json:"xfsDataSw"`
	XFSLogDev           string `json:"xfsLogDev"`
	DisklessStoragePool string `json:"disklessStoragePool"`
	MountOpts           string `json:"mountOpts"`
	FSOpts              string `json:"fsOpts"`

	// Parsed option ready to pass to linstor.FSUtil
	xfsDataSW        int
	blockSize        int64
	force            bool
	xfsdiscardblocks bool
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

	// BlockSizes of zero are ignored by FSUtil
	if opts.BlockSize == "" {
		opts.BlockSize = "0"
	}
	opts.blockSize, err = strconv.ParseInt(opts.BlockSize, 10, 32)
	if err != nil {
		return opts, err
	}

	if opts.XFSDataSW == "" {
		opts.XFSDataSW = "0"
	}
	xfsdatasw, err := strconv.ParseInt(opts.XFSDataSW, 10, 32)
	if err != nil {
		return opts, err
	}
	opts.xfsDataSW = int(xfsdatasw)

	if opts.Force == "" {
		opts.Force = "false"
	}
	opts.force, err = strconv.ParseBool(opts.Force)
	if err != nil {
		return opts, err
	}

	if opts.XFSDiscardBlocks == "" {
		opts.XFSDiscardBlocks = "false"
	}
	opts.xfsdiscardblocks, err = strconv.ParseBool(opts.XFSDiscardBlocks)
	if err != nil {
		return opts, err
	}

	return opts, nil
}

var logOutput io.Writer

func init() {
	out, err := syslog.New(syslog.LOG_INFO, "Linstor FlexVolume")
	if err != nil {
		log.Fatal(err)
	}

	logOutput = out
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

	resource := linstor.NewResourceDeployment(linstor.ResourceDeploymentConfig{
		Name:                opts.getResource(),
		ClientList:          []string{s[2]},
		DisklessStoragePool: opts.DisklessStoragePool,
		LogOut:              logOutput,
	})

	err = resource.Assign()
	if err != nil {
		res, _ := json.Marshal(response{
			Status:  "Failure",
			Message: flexAPIErr{fmt.Sprintf("%s: failed to assign resource %s: %v", s[0], resource.Name, err)}.Error(),
		})
		return string(res), EXITDRBDFAILURE
	}

	// Only one resource is attached at a time: it's safe to assume the zeroth
	// element is node that we want.
	path, err := resource.GetDevPath(resource.ClientList[0], false)
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

	resource := linstor.NewResourceDeployment(
		linstor.ResourceDeploymentConfig{
			Name:   s[1],
			LogOut: logOutput,
		})

	// Do not unassign resources that have local storage.
	if !resource.IsClient(s[2]) {
		res, _ := json.Marshal(response{Status: "Success"})
		return string(res), EXITSUCCESS
	}

	err := resource.Unassign(s[2])
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
	r := linstor.NewResourceDeployment(
		linstor.ResourceDeploymentConfig{Name: opts.getResource(),
			LogOut: logOutput,
		})

	mounter := linstor.FSUtil{
		ResourceDeployment: &r,
		FSType:             opts.FsType,
		BlockSize:          opts.blockSize,
		Force:              opts.force,
		XFSDiscardBlocks:   opts.xfsdiscardblocks,
		XFSDataSU:          opts.XFSDataSU,
		XFSDataSW:          opts.xfsDataSW,
		XFSLogDev:          opts.XFSLogDev,
		MountOpts:          opts.MountOpts,
		FSOpts:             opts.FSOpts,
	}

	localNode, err := os.Hostname()
	if err != nil {
		res, _ := json.Marshal(response{
			Status:  "Failure",
			Message: flexAPIErr{fmt.Sprintf("%s: %v", s[0], err)}.Error(),
		})
		return string(res), EXITDRBDFAILURE
	}

	err = mounter.Mount(s[1], localNode)
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
	umounter := linstor.FSUtil{}

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

	resource := linstor.NewResourceDeployment(
		linstor.ResourceDeploymentConfig{Name: opts.getResource(),
			LogOut: logOutput,
		})

	ok, err := resource.OnNode(s[2])
	if err != nil {
		res, _ := json.Marshal(response{
			Status:  "Failure",
			Message: flexAPIErr{fmt.Sprintf("%s: %v", s[0], err)}.Error(),
		})
		return string(res), EXITDRBDFAILURE
	}

	if !ok {
		res, _ := json.Marshal(isAttachedResponse{
			Attached: ok,
			response: response{Status: "Failure"},
		})
		return string(res), EXITSUCCESS
	}

	res, _ := json.Marshal(isAttachedResponse{
		Attached: ok,
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
