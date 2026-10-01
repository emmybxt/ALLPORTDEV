package main

import (
	"reflect"
	"testing"

	"allportdev/internal/config"
)

func TestStartupSelection(t *testing.T) {
	disabled := false
	services := []config.Service{{Name: "api"}, {Name: "worker", Autostart: &disabled}}
	for _, tc := range []struct {
		name    string
		names   []string
		noStart bool
		want    []int
		wantErr bool
	}{
		{name: "defaults", want: []int{0}},
		{name: "explicit manual service", names: []string{"worker"}, want: []int{1}},
		{name: "dashboard only", noStart: true},
		{name: "unknown", names: []string{"missing"}, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := startupServices(services, tc.names, tc.noStart)
			if (err != nil) != tc.wantErr {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestInitNeverOverwritesConfig(t *testing.T) {
	path := t.TempDir() + "/allportdev.yaml"
	if err := writeExample(path); err != nil {
		t.Fatal(err)
	}
	if err := writeExample(path); err == nil {
		t.Fatal("existing config overwritten")
	}
}
