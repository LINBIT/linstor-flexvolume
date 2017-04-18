package main

import "fmt"
import "linbit/drbd-flexvolume/pkg/api"
import "os"

func main() {
	api := api.FlexVolumeApi{}
	out, ret := api.Call(os.Args[1:])
	fmt.Print(out)
	os.Exit(ret)
}
