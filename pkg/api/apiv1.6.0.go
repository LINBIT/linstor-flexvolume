package api

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
