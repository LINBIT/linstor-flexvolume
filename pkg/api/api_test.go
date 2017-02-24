package api

import "testing"

func TestDoGetKubeServerVersion(t *testing.T) {
	var versionTests = []struct {
		in  string
		out string
	}{
		{"Client Version: v1.7.0-alpha.0.41+2e1271116023d6\nThe connection to the server localhost:8080 was refused - did you specify the right host or port?", ""},
		{"Client Version: v1.7.0-alpha.0.41+2e1271116023d6\nServer Version: v1.7.0-alpha.0.41+2e1271116023d6", "v1.7.0"},
		{"Client Version: v1.7.0-alpha.0.41+2e1271116023d6\nServer Version: v1.6.0-beta.3.41+2e1271116023d6", "v1.6.0"},
		{"Client Version: v1.7.0-alpha.0.41+2e1271116023d6\nServer Version: v1.5.2-beta.0.41+2e1271116023d6", "v1.5.2"},
		{"Client Version: v1.7.0-alpha.0.41+2e1271116023d6\nServer Version: v1.5.0-dirty", "v1.5.0"},
	}

	for _, tt := range versionTests {
		v, _ := doGetKubeServerVersion(tt.in)
		if v != tt.out {
			t.Errorf("Didn't get correct server version (%q) from %q. Got: %q", tt.out, tt.in, v)
		}
	}
}
