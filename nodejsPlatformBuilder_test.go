package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

const expectedPackageJson = `{
    "name": "stage-predict-arrivals",
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
        "topological-kafka":"^1.0.4"
    }
}`

const expectedImports = `const { Node, Topology } = require('topological'),
    express = require('express'),
    app = express(),
    server = require('http').createServer(app),
    promClient = require('prom-client'),
    estimatedArrivalsConnectionClass = require('topological-kafka'),
    locationsConnectionClass = require('topological-kafka');`

func TestFillPackageJson(t *testing.T) {
	builder := NewBuilder("fixtures/topology.json", "fixtures/environment.json")
	err := builder.Load()
	if err != nil {
		t.Errorf("builder failed to load: %s", err)
	}

	deploymentID := "predict-arrivals"
	deployment := builder.Environment.Deployments[deploymentID]

	nodeJsBuilder := NodeJsPlatformBuilder{
		DeploymentID: deploymentID,
		Deployment:   deployment,
		Topology:     builder.Topology,
		Environment:  builder.Environment,
	}

	packageJson := nodeJsBuilder.FillPackageJson()

	if packageJson != expectedPackageJson {
		t.Errorf("package.json did not match:-->%s<-- vs. -->%s<-- did not complete successfully.", packageJson, expectedPackageJson)
	}
}

func TestFillImports(t *testing.T) {
	builder := NewBuilder("fixtures/topology.json", "fixtures/environment.json")
	err := builder.Load()
	if err != nil {
		t.Errorf("builder failed to load: %s", err)
	}

	deploymentID := "predict-arrivals"
	deployment := builder.Environment.Deployments[deploymentID]

	nodeJsBuilder := NodeJsPlatformBuilder{
		DeploymentID: deploymentID,
		Deployment:   deployment,
		Topology:     builder.Topology,
		Environment:  builder.Environment,
	}

	importsString := nodeJsBuilder.FillImports()
	if importsString != expectedImports {
		t.Errorf("imports did not match:-->%s<-- vs. -->%s<-- did not complete successfully.", importsString, expectedImports)
	}
}

func TestBuild(t *testing.T) {
	builder := NewBuilder("fixtures/topology.json", "fixtures/environment.json")

	_, err := builder.LoadTopology()
	if err != nil {
		t.Errorf("LoadTopology did not complete successfully.")
	}

	_, err = builder.LoadEnvironment()
	if err != nil {
		t.Errorf("LoadEnvironment did not complete successfully.")
	}

	err = builder.Build()
	if err != nil {
		t.Errorf("Build did not complete successfully: %s", err)
	}

	expectedDirectories := []string{
		"build",
		"build/notify-arrivals",
		"build/notify-arrivals/code",
		"build/write-locations",
		"build/write-locations/code",
		"build/predict-arrivals",
		"build/predict-arrivals/code",
	}

	for _, directory := range expectedDirectories {
		if _, err := os.Stat(directory); os.IsNotExist(err) {
			t.Errorf("Build did not created expected directory: %s", directory)
		}
	}

	packageJsonBytes, err := ioutil.ReadFile("build/predict-arrivals/code/package.json")
	if err != nil {
		t.Errorf("Build did not complete successfully: %s", err)
	}

	// TODO: how does one really convert a []byte array to string
	if fmt.Sprintf("%s", packageJsonBytes) != expectedPackageJson {
		t.Errorf("package.json did not match:-->%s<-- vs. -->%s<-- did not complete successfully.", packageJsonBytes, expectedPackageJson)
	}
}
