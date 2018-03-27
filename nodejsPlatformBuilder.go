package main

import (
	"encoding/json"
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

WORKDIR /code

COPY code/. .
RUN npm install
COPY ./start-stage .

EXPOSE 80

CMD [ "./start-stage" ]
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
    "name": "stage-%s",
    "version": "1.0.0",
    "main": "stage.js",
    "scripts": {
        "start": "node stage.js"
    },
    "dependencies": {
        "express": "^4.16.2",
        "prom-client": "^10.2.2",
        "request": "^2.83.0",
        "topological": "^1.0.29",
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
		var connectionConfigJson string
		if len(connection.Config) > 0 {
			connectionConfigJsonBytes, _ := json.Marshal(connection.Config)
			connectionConfigJson = string(connectionConfigJsonBytes)
		} else {
			connectionConfigJson = "{}"
		}
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
		processorConfig := b.Environment.Processors[nodeId].Config
		var processorConfigJson string
		if len(processorConfig) > 0 {
			processorConfigJsonBytes, _ := json.Marshal(processorConfig)
			processorConfigJson = string(processorConfigJsonBytes)
		} else {
			processorConfigJson = "{}"
		}
		processorInstantiation := fmt.Sprintf(`let %sProcessor = new %sProcessorClass({
    "id": "%s",
    "config": %s
});`, nodeId, nodeId, nodeId, processorConfigJson)
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

server.listen(process.env.PORT);
topology.log.info("listening on port: " + process.env.PORT);

promClient.collectDefaultMetrics();
`, imports, connections, processors, topology)
}

func (b *NodeJsPlatformBuilder) FillValuesYaml() (valuesYaml string) {
	CPU := "250m"
	if b.Deployment.Replicas.CPU != "" {
		CPU = b.Deployment.Replicas.CPU
	}

	Memory := "250Mi"
	if b.Deployment.Replicas.Memory != "" {
		Memory = b.Deployment.Replicas.Memory
	}

	return fmt.Sprintf(`serviceName: "%s"
serviceNamespace: "%s"
servicePort: 80
replicas: %d
imagePullPolicy: "Always"
imagePullSecrets: %s
cpu: "%s"
memory: "%s"`, b.DeploymentID, b.Environment.Namespace, b.Deployment.Replicas.Min, b.Environment.PullSecret, CPU, Memory)
}

func CopyFile(sourcePath string, destPath string) (err error) {
	sourceBytes, err := ioutil.ReadFile(sourcePath)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(destPath, sourceBytes, 0644)
}

func (b *NodeJsPlatformBuilder) BuildDeployment() (err error) {
	b.DeploymentPath = path.Join("build", b.Environment.Tier, b.DeploymentID)
	err = os.Mkdir(b.DeploymentPath, 0755)
	if err != nil {
		return err
	}

	deployStage := fmt.Sprintf(deployStageTemplate, b.Environment.ContainerRepo, b.DeploymentID, b.Environment.Namespace)
	ioutil.WriteFile(path.Join(b.DeploymentPath, "deploy-stage"), []byte(deployStage), 0755)
	ioutil.WriteFile(path.Join(b.DeploymentPath, "Dockerfile"), []byte(dockerFile), 0644)
	ioutil.WriteFile(path.Join(b.DeploymentPath, "start-stage"), []byte(startStage), 0755)
	ioutil.WriteFile(path.Join(b.DeploymentPath, "values.yaml"), []byte(b.FillValuesYaml()), 0644)

	// create directory for code (./build/{deploymentId}/code)
	b.CodePath = path.Join(b.DeploymentPath, "code")
	err = os.Mkdir(b.CodePath, 0755)
	if err != nil {
		return err
	}

	// create directoatry for code (./build/{deploymentId}/code/processors)
	b.ProcessorPath = path.Join(b.CodePath, "processors")
	err = os.Mkdir(b.ProcessorPath, 0755)
	if err != nil {
		return err
	}

	// create package.json
	ioutil.WriteFile(path.Join(b.CodePath, "package.json"), []byte(b.FillPackageJson()), 0644)

	// create stage.js
	ioutil.WriteFile(path.Join(b.CodePath, "stage.js"), []byte(b.FillStage()), 0644)

	// copy processors down into builds
	err = b.CopyProcessors()
	if err != nil {
		return err
	}

	return err
}
