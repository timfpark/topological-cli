package main

import (
	"testing"
)

func TestBuildPackageJson(t *testing.T) {
	builder := NewBuilder("fixtures/topology.json", "fixtures/environment.json")
	environment, err := builder.LoadEnvironment()
	if err != nil {
		t.Errorf("LoadEnvironment did not complete successfully.")
	}

	nodeJsBuilder := NodeJsPlatformBuilder{
		DeploymentID: "write-locations",
		Environment:  *environment,
	}

	packageJson := nodeJsBuilder.BuildPackageJson()

	const expectedPackageJson = `{
		"name": "stage-write-locations",
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

	if packageJson != expectedPackageJson {
		t.Errorf("package.json did not match:-->%s<-- vs. -->%s<-- did not complete successfully.", packageJson, expectedPackageJson)
	}
}
