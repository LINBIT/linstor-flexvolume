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
