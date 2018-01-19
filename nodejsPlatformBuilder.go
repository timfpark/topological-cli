package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

type NodeJsPlatformBuilder struct {
	DeploymentID string
	Deployment   Deployment
	Topology     Topology
	Environment  Environment

	DeploymentPath string
	CodePath       string
}

func (b *NodeJsPlatformBuilder) collectDependencies() (dependencies map[string]string) {
	dependencies = map[string]string{}
	for _, connection := range b.Environment.Connections {
		for packageName, version := range connection.Dependencies {
			dependencies[packageName] = version
		}
	}

	return dependencies
}

func (b *NodeJsPlatformBuilder) FillPackageJson() (packageJson string) {
	dependencies := b.collectDependencies()
	var dependencyStrings []string
	for packageName, version := range dependencies {
		dependencyStrings = append(dependencyStrings, fmt.Sprintf(`"%s":"%s"`, packageName, version))
	}

	return fmt.Sprintf(`{
    "name": "stage-%s",
    "version": "1.0.0",
    "main": "stage.js",
    "scripts": {
        "start": "node stage.js",
    },
    "dependencies": {
        "express": "^4.16.2",
        "prom-client": "^10.2.2",
        "request": "^2.83.0",
        "topological": "^1.0.28",
        %s
    }
}`,
		b.DeploymentID,
		strings.Join(dependencyStrings, "\n"))
}

func (b *NodeJsPlatformBuilder) consolidateDeploymentConnections() (connections map[string]bool) {
	connections = map[string]bool{}

	for _, nodeId := range b.Deployment.Nodes {
		node := b.Topology.Nodes[nodeId]
		for _, connectionId := range node.Inputs {
			connections[connectionId] = true
		}
		for _, connectionId := range node.Outputs {
			connections[connectionId] = true
		}
	}

	return connections
}

func (b *NodeJsPlatformBuilder) FillImports() (imports string) {
	connections := b.consolidateDeploymentConnections()

	connectionImports := []string{}
	for connectionId, _ := range connections {
		connection := b.Environment.Connections[connectionId]
		for packageName, _ := range connection.Dependencies {
			importString := fmt.Sprintf(`    %sConnectionClass = require('%s')`, connectionId, packageName)
			connectionImports = append(connectionImports, importString)
		}
	}

	sort.Strings(connectionImports)

	processorImports := []string{}

	for _, nodeId := range b.Deployment.Nodes {
		node := b.Topology.Nodes[nodeId]
		importString := fmt.Sprintf(`    %sProcessorClass = require('%s')`, nodeId, node.Processor.File)
		processorImports = append(processorImports, importString)
	}

	return fmt.Sprintf(`const { Node, Topology } = require('topological'),
    express = require('express'),
    app = express(),
    server = require('http').createServer(app),
    promClient = require('prom-client'),
%s,
%s;`, strings.Join(connectionImports, ",\n"), strings.Join(processorImports, ",\n"))
}

func (b *NodeJsPlatformBuilder) FillConnections() (connectionInstantiations string) {
	connections := b.consolidateDeploymentConnections()

	instantiations := []string{}

	for connectionId, _ := range connections {
		connection := b.Environment.Connections[connectionId]
		connectionConfigJson, _ := json.Marshal(connection.Config)
		connectionInstantiation := fmt.Sprintf(`let %sConnection = new %sConnectionClass({
    "id": "%s",
    "config": %s
});`, connectionId, connectionId, connectionId, connectionConfigJson)
		instantiations = append(instantiations, connectionInstantiation)
	}

	sort.Strings(instantiations)

	return strings.Join(instantiations, "\n\n")
}

func (b *NodeJsPlatformBuilder) FillProcessors() (processorInstantiations string) {
	instantiations := []string{}

	for _, nodeId := range b.Deployment.Nodes {
		node := b.Topology.Nodes[nodeId]
		processorConfigJson, _ := json.Marshal(node.Processor.Config)
		processorInstantiation := fmt.Sprintf(`let %sProcessor = new %sProcessorClass({
    "id": "%s",
    "config": %s
});`, nodeId, nodeId, nodeId, processorConfigJson)
		instantiations = append(instantiations, processorInstantiation)
	}

	sort.Strings(instantiations)

	return strings.Join(instantiations, "\n\n")
}

func buildConnectionInstanceNamesFromIds(connectionIds []string) []string {
	instances := []string{}
	for _, connectionId := range connectionIds {
		instances = append(instances, fmt.Sprintf("%sConnection", connectionId))
	}

	return instances
}

func (b *NodeJsPlatformBuilder) FillNodes() (nodeInstantiations string) {
	instances := []string{}

	for _, nodeId := range b.Deployment.Nodes {
		node := b.Topology.Nodes[nodeId]

		inputConnectionInstances := buildConnectionInstanceNamesFromIds(node.Inputs)
		outputConnectionInstances := buildConnectionInstanceNamesFromIds(node.Outputs)

		nodeInstantiation := fmt.Sprintf(`new Node({
            id: '%s',
            inputs: [%s],
            processor: %sProcessor,
            outputs: [%s]
        })`, nodeId, strings.Join(inputConnectionInstances, ","), nodeId, strings.Join(outputConnectionInstances, ","))

		instances = append(instances, nodeInstantiation)
	}

	return strings.Join(instances, ",\n")
}

func (b *NodeJsPlatformBuilder) FillTopology() (topologyInstantiation string) {
	nodesString := b.FillNodes()

	return fmt.Sprintf(`let topology = new Topology({
    id: 'topology',
    nodes: [
        %s
    ]
});

topology.start(err => {
    if (err) {
        topology.log.error("topology start failed with: " + err);
        return process.exit(0);
    }
});
`, nodesString)
}

func (b *NodeJsPlatformBuilder) FillStage() (stage string) {
	imports := b.FillImports()
	connections := b.FillConnections()
	processors := b.FillProcessors()
	topology := b.FillTopology()

	return fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s", imports, connections, processors, topology)
}

func (b *NodeJsPlatformBuilder) BuildDeployment() (err error) {
	b.DeploymentPath = fmt.Sprintf("build/%s", b.DeploymentID)
	err = os.Mkdir(b.DeploymentPath, 0755)
	if err != nil {
		return err
	}

	// create directory for code (./build/{deploymentId}/code)
	b.CodePath = fmt.Sprintf("%s/code", b.DeploymentPath)
	err = os.Mkdir(b.CodePath, 0755)
	if err != nil {
		return err
	}

	// create package.json
	packageJsonPath := fmt.Sprintf("%s/package.json", b.CodePath)
	packageJsonFile, err := os.OpenFile(packageJsonPath, os.O_RDWR|os.O_CREATE, 0744)
	if err != nil {
		return err
	}

	_, err = packageJsonFile.WriteString(b.FillPackageJson())
	packageJsonFile.Close()

	// create stage.js
	stagePath := fmt.Sprintf("%s/stage.js", b.CodePath)
	stageFile, err := os.OpenFile(stagePath, os.O_RDWR|os.O_CREATE, 0744)
	if err != nil {
		return err
	}

	_, err = stageFile.WriteString(b.FillStage())
	stageFile.Close()

	// write common deployment elements ./build/common
	// place deployment deps in ./build/{deployment}
	// 		laydown deploy-stage
	// 		laydown Dockerfile
	// 		laydown start-service
	// 		laydown values.yaml

	return err

}
