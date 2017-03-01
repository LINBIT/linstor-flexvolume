package api

import (
	"encoding/json"
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

func (v v160) Call(s string) string {
	return "{\"status\": \"Failure\"}"
}

func (v v160) init() string {
	res, _ := json.Marshal(responce{Status: "Success"})
	return string(res)
}

func (v v160) attach(s []string) string {
	res, _ := json.Marshal(attachResponce{
		Device: "/dev/drbd100",
		responce: responce{
			Status: "Success",
		},
	})
	return string(res)
}

func (v v160) waitForAttach(s []string) string {
	res, _ := json.Marshal(responce{Status: "Success"})
	return string(res)
}

func (v v160) detach(s []string) string {
	res, _ := json.Marshal(responce{Status: "Success"})
	return string(res)
}

func (v v160) mountDevice(s []string) string {
	res, _ := json.Marshal(responce{Status: "Success"})
	return string(res)
}

func (v v160) unmountDevice(s []string) string {
	res, _ := json.Marshal(responce{Status: "Success"})
	return string(res)
}

func (v v160) getVolumeName(s []string) string {
	res, _ := json.Marshal(getVolNameResponce{
		VolumeName: "test0",
		responce: responce{
			Status: "Success",
		},
	})
	return string(res)
}

func (v v160) isAttached(s []string) string {
	res, _ := json.Marshal(isAttachedResponce{
		Attached: "true",
		responce: responce{Status: "Success"},
	})
	return string(res)
}
