/*
* DRBD Flexvolume plugin for Kubernetes.
* Copyright © 2017 LINBIT USA LLC
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

package main

import (
	"fmt"
	"log"
	"log/syslog"
	"os"
	"strings"

	"github.com/linbit/drbd-flexvolume/pkg/api"
)

// Version is set via ldflags configued in the Makefile.
var Version string

func main() {

	apiCall := os.Args[1]

	// Print version and exit.
	if apiCall == "--version" {
		fmt.Println(Version)
		os.Exit(0)
	}

	sysLog, err := syslog.New(syslog.LOG_INFO, "DRBD FlexVolume")
	if err != nil {
		log.Fatal(err)
	}
	log.SetOutput(sysLog)

	log.Printf("called with %s: %s", apiCall, strings.Join(os.Args[2:], ", "))

	api := api.FlexVolumeApi{}

	out, ret := api.Call(os.Args[1:])

	log.Printf("responded to %s: %s", os.Args[1], out)

	fmt.Print(out)
	os.Exit(ret)
}
