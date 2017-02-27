package api

import (
	"sort"
	"testing"

	version "github.com/mcuadros/go-version"
)

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

type oldAPI struct{}

func (o oldAPI) apiVersion() string {
	return "v1.5.2"
}
func (o oldAPI) Call(string) string {
	return ""
}

type newAPI struct{}

func (n newAPI) apiVersion() string {
	return "v1.6.1"
}
func (n newAPI) Call(string) string {
	return ""
}

type medAPI struct{}

func (m medAPI) apiVersion() string {
	return "v1.5.6"
}

func (m medAPI) Call(string) string {
	return ""
}

func TestByLatestVersion(t *testing.T) {

	var apis = []FlexVolumeAPI{medAPI{}, oldAPI{}, newAPI{}}

	// Redefining the function here to test, this could be better.
	sort.Slice(apis, func(i, j int) bool {
		return version.Compare(version.Normalize(apis[i].apiVersion()),
			version.Normalize(apis[j].apiVersion()),
			">")
	})
	// Sorts Apis by greatest to least.
	if apis[0].apiVersion() != "v1.6.1" {
		t.Errorf("LatestVerison: Newest api not first!")
	}
	if apis[1].apiVersion() != "v1.5.6" {
		t.Errorf("LatestVerison: middle api not in the middle!")
	}
	if apis[2].apiVersion() != "v1.5.2" {
		t.Errorf("LatestVerison: oldest api not last!")
	}
}
