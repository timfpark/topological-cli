package main

type Connection struct {
	Platform     string
	Dependencies map[string]string
	Config       map[string]interface{}
}
