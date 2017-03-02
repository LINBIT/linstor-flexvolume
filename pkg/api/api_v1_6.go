package api

import (
	"encoding/json"
	"fmt"
)

func init() {
	flexAPIs = append(flexAPIs, v160{})
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
	FsType     string `json:"fsType"`
	Resource   string `json:"resource"`
	Size       string `json:"size"`
	Site       string `json:"site"`
	Redundancy int    `json:"redundancy"`
}

type v160 struct {
}

func (v v160) apiVersion() string {
	return "1.6.0"
}

func (v v160) Call(s []string) (string, int) {
	if len(s) < 1 {
		res, _ := json.Marshal(responce{
			Status:  "Failure",
			Message: "No driver action! Valid actions are: init, attach, detach, mountdevice, unmountdevice, getvolumename, isattached",
		})
		return string(res), 2
	}
	switch s[0] {
	case "init":
		return v.init()
	case "attach":
		return v.attach(s)
	case "waitforattach":
		return v.waitForAttach(s)
	case "detach":
		return v.detach(s)
	case "mountdevice":
		return v.mountDevice(s)
	case "unmountdevice":
		return v.unmountDevice(s)
	case "getvolumename":
		return v.getVolumeName(s)
	case "isattached":
		return v.isAttached(s)
	default:
		res, _ := json.Marshal(responce{
			Status:  "Failure",
			Message: fmt.Sprintf("Unsupported driver action: %q", s[0]),
		})
		return string(res), 2
	}
}

func (v v160) init() (string, int) {
	res, _ := json.Marshal(responce{Status: "Success"})
	return string(res), 0
}

func (v v160) attach(s []string) (string, int) {
	res, _ := json.Marshal(attachResponce{
		Device: "/dev/drbd100",
		responce: responce{
			Status: "Success",
		},
	})
	return string(res), 0
}

func (v v160) waitForAttach(s []string) (string, int) {
	res, _ := json.Marshal(responce{Status: "Success"})
	return string(res), 0
}

func (v v160) detach(s []string) (string, int) {
	res, _ := json.Marshal(responce{Status: "Success"})
	return string(res), 0
}

func (v v160) mountDevice(s []string) (string, int) {
	res, _ := json.Marshal(responce{Status: "Success"})
	return string(res), 0
}

func (v v160) unmountDevice(s []string) (string, int) {
	res, _ := json.Marshal(responce{Status: "Success"})
	return string(res), 0
}

func (v v160) getVolumeName(s []string) (string, int) {
	res, _ := json.Marshal(getVolNameResponce{
		VolumeName: "test0",
		responce: responce{
			Status: "Success",
		},
	})
	return string(res), 0
}

func (v v160) isAttached(s []string) (string, int) {
	res, _ := json.Marshal(isAttachedResponce{
		Attached: "true",
		responce: responce{Status: "Success"},
	})
	return string(res), 0
}
