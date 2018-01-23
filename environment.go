package main

type Environment struct {
	Target        string
	Tier          string
	Namespace     string
	ContainerRepo string
	PullSecret    string
	Connections   map[string]Connection
	Processors    map[string]ProcessorEnv
	Deployments   map[string]Deployment
}
