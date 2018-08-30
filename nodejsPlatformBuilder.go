package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
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
	ProcessorPath  string
}

const deployStageTemplate = `#/bin/bash

CONTAINER_REPO=%s SERVICE_NAME=%s SERVICE_NAMESPACE=%s APP_TYPE=pipeline-stage ../common/deploy-stage
`

const dockerFile = `FROM node:carbon

WORKDIR /app

COPY . .
RUN npm install

EXPOSE 80

CMD [ "devops/start-stage" ]
`

const startStage = `#!/bin/bash

export PORT=80

npm start
`

func (b *NodeJsPlatformBuilder) collectDependencies() (dependencies map[string]string) {
	dependencies = map[string]string{}
	for _, connection := range b.Environment.Connections {
		for packageName, version := range connection.Dependencies {
			dependencies[packageName] = version
		}
	}

	for _, nodeId := range b.Deployment.Nodes {
		node := b.Topology.Nodes[nodeId]
		for packageName, version := range node.Processor.Dependencies {
			dependencies[packageName] = version
		}
	}

	return dependencies
}

func (b *NodeJsPlatformBuilder) FillPackageJson() (packageJson string) {
	dependencies := b.collectDependencies()
	var dependencyStrings []string
	for packageName, version := range dependencies {
		dependencyStrings = append(dependencyStrings, fmt.Sprintf(`        "%s":"%s"`, packageName, version))
	}

	sort.Strings(dependencyStrings)

	return fmt.Sprintf(`{
    "name": "%s",
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
        "topological": "^1.0.33",
%s
    }
}`,
		b.DeploymentID,
		strings.Join(dependencyStrings, ",\n"))
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
    morgan = require('morgan'),
    server = require('http').createServer(app),
    promClient = require('prom-client'),
%s,
%s;`, strings.Join(connectionImports, ",\n"), strings.Join(processorImports, ",\n"))
}

func (b *NodeJsPlatformBuilder) FillConnections() (connectionInstantiations string) {
	connections := b.consolidateDeploymentConnections()

	instantiations := []string{}

	for connectionId, _ := range connections {
		connectionConfigJSON := b.buildConfig(b.Environment.Connections[connectionId].Config)
		connectionInstantiation := fmt.Sprintf(`let %sConnection = new %sConnectionClass({
    "id": "%s",
    "config": %s
});`, connectionId, connectionId, connectionId, connectionConfigJSON)
		instantiations = append(instantiations, connectionInstantiation)
	}

	sort.Strings(instantiations)

	return strings.Join(instantiations, "\n\n")
}

func (b *NodeJsPlatformBuilder) buildConfig(config map[string]interface{}) (configJSON string) {
	configKeys := make([]string, 0, len(config))
	for k := range config {
		configKeys = append(configKeys, k)
	}
	sort.Strings(configKeys)

	configEntries := []string{}

	for _, key := range configKeys {
		secret := config[key].(string)
		envVarName := strings.ToUpper(strings.Replace(secret, "-", "_", -1))
		configEntries = append(configEntries, fmt.Sprintf(`"%s": process.env.%s`, key, envVarName))
	}

	return fmt.Sprintf(`{%s}`, strings.Join(configEntries, ", "))
}

func (b *NodeJsPlatformBuilder) FillProcessors() (processorInstantiations string) {
	instantiations := []string{}

	for _, nodeID := range b.Deployment.Nodes {
		processorConfigJSON := b.buildConfig(b.Environment.Processors[nodeID].Config)
		processorInstantiation := fmt.Sprintf(`let %sProcessor = new %sProcessorClass({
    "id": "%s",
    "config": %s
});`, nodeID, nodeID, nodeID, processorConfigJSON)
		instantiations = append(instantiations, processorInstantiation)
	}

	sort.Strings(instantiations)

	return strings.Join(instantiations, "\n\n")
}

func (b *NodeJsPlatformBuilder) CopyProcessors() (err error) {
	for _, nodeId := range b.Deployment.Nodes {
		node := b.Topology.Nodes[nodeId]

		processorFileNameParts := strings.Split(node.Processor.File, "/")
		processorFileName := processorFileNameParts[len(processorFileNameParts)-1]
		processorPath := fmt.Sprintf("%s/%s", b.ProcessorPath, processorFileName)

		err = CopyFile(node.Processor.File, processorPath)
		if err != nil {
			return err
		}
	}

	return nil
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

	return fmt.Sprintf(`%s

// CONNECTIONS =============================================================

%s

// PROCESSORS ==============================================================

%s

// TOPOLOGY ================================================================

%s

// METRICS ================================================================

app.get("/metrics", (req, res) => {
    res.set("Content-Type", promClient.register.contentType);
    res.end(promClient.register.metrics());
});

app.use(morgan("combined"));

server.listen(process.env.PORT);
topology.log.info("listening on port: " + process.env.PORT);

promClient.collectDefaultMetrics();
`, imports, connections, processors, topology)
}

func CopyFile(sourcePath string, destPath string) (err error) {
	sourceBytes, err := ioutil.ReadFile(sourcePath)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(destPath, sourceBytes, 0644)
}

func (b *NodeJsPlatformBuilder) BuildSource() (err error) {
	b.DeploymentPath = path.Join("build", b.Environment.Tier, b.DeploymentID)

	//  deployStage := fmt.Sprintf(deployStageTemplate, b.Environment.ContainerRepo, b.DeploymentID, b.Environment.Namespace)
	err = ioutil.WriteFile(path.Join(b.DeploymentPath, "Dockerfile"), []byte(dockerFile), 0644)
	if err != nil {
		return err
	}

	devopsPath := path.Join(b.DeploymentPath, "devops")

	err = ioutil.WriteFile(path.Join(devopsPath, "start-stage"), []byte(startStage), 0755)
	if err != nil {
		return err
	}

	// create directory for code (./build/{deploymentId}/code)
	b.CodePath = b.DeploymentPath
	/*err = os.Mkdir(b.CodePath, 0755)
	if err != nil {
		return err
	}
	*/

	// create directoatry for code (./build/{deploymentId}/code/processors)
	b.ProcessorPath = path.Join(b.CodePath, "processors")
	err = os.Mkdir(b.ProcessorPath, 0755)
	if err != nil {
		return err
	}

	// create package.json
	err = ioutil.WriteFile(path.Join(b.CodePath, "package.json"), []byte(b.FillPackageJson()), 0644)
	if err != nil {
		return err
	}

	// create stage.js
	err = ioutil.WriteFile(path.Join(b.CodePath, "stage.js"), []byte(b.FillStage()), 0644)
	if err != nil {
		return err
	}

	// copy processors down into builds
	err = b.CopyProcessors()
	if err != nil {
		return err
	}

	return err
}
