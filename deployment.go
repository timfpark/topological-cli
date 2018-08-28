package main

type Deployment struct {
	Instances   uint32
	Concurrency uint32
	CPU         CPUSpec
	LogSeverity string
	Memory      MemorySpec
	Nodes       []string
	Replicas    ReplicaSpec
}
