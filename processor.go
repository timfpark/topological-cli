package main

type Processor struct {
	Config       map[string]string
	File         string
	Platform     string
	Dependencies map[string]string
}
