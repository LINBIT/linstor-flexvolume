package api

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mcuadros/go-version"
)

var flexAPIs = []FlexVolumeAPI{}

// FlexVolumeAPI recieves flexvolume calls, performs the
// action and returns a status message.
type FlexVolumeAPI interface {
	apiVersion() string
	// Parse and act on API calls from Kubernetes.
	Call([]string) (string, int)
}

// NewFlexVolumeAPI tries to return the most appropreate API based on the
// Kubernetes server version. If the server version can't be determined, the
// most recent API version and an error are returned.
func NewFlexVolumeAPI() (FlexVolumeAPI, error) {
	// Sort APIs from most to least recent.
	sort.Slice(flexAPIs, func(i, j int) bool {
		return version.Compare(version.Normalize(flexAPIs[i].apiVersion()),
			version.Normalize(flexAPIs[j].apiVersion()),
			">")
	})
	latestAPI := flexAPIs[0]
	kubeVersion, err := getKubeServerVersion()
	if err != nil {
		return latestAPI, fmt.Errorf("unable to determine Kubernetes server version: %v", err)
	}
	kubeVersion = version.Normalize(kubeVersion)

	// Return exact API match if we find one.
	for _, api := range flexAPIs {
		if version.Compare(version.Normalize(api.apiVersion()), kubeVersion, "=") {
			return api, nil
		}
	}

	// No exact matches, try to return lastest API that matches major and minor revisions.
	for _, api := range flexAPIs {
		if strings.HasSuffix(version.Normalize(api.apiVersion()), kubeVersion[:strings.LastIndex(kubeVersion, ".")]) {
			return api, nil
		}
	}
	// Generate list of api versions for final error message.
	versionList := []string{}
	for _, api := range flexAPIs {
		versionList = append(versionList, api.apiVersion())
	}

	return latestAPI, fmt.Errorf("unable to match major and minor versions of flexvolume API and Kuberbetes server version (%q), availible flex volume APIs: %v", kubeVersion, versionList)
}

func getKubeServerVersion() (string, error) {
	kubectl := "kubectl"
	// See if kubectl is in the path and try to fallback to a locally running cluster if not.
	if _, err := exec.Command("which", "kubectl").CombinedOutput(); err != nil {
		gopath := os.Getenv("GOPATH")
		if gopath == "" {
			gopath = filepath.Join(os.Getenv("HOME"), "/go/")
		}

		kubectl = filepath.Join(gopath, "/src/k8s.io/kubernetes/cluster/kubectl.sh")
		if _, err := os.Stat(kubectl); err != nil {
			return "", fmt.Errorf("Couldn't find kubectl in path or in GOPATH")
		}
	}

	out, err := exec.Command(kubectl, "version", "--short").CombinedOutput()
	if err != nil {
		return "", err
	}
	return doGetKubeServerVersion(string(out))
}

func doGetKubeServerVersion(s string) (string, error) {
	out := strings.Split(s, "\n")
	if len(out) != 2 {
		return "", fmt.Errorf("Unable to parse server version info from %v", out)
	}

	serverLine := out[1]

	serverPrefix := "Server Version: "

	if ok := strings.HasPrefix(serverLine, serverPrefix); !ok {
		return "", fmt.Errorf("Unexpected server line: %s", s)
	}

	longVersion := strings.TrimPrefix(serverLine, serverPrefix)

	return longVersion[:6], nil
}
