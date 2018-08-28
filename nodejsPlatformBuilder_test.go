package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

const expectedWriteLocationsPackageJson = `{
    "name": "write-locations",
    "version": "1.0.0",
    "main": "stage.js",
    "scripts": {
        "start": "node stage.js"
    },
    "dependencies": {
        "express": "^4.16.2",
        "morgan": "^1.9.0",
        "prom-client": "^11.0.0",
        "request": "^2.83.0",
        "topological": "^1.0.32",
        "cassandra-driver":"^3.3.0",
        "topological-kafka":"^1.0.4"
    }
}`

const expectedPackageJson = `{
    "name": "predict-arrivals",
    "version": "1.0.0",
    "main": "stage.js",
    "scripts": {
        "start": "node stage.js"
    },
    "dependencies": {
        "express": "^4.16.2",
        "morgan": "^1.9.0",
        "prom-client": "^11.0.0",
        "request": "^2.83.0",
        "topological": "^1.0.32",
        "topological-kafka":"^1.0.4"
    }
}`

const expectedImports = `const { Node, Topology } = require('topological'),
    express = require('express'),
    app = express(),
    morgan = require('morgan'),
    server = require('http').createServer(app),
    promClient = require('prom-client'),
    estimatedArrivalsConnectionClass = require('topological-kafka'),
    locationsConnectionClass = require('topological-kafka'),
    predictArrivalsProcessorClass = require('./processors/predictArrivals.js');`

const expectedConnectionsString = `let estimatedArrivalsConnection = new estimatedArrivalsConnectionClass({
    "id": "estimatedArrivals",
    "config": {"endpoint": process.env.KAFKA_ENDPOINT, "keyField": process.env.ESTIMATED_ARRIVALS_KEYFIELD, "topic": process.env.ESTIMATED_ARRIVALS_TOPIC}
});

let locationsConnection = new locationsConnectionClass({
    "id": "locations",
    "config": {"endpoint": process.env.KAFKA_ENDPOINT, "keyField": process.env.LOCATIONS_KEYFIELD, "topic": process.env.LOCATIONS_TOPIC}
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
    morgan = require('morgan'),
    server = require('http').createServer(app),
    promClient = require('prom-client'),
    estimatedArrivalsConnectionClass = require('topological-kafka'),
    locationsConnectionClass = require('topological-kafka'),
    predictArrivalsProcessorClass = require('./processors/predictArrivals.js');

// CONNECTIONS =============================================================

let estimatedArrivalsConnection = new estimatedArrivalsConnectionClass({
    "id": "estimatedArrivals",
    "config": {"endpoint": process.env.KAFKA_ENDPOINT, "keyField": process.env.ESTIMATED_ARRIVALS_KEYFIELD, "topic": process.env.ESTIMATED_ARRIVALS_TOPIC}
});

let locationsConnection = new locationsConnectionClass({
    "id": "locations",
    "config": {"endpoint": process.env.KAFKA_ENDPOINT, "keyField": process.env.LOCATIONS_KEYFIELD, "topic": process.env.LOCATIONS_TOPIC}
});

// PROCESSORS ==============================================================

let predictArrivalsProcessor = new predictArrivalsProcessorClass({
    "id": "predictArrivals",
    "config": {}
});

// TOPOLOGY ================================================================

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


// METRICS ================================================================

app.get("/metrics", (req, res) => {
    res.set("Content-Type", promClient.register.contentType);
    res.end(promClient.register.metrics());
});

app.use(morgan("combined"));

server.listen(process.env.PORT);
topology.log.info("listening on port: " + process.env.PORT);

promClient.collectDefaultMetrics();
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
		"build/production",
		"build/production/deploy-all",
		"build/production/notify-arrivals",
		"build/production/notify-arrivals/Dockerfile",
		"build/production/notify-arrivals/devops",
		"build/production/notify-arrivals/devops/Chart.yaml",
		"build/production/notify-arrivals/devops/start-stage",
		"build/production/notify-arrivals/devops/values.yaml",
		"build/production/notify-arrivals/devops/templates/deployment.yaml",
		"build/production/notify-arrivals/devops/templates/service.yaml",
		"build/production/notify-arrivals",
		"build/production/notify-arrivals/package.json",
		"build/production/notify-arrivals/stage.js",
		"build/production/notify-arrivals/processors/notifyArrivals.js",
		"build/production/write-locations",
		"build/production/write-locations/Dockerfile",
		"build/production/write-locations/package.json",
		"build/production/write-locations/stage.js",
		"build/production/write-locations/processors/writeLocations.js",
		"build/production/write-locations/devops",
		"build/production/write-locations/devops/Chart.yaml",
		"build/production/write-locations/devops/start-stage",
		"build/production/write-locations/devops/values.yaml",
		"build/production/write-locations/devops/templates/deployment.yaml",
		"build/production/write-locations/devops/templates/service.yaml",
		"build/production/predict-arrivals",
		"build/production/predict-arrivals/Dockerfile",
		"build/production/predict-arrivals/package.json",
		"build/production/predict-arrivals/stage.js",
		"build/production/predict-arrivals/processors/predictArrivals.js",
		"build/production/predict-arrivals/devops",
		"build/production/predict-arrivals/devops/Chart.yaml",
		"build/production/predict-arrivals/devops/start-stage",
		"build/production/predict-arrivals/devops/values.yaml",
		"build/production/predict-arrivals/devops/templates/deployment.yaml",
		"build/production/predict-arrivals/devops/templates/service.yaml",
	}

	for _, directory := range expectedItems {
		if _, err := os.Stat(directory); os.IsNotExist(err) {
			t.Errorf("Build did not created expected directory: %s", directory)
		}
	}

	packageJsonBytes, err := ioutil.ReadFile("build/production/predict-arrivals/package.json")
	if err != nil {
		t.Errorf("Could not read package.json: %s", err)
	}

	// TODO: how does one really convert a []byte array to string
	if fmt.Sprintf("%s", packageJsonBytes) != expectedPackageJson {
		t.Errorf("package.json did not match:-->%s<-- vs. -->%s<-- did not complete successfully.", packageJsonBytes, expectedPackageJson)
	}

	stageJsBytes, err := ioutil.ReadFile("build/production/predict-arrivals/stage.js")
	if err != nil {
		t.Errorf("Could not read stage.js: %s", err)
	}

	// TODO: how does one really convert a []byte array to string
	if fmt.Sprintf("%s", stageJsBytes) != expectedStageJs {
		t.Errorf("stage.js did not match:-->%s<-- vs. -->%s<-- did not complete successfully.", stageJsBytes, expectedStageJs)
	}
}
