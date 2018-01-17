package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Printf("usage: topo topology.json environment.json\n")
		os.Exit(1)
	}

	builder := NewBuilder(os.Args[1], os.Args[2])
	if !builder.Build() {
		fmt.Printf("building environment failed.\n")
		os.Exit(1)
	}
}
