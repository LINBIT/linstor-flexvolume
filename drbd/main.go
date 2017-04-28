package main

import (
	"fmt"
	"linbit/drbd-flexvolume/pkg/api"
	"log"
	"os"
	"strings"
)

func main() {
	api := api.FlexVolumeApi{}

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
