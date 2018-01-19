package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

const expectedWriteLocationsPackageJson = `{
    "name": "stage-write-locations",
    "version": "1.0.0",
    "main": "stage.js",
    "scripts": {
        "start": "node stage.js"
    },
    "dependencies": {
        "express": "^4.16.2",
        "prom-client": "^10.2.2",
        "request": "^2.83.0",
        "topological": "^1.0.28",
        "topological-kafka":"^1.0.4",
        "cassandra-driver":"^3.3.0"
    }
}`

const expectedPackageJson = `{
    "name": "stage-predict-arrivals",
    "version": "1.0.0",
    "main": "stage.js",
    "scripts": {
        "start": "node stage.js"
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
    locationsConnectionClass = require('topological-kafka'),
    predictArrivalsProcessorClass = require('./processors/predictArrivals.js');`

const expectedConnectionsString = `let estimatedArrivalsConnection = new estimatedArrivalsConnectionClass({
    "id": "estimatedArrivals",
    "config": {"endpoint":"kafka-zookeeper.kafka.svc.cluster.local:2181","keyField":"busId","topic":"estimated-arrivals"}
});

let locationsConnection = new locationsConnectionClass({
    "id": "locations",
    "config": {"endpoint":"kafka-zookeeper.kafka.svc.cluster.local:2181","keyField":"busId","topic":"locations"}
});`

const expectedProcessorsString = `let predictArrivalsProcessor = new predictArrivalsProcessorClass({
    "id": "predictArrivals",
    "config": {}
});`

const expectedNodesString = `new Node({
            id: 'predictArrivals',
            inputs: [locationsConnection],
            processor: predictArrivalsProcessor,
            outputs: [estimatedArrivalsConnection]
        })`

const expectedTopologyString = `let topology = new Topology({
    id: 'topology',
    nodes: [
        new Node({
            id: 'predictArrivals',
            inputs: [locationsConnection],
            processor: predictArrivalsProcessor,
            outputs: [estimatedArrivalsConnection]
        })
    ]
});

topology.start(err => {
    if (err) {
        topology.log.error("topology start failed with: " + err);
        return process.exit(0);
    }
});
`

const expectedStageJs = `const { Node, Topology } = require('topological'),
    express = require('express'),
    app = express(),
    server = require('http').createServer(app),
    promClient = require('prom-client'),
    estimatedArrivalsConnectionClass = require('topological-kafka'),
    locationsConnectionClass = require('topological-kafka'),
    predictArrivalsProcessorClass = require('./processors/predictArrivals.js');

let estimatedArrivalsConnection = new estimatedArrivalsConnectionClass({
    "id": "estimatedArrivals",
    "config": {"endpoint":"kafka-zookeeper.kafka.svc.cluster.local:2181","keyField":"busId","topic":"estimated-arrivals"}
});

let locationsConnection = new locationsConnectionClass({
    "id": "locations",
    "config": {"endpoint":"kafka-zookeeper.kafka.svc.cluster.local:2181","keyField":"busId","topic":"locations"}
});

let predictArrivalsProcessor = new predictArrivalsProcessorClass({
    "id": "predictArrivals",
    "config": {}
});

let topology = new Topology({
    id: 'topology',
    nodes: [
        new Node({
            id: 'predictArrivals',
            inputs: [locationsConnection],
            processor: predictArrivalsProcessor,
            outputs: [estimatedArrivalsConnection]
        })
    ]
});

topology.start(err => {
    if (err) {
        topology.log.error("topology start failed with: " + err);
        return process.exit(0);
    }
});
`

func TestFillPackageJson(t *testing.T) {
	builder := NewBuilder("fixtures/topology.json", "fixtures/environment.json")
	err := builder.Load()
	if err != nil {
		t.Errorf("builder failed to load: %s", err)
	}

	deploymentID := "write-locations"
	deployment := builder.Environment.Deployments[deploymentID]

	nodeJsBuilder := NodeJsPlatformBuilder{
		DeploymentID: deploymentID,
		Deployment:   deployment,
		Topology:     builder.Topology,
		Environment:  builder.Environment,
	}

	packageJson := nodeJsBuilder.FillPackageJson()

	if packageJson != expectedWriteLocationsPackageJson {
		t.Errorf("package.json did not match:-->%s<-- vs. -->%s<-- did not complete successfully.", packageJson, expectedWriteLocationsPackageJson)
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

func TestFillConnections(t *testing.T) {
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

	connectionsString := nodeJsBuilder.FillConnections()
	if connectionsString != expectedConnectionsString {
		t.Errorf("connections did not match:-->%s<-- vs. -->%s<-- did not complete successfully.", connectionsString, expectedConnectionsString)
	}
}

func TestFillProcessors(t *testing.T) {
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

	processorsString := nodeJsBuilder.FillProcessors()
	if processorsString != expectedProcessorsString {
		t.Errorf("processors did not match:-->%s<-- vs. -->%s<-- did not complete successfully.", processorsString, expectedProcessorsString)
	}
}

func TestFillNodes(t *testing.T) {
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

	nodesString := nodeJsBuilder.FillNodes()
	if nodesString != expectedNodesString {
		t.Errorf("nodes did not match:-->%s<-- vs. -->%s<-- did not complete successfully.", nodesString, expectedNodesString)
	}
}

func TestFillTopology(t *testing.T) {
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

	topologyString := nodeJsBuilder.FillTopology()
	if topologyString != expectedTopologyString {
		t.Errorf("topology did not match:-->%s<-- vs. -->%s<-- did not complete successfully.", topologyString, expectedTopologyString)
	}
}

func TestFillStage(t *testing.T) {
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

	stageJsString := nodeJsBuilder.FillStage()

	if stageJsString != expectedStageJs {
		t.Errorf("stage.js did not match:-->%s<-- vs. -->%s<-- did not complete successfully.", stageJsString, expectedStageJs)
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

	expectedItems := []string{
		"build",
		"build/notify-arrivals",
		"build/notify-arrivals/code",
		"build/notify-arrivals/code/package.json",
		"build/notify-arrivals/code/stage.js",
		"build/write-locations",
		"build/write-locations/code",
		"build/write-locations/code/package.json",
		"build/write-locations/code/stage.js",
		"build/predict-arrivals",
		"build/predict-arrivals/code",
		"build/predict-arrivals/code/package.json",
		"build/predict-arrivals/code/stage.js",
	}

	for _, directory := range expectedItems {
		if _, err := os.Stat(directory); os.IsNotExist(err) {
			t.Errorf("Build did not created expected directory: %s", directory)
		}
	}

	packageJsonBytes, err := ioutil.ReadFile("build/predict-arrivals/code/package.json")
	if err != nil {
		t.Errorf("Could not read package.json: %s", err)
	}

	// TODO: how does one really convert a []byte array to string
	if fmt.Sprintf("%s", packageJsonBytes) != expectedPackageJson {
		t.Errorf("package.json did not match:-->%s<-- vs. -->%s<-- did not complete successfully.", packageJsonBytes, expectedPackageJson)
	}

	stageJsBytes, err := ioutil.ReadFile("build/predict-arrivals/code/stage.js")
	if err != nil {
		t.Errorf("Could not read stage.js: %s", err)
	}

	// TODO: how does one really convert a []byte array to string
	if fmt.Sprintf("%s", stageJsBytes) != expectedStageJs {
		t.Errorf("stage.js did not match:-->%s<-- vs. -->%s<-- did not complete successfully.", stageJsBytes, expectedStageJs)
	}
}
