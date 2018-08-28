package main

type PlatformBuilder interface {
	BuildSource() (err error)
}
