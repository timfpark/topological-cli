package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

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

func TestFillPackageJson(t *testing.T) {
	builder := NewBuilder("fixtures/topology.json", "fixtures/environment.json")
	environment, err := builder.LoadEnvironment()
	if err != nil {
		t.Errorf("LoadEnvironment did not complete successfully.")
	}

	nodeJsBuilder := NodeJsPlatformBuilder{
		DeploymentID: "write-locations",
		Environment:  *environment,
	}

	packageJson := nodeJsBuilder.FillPackageJson()

	if packageJson != expectedPackageJson {
		t.Errorf("package.json did not match:-->%s<-- vs. -->%s<-- did not complete successfully.", packageJson, expectedPackageJson)
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

	packageJsonBytes, err := ioutil.ReadFile("build/write-locations/code/package.json")
	if err != nil {
		t.Errorf("Build did not complete successfully: %s", err)
	}

	// TODO: how does one really convert a []byte array to string
	if fmt.Sprintf("%s", packageJsonBytes) != expectedPackageJson {
		t.Errorf("package.json did not match:-->%s<-- vs. -->%s<-- did not complete successfully.", packageJsonBytes, expectedPackageJson)
	}
}
