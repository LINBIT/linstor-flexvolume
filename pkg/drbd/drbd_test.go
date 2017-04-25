/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package drbd

import "testing"

func TestDoGetDevPath(t *testing.T) {
	var volumeStringTests = []struct {
		in  string
		out string
	}{
		{"test0,,0,102400,7001,130,\n", "/dev/drbd130"},
		{"test1,,0,102400,7002,131,\n", "/dev/drbd131"},
		{"test2,,0,102400,2003,132,\ntest3,,0,102400,2004,133,\n", ""},
	}

	for _, tt := range volumeStringTests {
		dev, _ := doGetDevPath(tt.in)
		if dev != tt.out {
			t.Errorf("Called: doGetDevPath(%q), Expected: %q, Got: %q", tt.in, tt.out, dev)
		}
	}
}

func TestDoResExists(t *testing.T) {
	var resInfoTests = []struct {
		resource string
		resInfo  string
		out      bool
	}{
		{"test0", "test0,7001,\n", true},
		{"test1", "test1,7002,\n", true},
		{"test2", "test9,7003,\n", false},
		{"test3", "", false},
	}

	for _, tt := range resInfoTests {
		ok, _ := doResExists(tt.resource, tt.resInfo)
		if ok != tt.out {
			t.Errorf("Called: doResExists(%q, %q), Expected: %v, Got: %v", tt.resource, tt.resInfo, tt.out, ok)
		}
	}
}

func TestDoResAssigned(t *testing.T) {
	var resAssignmentTests = []struct {
		assignmentInfo string
		out            bool
	}{
		{"node0,test0,0,connect|deploy,connect|deploy\n", true},
		{"node1,test1,0,connect|deploy|diskless,connect|deploy|diskless\n", true},
		{"", false},
		{"node0,test0,0,,connect|deploy\n", false},
	}

	for _, tt := range resAssignmentTests {
		ok, _ := doResAssigned(tt.assignmentInfo)
		if ok != tt.out {
			t.Errorf("Called: doResAssigned(%q), Expected: %v, Got: %v", tt.assignmentInfo, tt.out, ok)
		}
	}
}

func TestDoIsClient(t *testing.T) {
	var isClientTests = []struct {
		assignmentInfo string
		out            bool
	}{
		{"node0,test0,1,connect|deploy|diskless,connect|deploy|diskless\n", true},
		{"bielefeld,test1,1,connect|deploy,connect|deploy\n", false},
		{"", false},
	}

	for _, tt := range isClientTests {
		ok := doIsClient(tt.assignmentInfo)
		if ok != tt.out {
			t.Errorf("Called: doIsClient(%q), Expected: %v, Got: %v", tt.assignmentInfo, tt.out, ok)
		}
	}
}

func TestDoGetMinorFromDevice(t *testing.T) {
	var getMinorTests = []struct {
		device string
		out    string
	}{
		{"/dev/drbd100", "100"},
		{"/dev/drbd5123", "5123"},
		{"/dev/drbd0", "0"},
		{"/dev/sda1", ""},
	}

	for _, tt := range getMinorTests {
		minor, _ := getMinorFromDevice(tt.device)
		if minor != tt.out {
			t.Errorf("Called: getMinorFromDevice(%q), Expected: %v, Got: %v", tt.device, tt.out, minor)
		}
	}
}

func TestGetResFromVolumes(t *testing.T) {
	var getResFromVolumesTests = []struct {
		volumes string
		minor   string
		out     string
	}{
		{"test0,,0,102400,7000,100,\n", "100", "test0"},
		{"test0,,0,102400,7000,100,\ntest1,,0,102400,7001,101,\ntest2,,0,2097152,7002,102,\n", "101", "test1"},
		{"test0,,0,102400,7000,100,\ntest1,,0,102400,7001,101,\ntest2,,0,2097152,7002,102,\n", "7001", ""},
		{"", "104", ""},
	}

	for _, tt := range getResFromVolumesTests {
		res, _ := getResFromVolumes(tt.volumes, tt.minor)
		if res != tt.out {
			t.Errorf("Called: getResFromVolumes(%q, %q) Expected: %v, Got: %v", tt.volumes, tt.minor, tt.out, res)
		}
	}
}
