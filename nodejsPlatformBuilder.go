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

func (b *NodeJsPlatformBuilder) FillImports() (imports string) {
	connections := map[string]bool{}

	for _, nodeId := range b.Deployment.Nodes {
		node := b.Topology.Nodes[nodeId]
		for _, connectionId := range node.Inputs {
			connections[connectionId] = true
		}
		for _, connectionId := range node.Outputs {
			connections[connectionId] = true
		}
	}

	connectionImports := []string{}
	for connectionId, _ := range connections {
		connection := b.Environment.Connections[connectionId]
		for packageName, _ := range connection.Dependencies {
			importString := fmt.Sprintf(`    %sConnectionClass = require('%s')`, connectionId, packageName)
			connectionImports = append(connectionImports, importString)
		}
	}

	sort.Strings(connectionImports)

	return fmt.Sprintf(`const { Node, Topology } = require('topological'),
    express = require('express'),
    app = express(),
    server = require('http').createServer(app),
    promClient = require('prom-client'),
%s;`, strings.Join(connectionImports, ",\n"))
}

func (b *NodeJsPlatformBuilder) FillConnections() (connectionInstantiations string) {
	connections := map[string]bool{}

	for _, nodeId := range b.Deployment.Nodes {
		node := b.Topology.Nodes[nodeId]
		for _, connectionId := range node.Inputs {
			connections[connectionId] = true
		}
		for _, connectionId := range node.Outputs {
			connections[connectionId] = true
		}
	}

	instantiations := []string{}

	for connectionId, _ := range connections {
		connection := b.Environment.Connections[connectionId]
		mapJson, _ := json.Marshal(connection.Config)
		connectionInstantiation := fmt.Sprintf(`let %sConnection = new %sConnectionClass({
    "id": "%s",
    "config": %s
});`, connectionId, connectionId, connectionId, mapJson)
		instantiations = append(instantiations, connectionInstantiation)
	}

	sort.Strings(instantiations)

	return strings.Join(instantiations, "\n\n")
}

func (b *NodeJsPlatformBuilder) FillStage() (stage string) {
	imports := b.FillImports()
	connections := b.FillConnections()

	return fmt.Sprintf("%s\n\n%s", imports, connections)
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
