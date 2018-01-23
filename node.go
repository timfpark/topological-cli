package main

type Node struct {
	Inputs    []string
	Processor ProcessorSpec
	Outputs   []string
}
