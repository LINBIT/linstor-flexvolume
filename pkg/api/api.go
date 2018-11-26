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

type FlexVolumeApi struct {
	action string
}

func (api FlexVolumeApi) fmtAPIError(err error) (string, int) {
	res, _ := json.Marshal(response{
		Status:  "Failure",
		Message: flexAPIErr{fmt.Sprintf("%s: %v", api.action, err)}.Error(),
	})
	return string(res), EXITBADAPICALL
}

func (api *FlexVolumeApi) Call(args []string) (string, int) {
	if len(args) < 1 {
		res, _ := json.Marshal(response{
			Status:  "Failure",
			Message: flexAPIErr{"No driver action! Valid actions are: init, attach, detach, mountdevice, unmountdevice, isattached"}.Error(),
		})
		return string(res), EXITBADAPICALL
	}
	api.action = args[0]
	switch api.action {
	case "init":
		return api.init()
	case "attach":
		if len(args) < 3 {
			return tooFewArgsResponse(args)
		}
		return api.attach(args[1], args[2])
	case "waitforattach":
		return api.waitForAttach()
	case "detach":
		if len(args) < 3 {
			return tooFewArgsResponse(args)
		}
		return api.detach(args[1], args[2])
	case "mountdevice":
		if len(args) < 4 {
			return tooFewArgsResponse(args)
		}
		return api.mountDevice(args[1], args[3])
	case "unmountdevice":
		if len(args) < 2 {
			return tooFewArgsResponse(args)
		}
		return api.unmountDevice(args[1])
	case "unmount":
		if len(args) < 2 {
			return tooFewArgsResponse(args)
		}
		return api.unmount(args[1])
	case "isattached":
		if len(args) < 3 {
			return tooFewArgsResponse(args)
		}
		return api.isAttached(args[1], args[2])
	default:
		res, _ := json.Marshal(response{
			Status:  "Not supported",
			Message: flexAPIErr{fmt.Sprintf("Unsupported driver action: %s", api.action)}.Error(),
		})
		return string(res), EXITBADAPICALL
	}
}

func (api FlexVolumeApi) init() (string, int) {
	res, _ := json.Marshal(response{Status: "Success"})
	return string(res), EXITSUCCESS
}

func (api FlexVolumeApi) attach(rawOpts, node string) (string, int) {
	opts, err := parseOptions(rawOpts)
	if err != nil {
		return api.fmtAPIError(err)
	}

	resource := linstor.NewResourceDeployment(linstor.ResourceDeploymentConfig{
		Name:                opts.getResource(),
		ClientList:          []string{node},
		DisklessStoragePool: opts.DisklessStoragePool,
		LogOut:              logOutput,
	})

	err = resource.Assign()
	if err != nil {
		res, _ := json.Marshal(response{
			Status: "Failure",
			Message: flexAPIErr{fmt.Sprintf(
				"%s: failed to assign resource %s: %v", api.action, resource.Name, err)}.Error(),
		})
		return string(res), EXITDRBDFAILURE
	}

	path, err := resource.GetDevPath(node, false)
	if err != nil {
		res, _ := json.Marshal(response{
			Status:  "Failure",
			Message: flexAPIErr{fmt.Sprintf("%s: unable to find device path for resource %s: %v", api.action, resource.Name, err)}.Error(),
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

func (api FlexVolumeApi) waitForAttach() (string, int) {
	res, _ := json.Marshal(response{Status: "Success"})
	return string(res), EXITSUCCESS
}

func (api FlexVolumeApi) detach(name, node string) (string, int) {

	resource := linstor.NewResourceDeployment(
		linstor.ResourceDeploymentConfig{
			Name:   name,
			LogOut: logOutput,
		})

	// Do not unassign resources that have local storage.
	if !resource.IsClient(node) {
		res, _ := json.Marshal(response{Status: "Success"})
		return string(res), EXITSUCCESS
	}

	err := resource.Unassign(node)
	if err != nil {
		api.fmtAPIError(err)
	}

	res, _ := json.Marshal(response{Status: "Success"})
	return string(res), EXITSUCCESS
}

func (api FlexVolumeApi) mountDevice(path, rawOpts string) (string, int) {
	opts, err := parseOptions(rawOpts)
	if err != nil {
		return api.fmtAPIError(err)
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
		return api.fmtAPIError(err)
	}

	err = mounter.Mount(path, localNode)
	if err != nil {
		return api.fmtAPIError(err)
	}

	res, _ := json.Marshal(response{Status: "Success"})
	return string(res), EXITSUCCESS
}

func (api FlexVolumeApi) unmountDevice(path string) (string, int) {
	return api.unmount(path)
}

func (api FlexVolumeApi) unmount(path string) (string, int) {
	umounter := linstor.FSUtil{}

	err := umounter.UnMount(path)
	if err != nil {
		return api.fmtAPIError(err)
	}
	res, _ := json.Marshal(response{Status: "Success"})
	return string(res), EXITSUCCESS
}

func (api FlexVolumeApi) getVolumeName(s []string) (string, int) {
	opts, err := parseOptions(s[1])
	if err != nil {
		res, _ := json.Marshal(response{
			Status:  "Failure",
			Message: flexAPIErr{fmt.Sprintf("%s: %v", api.action, err)}.Error(),
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

func (api FlexVolumeApi) isAttached(rawOpts, node string) (string, int) {
	opts, err := parseOptions(rawOpts)
	if err != nil {
		return api.fmtAPIError(err)
	}

	resource := linstor.NewResourceDeployment(
		linstor.ResourceDeploymentConfig{Name: opts.getResource(),
			LogOut: logOutput,
		})

	ok, err := resource.OnNode(node)
	if err != nil {
		return api.fmtAPIError(err)
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
