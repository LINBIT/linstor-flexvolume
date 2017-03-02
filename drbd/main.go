package main

import "fmt"
import "linbit/drbd-flexvolume/pkg/api"
import "os"

func main() {
	api, _ := api.NewFlexVolumeAPI()
	out, ret := api.Call(os.Args[1:])
	fmt.Print(out)
	os.Exit(ret)
}
