package main

type Node struct {
	Inputs    []string
	Processor Processor
	Outputs   []string
}
