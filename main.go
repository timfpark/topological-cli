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
	err := builder.Build()
	if err != nil {
		fmt.Printf("building environment failed with error: %s\n", err)
		os.Exit(1)
	}
}
