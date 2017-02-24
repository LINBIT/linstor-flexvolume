package api

func init() {
	flexAPIs = append(flexAPIs, v160{})
}

type v160 struct {
}

func (v v160) apiVersion() string {
	return "1.6.0"
}

func (v v160) Call(s string) string {
	return "{\"status\": \"Failure\"}"
}
