package main

type PlatformBuilder interface {
	BuildDeployment() (artifacts map[string]string, err error)
}
