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
	"linbit/drbd-flexvolume/pkg/api"
	"log"
	"os"
	"strings"
)

// Version is set via ldflags configued in the Makefile.
var Version string

func main() {
	api := api.FlexVolumeApi{}

	// Print version and exit.
	if os.Args[1] == "--version" {
		fmt.Println(Version)
		os.Exit(0)
	}

	f, err := os.OpenFile("/tmp/drbdflex.log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	f.WriteString("called with: " + strings.Join(os.Args[1:], ", ") + "\n")

	out, ret := api.Call(os.Args[1:])
	f.WriteString("responded: " + out + "\n\n")

	fmt.Print(out)
	os.Exit(ret)
}
