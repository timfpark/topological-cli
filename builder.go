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

	Topology    Topology
	Environment Environment
}

// DATE_TAG=`date -u +"%Y%m%dT%H%M%SZ"\`

const commonDeployStage = `#!/bin/bash

kubectl create namespace $SERVICE_NAME

RELEASE_TAG=$CONTAINER_REPO/$SERVICE_NAME:$DATE_TAG
docker build -t $RELEASE_TAG .
docker push $RELEASE_TAG

helm upgrade $SERVICE_NAME --namespace $SERVICE_NAME --install --set image=$RELEASE_TAG --values=./values.yaml ../common/$APP_TYPE/.
`

const chartYaml = `name: pipeline-stage`

const deploymentYaml = `apiVersion: apps/v1beta1
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
  namespace: {{ .Values.serviceName }}
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
		node := b.Topology.Nodes[nodeId]

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

	return platformBuilder.BuildDeployment()
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

	// copy common deployment elements down into build
	commonDir := path.Join(tierDir, "common")
	err = os.Mkdir(commonDir, 0755)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path.Join(commonDir, "deploy-stage"), []byte(commonDeployStage), 0755)
	if err != nil {
		return err
	}

	helmDir := path.Join(commonDir, "pipeline-stage")
	err = os.Mkdir(helmDir, 0755)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path.Join(helmDir, "Chart.yaml"), []byte(chartYaml), 0755)
	if err != nil {
		return err
	}

	templateDir := path.Join(helmDir, "templates")
	err = os.Mkdir(templateDir, 0755)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path.Join(templateDir, "deployment.yaml"), []byte(deploymentYaml), 0644)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path.Join(templateDir, "service.yaml"), []byte(serviceYaml), 0644)
}
