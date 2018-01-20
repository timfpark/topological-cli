package main

type Environment struct {
	Target        string
	Tier          string
	Namespace     string
	ContainerRepo string
	Connections   map[string]Connection
	Deployments   map[string]Deployment
}
