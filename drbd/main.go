package main

import "fmt"
import "linbit/drbd-flexvolume/pkg/api"
import "os"

func main() {
	api, err := api.NewFlexVolumeAPI()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to get api: %v\n", err)
	}
	out, ret := api.Call(os.Args[1:])
	fmt.Print(out)
	os.Exit(ret)
}
