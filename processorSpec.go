package main

type ProcessorSpec struct {
	File         string
	Platform     string
	Dependencies map[string]string
}
