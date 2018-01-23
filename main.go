package main

import (
	"fmt"
	"os"
)

func printHelp() {
	fmt.Println("usage: topo build <topology definition> <environment definition>: builds code and scripts for deployment and execution.")
	fmt.Println("")
	fmt.Println("example: topo build location-pipeline.json production.json")
}

func buildDeployment() {
	if len(os.Args) != 4 {
		printHelp()
		os.Exit(1)
	}

	builder := NewBuilder(os.Args[2], os.Args[3])
	err := builder.Build()
	if err != nil {
		fmt.Printf("building environment failed with error: %s\n", err)
	}
}

func printVersion() {
	fmt.Println("v1.0.0")
}

func main() {
	if len(os.Args) <= 1 {
		printHelp()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "build":
		buildDeployment()
	case "version":
		printVersion()
	case "help":
		printHelp()
	default:
		printHelp()
	}
}
