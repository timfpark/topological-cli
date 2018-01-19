package main

import (
	"testing"
)

func TestLoadEnvironment(t *testing.T) {
	builder := NewBuilder("fixtures/topology.json", "fixtures/environment.json")
	environment, err := builder.LoadEnvironment()
	if err != nil {
		t.Errorf("LoadEnvironment did not complete successfully: %s", err)
	}

	if environment.Target != "kubernetes" {
		t.Errorf("Target was not parsed correctly, got: %s", environment.Target)
	}

	if environment.Tier != "production" {
		t.Errorf("Tier was not parsed correctly, got %s", environment.Tier)
	}

	if len(environment.Connections) != 2 {
		t.Errorf("Connections was not parsed correctly, got %d", len(environment.Connections))
	}

	if _, ok := environment.Connections["locations"]; !ok {
		t.Errorf("locations connection was not parsed correctly")
	}

	if len(environment.Connections["locations"].Config) != 3 {
		t.Errorf("locations Connection config was not parsed correctly")
	}

	if len(environment.Deployments) != 3 {
		t.Errorf("Deployments was not parsed correctly, got %d", len(environment.Connections))
	}
}

func TestLoadTopology(t *testing.T) {
	builder := NewBuilder("fixtures/topology.json", "fixtures/environment.json")
	topology, err := builder.LoadTopology()
	if err != nil {
		t.Errorf("LoadTopology did not complete successfully.")
	}

	if topology.Name != "location-pipeline" {
		t.Errorf("Name was not parsed correctly")
	}

	if len(topology.Nodes) != 3 {
		t.Errorf("Nodes was not parsed correctly")
	}

	if _, ok := topology.Nodes["writeLocations"]; !ok {
		t.Errorf("writeLocations Node was not parsed correctly")
	}

	if len(topology.Nodes["writeLocations"].Inputs) != 1 {
		t.Errorf("writeLocations inputs was not parsed correctly")
	}

	if topology.Nodes["writeLocations"].Processor.File != "./processors/writeLocation.js" {
		t.Errorf("writeLocations processor was not parsed correctly")
	}

	if _, ok := topology.Nodes["predictArrivals"]; !ok {
		t.Errorf("predictArrivals Node was not parsed correctly")
	}

	if len(topology.Nodes["predictArrivals"].Outputs) != 1 {
		t.Errorf("predictArrivals outputs was not parsed correctly")
	}
}
