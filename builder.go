package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
)

type Builder struct {
	TopologyPath    string
	EnvironmentPath string

	DeploymentPath string

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
		err = errors.New(fmt.Sprintf("Topology failed to unmarshal: %s", err))
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
		err = errors.New(fmt.Sprintf("Environment failed to unmarshal: %s", err))
		return nil, err
	}

	return &b.Environment, nil
}

func (b *Builder) Load() (err error) {
	_, err = b.LoadEnvironment()
	if err != nil {
		return nil
	}

	_, err = b.LoadTopology()
	return err
}

func (b *Builder) MakeBuilder(deploymentID string) (platformBuilder PlatformBuilder, err error) {
	var platform string

	deployment := b.Environment.Deployments[deploymentID]

	// check to make sure platform is the same across the nodes of the deployment
	for nodeIdx, _ := range deployment.Nodes {
		nodeId := deployment.Nodes[nodeIdx]
		node, nodeExists := b.Topology.Nodes[nodeId]

		if !nodeExists {
			errString := fmt.Sprintf("no node named %s as found in deployment %s", nodeId, deploymentID)
			return nil, errors.New(errString)
		}

		if platform != "" && node.Processor.Platform != platform {
			errString := fmt.Sprintf("mismatched platforms: %s vs %s for deployment id %s", platform, node.Processor.Platform, deploymentID)
			return nil, errors.New(errString)
		} else {
			platform = node.Processor.Platform
		}
	}

	switch platform {
	case "node.js":
		platformBuilder = &NodeJsPlatformBuilder{
			DeploymentID: deploymentID,
			Deployment:   deployment,
			Topology:     b.Topology,
			Environment:  b.Environment,
		}
	default:
		errString := fmt.Sprintf("unknown platform %s", platform)
		return nil, errors.New(errString)
	}

	return platformBuilder, nil
}

func (b *Builder) BuildDeployment(deploymentID string) (err error) {
	platformBuilder, err := b.MakeBuilder(deploymentID)
	if err != nil {
		return err
	}

	b.DeploymentPath = path.Join("build", b.Environment.Tier, deploymentID)
	err = os.Mkdir(b.DeploymentPath, 0755)
	if err != nil {
		return err
	}

	err = platformBuilder.BuildSource()
	if err != nil {
		return err
	}

	return nil
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

	// create build directory if it doesn't exist
	os.Mkdir("build", 0755)

	tierDir := path.Join("build", b.Environment.Tier)

	// if it exists, remove old build for this tier
	os.RemoveAll(tierDir)

	err = os.Mkdir(tierDir, 0755)
	if err != nil {
		return err
	}

	var deployAllScript string
	for deploymentID := range b.Environment.Deployments {
		err = b.BuildDeployment(deploymentID)
		if err != nil {
			return err
		}

		deployAllScript += fmt.Sprintf("cd %s && ./deploy-stage && cd ..\n", deploymentID)
	}

	ioutil.WriteFile(path.Join(tierDir, "deploy-all"), []byte(deployAllScript), 0755)

	return nil
}
