package main

import (
	"fmt"
	"os"
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

func (b *NodeJsPlatformBuilder) FillPackageJson() (artifact string) {
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

func (b *NodeJsPlatformBuilder) BuildDeployment() (err error) {
	b.DeploymentPath = fmt.Sprintf("build/%s", b.DeploymentID)
	err = os.Mkdir(b.DeploymentPath, 0755)
	if err != nil {
		return err
	}

	b.CodePath = fmt.Sprintf("%s/code", b.DeploymentPath)
	err = os.Mkdir(b.CodePath, 0755)
	if err != nil {
		return err
	}

	return nil

	//b.CreateStageJS()

	// create directory for code (./build/{deploymentId}/stage)
	// 		create package.json
	// 		create stage.js
	// write common deployment elements ./build/common
	// place deployment deps in ./build/{deployment}
	// 		laydown deploy-stage
	// 		laydown Dockerfile
	// 		laydown start-service
	// 		laydown values.yaml
}
