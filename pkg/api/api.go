package api

import (
	"encoding/json"
	"fmt"
	"linbit/drbd-flexvolume/pkg/drbd"
)

type flexApiErr struct {
	message string
}

func (e flexApiErr) Error() string {
	return fmt.Sprintf("DRBD Flexvoume API: %s", e.message)
}

type responce struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type attachResponce struct {
	responce
	Device string `json:"device"`
}

type isAttachedResponce struct {
	responce
	Attached string `json:"attached"`
}

type getVolNameResponce struct {
	responce
	VolumeName string `json:"volumeName"`
}

type options struct {
	FsType   string `json:"fsType"`
	Resource string `json:"resource"`
}

type FlexVolumeApi struct {
}

func (api FlexVolumeApi) Call(s []string) (string, int) {
	if len(s) < 1 {
		res, _ := json.Marshal(responce{
			Status:  "Failure",
			Message: "No driver action! Valid actions are: init, attach, detach, mountdevice, unmountdevice, getvolumename, isattached",
		})
		return string(res), 2
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
	case "getvolumename":
		return api.getVolumeName(s)
	case "isattached":
		return api.isAttached(s)
	default:
		res, _ := json.Marshal(responce{
			Status:  "Failure",
			Message: fmt.Sprintf("Unsupported driver action: %q", s[0]),
		})
		return string(res), 2
	}
}

func (api FlexVolumeApi) init() (string, int) {
	res, _ := json.Marshal(responce{Status: "Success"})
	return string(res), 0
}

func (api FlexVolumeApi) attach(s []string) (string, int) {
	if len(s) < 2 {
		res, _ := json.Marshal(responce{
			Status:  "Failure",
			Message: flexApiErr{fmt.Sprintf("attach: too few arguments passed: %s", s)}.Error(),
		})
		return string(res), 2
	}

	opts := options{}
	err := json.Unmarshal([]byte(s[0]), &opts)
	if err != nil {
		res, _ := json.Marshal(responce{
			Status:  "Failure",
			Message: flexApiErr{fmt.Sprintf("attach: couldn't parse options from %s", s[0])}.Error(),
		})
		return string(res), 2
	}

	resource := drbd.Resource{Name: opts.Resource, NodeName: s[1]}

	_, err = drbd.AssignRes(resource)
	if err != nil {
		res, _ := json.Marshal(responce{
			Status:  "Failure",
			Message: flexApiErr{fmt.Sprintf("attach: failed to assign resource %q", resource.Name)}.Error(),
		})
		return string(res), 1
	}

	path, err := drbd.WaitForDevPath(resource, 4)
	if err != nil {
		res, _ := json.Marshal(responce{
			Status:  "Failure",
			Message: flexApiErr{fmt.Sprintf("attach: unable to find device path for resource %q", resource.Name)}.Error(),
		})
		return string(res), 1
	}

	res, _ := json.Marshal(attachResponce{
		Device: path,
		responce: responce{
			Status: "Success",
		},
	})
	return string(res), 0
}

func (api FlexVolumeApi) waitForAttach(s []string) (string, int) {
	res, _ := json.Marshal(responce{Status: "Success"})
	return string(res), 0
}

func (api FlexVolumeApi) detach(s []string) (string, int) {
	res, _ := json.Marshal(responce{Status: "Success"})
	return string(res), 0
}

func (api FlexVolumeApi) mountDevice(s []string) (string, int) {
	res, _ := json.Marshal(responce{Status: "Success"})
	return string(res), 0
}

func (api FlexVolumeApi) unmountDevice(s []string) (string, int) {
	res, _ := json.Marshal(responce{Status: "Success"})
	return string(res), 0
}

func (api FlexVolumeApi) getVolumeName(s []string) (string, int) {
	res, _ := json.Marshal(getVolNameResponce{
		VolumeName: "test0",
		responce: responce{
			Status: "Success",
		},
	})
	return string(res), 0
}

func (api FlexVolumeApi) isAttached(s []string) (string, int) {
	res, _ := json.Marshal(isAttachedResponce{
		Attached: "true",
		responce: responce{Status: "Success"},
	})
	return string(res), 0
}
