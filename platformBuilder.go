package main

type PlatformBuilder interface {
	BuildDeployment() (err error)
}
