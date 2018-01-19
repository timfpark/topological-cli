package main

import (
	"encoding/json"
	"errors"
	"fmt"
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
	return &Builder{
		TopologyPath:    topologyPath,
		EnvironmentPath: environmentPath,
	}
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

func (b *Builder) BuildDeployment(deploymentID string) (artifacts map[string]string, err error) {
	var platform string

	deployment := b.Environment.Deployments[deploymentID]

	// check to make sure platform is the same across the nodes of the deployment
	for nodeIdx, _ := range deployment.Nodes {
		nodeId := deployment.Nodes[nodeIdx]
		node := b.Topology.Nodes[nodeId]

		if platform != "" && node.Processor.Platform != platform {
			errString := fmt.Sprintf("mismatched platforms: %s vs %s for deployment id %s", platform, node.Processor.Platform, deploymentID)
			return nil, errors.New(errString)
		} else {
			platform = node.Processor.Platform
		}
	}

	var platformBuilder PlatformBuilder
	switch platform {
	case "node.js":
		platformBuilder = &NodeJsPlatformBuilder{
			Deployment:  deployment,
			Topology:    b.Topology,
			Environment: b.Environment,
		}
	default:
		errString := fmt.Sprintf("unknown platform %s", platform)
		return nil, errors.New(errString)
	}

	return platformBuilder.BuildDeployment()
}

func (b *Builder) Build() error {
	_, err := b.LoadTopology()
	if err != nil {
		return err
	}

	_, err = b.LoadEnvironment()
	if err != nil {
		return err
	}

	err = os.Mkdir("build", 0755)
	if err != nil {
		return err
	}

	for deploymentID := range b.Environment.Deployments {
		_, err = b.BuildDeployment(deploymentID)
		if err != nil {
			return err
		}
	}

	return nil
	/*
		fmt.Printf("topology => %+v\n", b.Topology)

		async.eachSeries(Object.keys(this.environment.deployments), (deploymentId, deploymentCallback) => {
			this.buildDeployment(deploymentId, deploymentCallback);
		}, callback);
	*/
}
