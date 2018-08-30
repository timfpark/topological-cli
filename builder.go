package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

type Builder struct {
	TopologyPath    string
	EnvironmentPath string

	DeploymentPath string

	Topology    Topology
	Environment Environment
}

// DATE_TAG=`date -u +"%Y%m%dT%H%M%SZ"\`

/*
const commonDeployStage = `#!/bin/bash

kubectl create namespace $SERVICE_NAMESPACE

DATE_TAG=` + "`" + `date -u +"%Y%m%dT%H%M%SZ"` + "`" + `

RELEASE_TAG=$CONTAINER_REPO/$SERVICE_NAME:$DATE_TAG
docker build -t $RELEASE_TAG .
docker push $RELEASE_TAG

helm upgrade $SERVICE_NAME --namespace $SERVICE_NAMESPACE --install --set image=$RELEASE_TAG --values=./values.yaml ../common/$APP_TYPE/.
`
*/

const chartYAMLTemplate = `name: %s`

const deploymentYAMLTemplate = `apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: {{ .Values.serviceName }}
  labels:
    name: {{ .Values.serviceName }}
spec:
  replicas: {{ .Values.replicas }}
  strategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        name: {{ .Values.serviceName }}
    spec:
      imagePullSecrets:
      - name: {{ .Values.imagePullSecrets }}
      containers:
      - name: {{ .Values.serviceName }}
        image: {{ .Values.image }}
        imagePullPolicy: {{ .Values.imagePullPolicy }}
        env:%s
        - name: LOG_LEVEL
          value: {{ .Values.logSeverity }}
        ports:
        - containerPort: {{ .Values.servicePort }}
          protocol: TCP
`

const serviceYaml = `apiVersion: v1
kind: Service
metadata:
  annotations:
    prometheus.io/scrape: 'true'
  labels:
    name: {{ .Values.serviceName }}
  name: {{ .Values.serviceName }}
  namespace: {{ .Values.serviceNamespace }}
spec:
  ports:
  - port: {{ .Values.servicePort }}
    protocol: TCP
    targetPort: {{ .Values.servicePort }}
  selector:
    name: {{ .Values.serviceName }}
  sessionAffinity: None
  type: ClusterIP
`

const valuesYAMLTemplate = `cpuRequest: '%s'
cpuLimit: '%s'
imagePullPolicy: 'Always'
imagePullSecrets: %s
logSeverity: '%s'
memoryRequest: '%s'
memoryLimit: '%s'
replicas: %d
serviceName: '%s'
serviceNamespace: '%s'
servicePort: 80`

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

func (b *Builder) addSecretEnvVarMappings(secretEnvVarMappings map[string]string, config map[string]interface{}) {
	for _, secret := range config {
		envVarName := strings.ToUpper(strings.Replace(secret.(string), "-", "_", -1))

		secretEnvVarMappings[secret.(string)] = envVarName
	}
}

func (b *Builder) collectEnvVarSecretMappings(deploymentID string) (secretEnvVarMappings map[string]string) {
	secretEnvVarMappings = map[string]string{}

	deployment := b.Environment.Deployments[deploymentID]

	// check to make sure platform is the same across the nodes of the deployment
	for nodeIdx, _ := range deployment.Nodes {
		nodeID := deployment.Nodes[nodeIdx]
		node, nodeExists := b.Topology.Nodes[nodeID]

		if !nodeExists {
			return nil
		}

		for _, connectionId := range node.Inputs {
			connection := b.Environment.Connections[connectionId]
			b.addSecretEnvVarMappings(secretEnvVarMappings, connection.Config)
		}

		for _, connectionId := range node.Outputs {
			connection := b.Environment.Connections[connectionId]
			b.addSecretEnvVarMappings(secretEnvVarMappings, connection.Config)
		}

		b.addSecretEnvVarMappings(secretEnvVarMappings, b.Environment.Processors[nodeID].Config)
	}

	return secretEnvVarMappings
}

func buildEnvVarSecretBlock(secret string, envVar string) (envVarBlock string) {
	return fmt.Sprintf(`
        - name: %s
          valueFrom:
            secretKeyRef:
              name: %s
              key: %s`, envVar, secret, envVar)
}

func (b *Builder) buildDeploymentSecretsEnvVarBlock(deploymentID string) (envVarBlock string) {
	envVarSecretEntries := ""

	envVarSecretMappings := b.collectEnvVarSecretMappings(deploymentID)
	for secret, envVar := range envVarSecretMappings {
		envVarSecretEntries += buildEnvVarSecretBlock(secret, envVar)
	}

	return envVarSecretEntries
}

func (b *Builder) FillValuesYAML(deploymentID string) (valuesYAML string) {
	deployment := b.Environment.Deployments[deploymentID]
	return fmt.Sprintf(valuesYAMLTemplate, deployment.CPU.Request, deployment.CPU.Limit, b.Environment.PullSecret, deployment.LogSeverity, deployment.Memory.Request, deployment.Memory.Limit, deployment.Replicas.Min, deploymentID, b.Environment.Namespace)
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

	devopsDir := path.Join(b.DeploymentPath, "devops")
	err = os.Mkdir(devopsDir, 0755)
	if err != nil {
		return err
	}

	err = platformBuilder.BuildSource()
	if err != nil {
		return err
	}

	valuesYAML := b.FillValuesYAML(deploymentID)
	err = ioutil.WriteFile(path.Join(devopsDir, "values.yaml"), []byte(valuesYAML), 0755)
	if err != nil {
		return err
	}

	chartYAML := fmt.Sprintf(chartYAMLTemplate, deploymentID)
	err = ioutil.WriteFile(path.Join(devopsDir, "Chart.yaml"), []byte(chartYAML), 0755)
	if err != nil {
		return err
	}

	templateDir := path.Join(devopsDir, "templates")
	err = os.Mkdir(templateDir, 0755)
	if err != nil {
		return err
	}

	deploymentEnvVarSecretsBlock := b.buildDeploymentSecretsEnvVarBlock(deploymentID)
	deploymentYAML := fmt.Sprintf(deploymentYAMLTemplate, deploymentEnvVarSecretsBlock)

	err = ioutil.WriteFile(path.Join(templateDir, "deployment.yaml"), []byte(deploymentYAML), 0644)
	if err != nil {
		return err
	}

	ioutil.WriteFile(path.Join(templateDir, "service.yaml"), []byte(serviceYaml), 0644)

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
