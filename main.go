/*
* Linstor Flexvolume plugin for Kubernetes.
* Copyright © 2018 LINBIT USA LLC
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

	"github.com/LINBIT/linstor-flexvolume/pkg/api"
)

// Version is set via ldflags configued in the Makefile.
var Version string

func main() {

	if len(os.Args) == 1 {
		fmt.Errorf("Invalid number of parameters")
		os.Exit(1)
	}
	apiCall := os.Args[1]

	// Print version and exit.
	if apiCall == "--version" {
		fmt.Println(Version)
		os.Exit(0)
	}

	syslogOut, err := syslog.New(syslog.LOG_INFO, "Linstor FlexVolume")
	if err == nil {
		log.SetOutput(syslogOut)
	} else {
		fileOut, err := os.OpenFile("/tmp/linstor_flex", os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
		if err != nil {
			log.Fatal(err)
		}
		defer fileOut.Close()
		log.SetOutput(fileOut)
	}

	log.Printf("called with %s: %s", apiCall, strings.Join(os.Args[2:], ", "))

	api := api.FlexVolumeApi{}

	out, ret := api.Call(os.Args[1:])

	log.Printf("responded to %s: %s", os.Args[1], out)

	fmt.Print(out)
	os.Exit(ret)
}
