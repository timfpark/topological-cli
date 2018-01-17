package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

type Builder struct {
	TopologyPath    string
	EnvironmentPath string

	Topology    Topology
	Environment Environment
}

func NewBuilder(topologyPath string, environmentPath string) *Builder {
	b := new(Builder)
	b.TopologyPath = topologyPath
	b.EnvironmentPath = environmentPath
	return b
}

func (b *Builder) LoadTopology() (topology *Topology, err error) {
	topologyContents, err := ioutil.ReadFile(b.TopologyPath)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(topologyContents, &b.Topology)
	if err != nil {
		return nil, err
	}

	return &b.Topology, nil
}

func (b *Builder) LoadEnvironment() (environment *Environment, err error) {
	environmentContents, err := ioutil.ReadFile(b.EnvironmentPath)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(environmentContents, &b.Environment)
	if err != nil {
		return nil, err
	}

	return &b.Environment, nil
}

func (b *Builder) Build() bool {
	_, err := b.LoadTopology()
	if err != nil {
		return false
	}

	_, err = b.LoadEnvironment()
	if err != nil {
		return false
	}
	os.Mkdir("build", 0755)

	return true
	/*
		fmt.Printf("topology => %+v\n", b.Topology)

		async.eachSeries(Object.keys(this.environment.deployments), (deploymentId, deploymentCallback) => {
			this.buildDeployment(deploymentId, deploymentCallback);
		}, callback);
	*/
}
