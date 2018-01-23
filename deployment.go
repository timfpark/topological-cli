package main

type Deployment struct {
	Instances   uint32
	Concurrency uint32
	Nodes       []string
	Replicas    Replicas
}
