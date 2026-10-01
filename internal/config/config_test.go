package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadResolvesPathsAndAutostart(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	data := "services:\n  - name: api\n    command: go run .\n  - name: worker\n    command: echo worker\n    autostart: false\n"
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Services[0].Dir != dir {
		t.Fatalf("wrong relative directory: %s", cfg.Services[0].Dir)
	}
	if !cfg.Services[0].StartsAutomatically() || cfg.Services[1].StartsAutomatically() {
		t.Fatal("incorrect autostart defaults")
	}
}

func TestRejectsInvalidConfig(t *testing.T) {
	cases := map[string]string{
		"empty":              "services: []",
		"unknown field":      "services: [{name: api, command: echo ok, commmand: typo}]",
		"missing command":    "services: [{name: api}]",
		"duplicate name":     "services: [{name: api, command: echo one}, {name: api, command: echo two}]",
		"missing directory":  "services: [{name: api, command: echo ok, dir: missing}]",
		"multiple documents": "services: [{name: api, command: echo ok}]\n---\nservices: []",
	}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			if err := os.WriteFile(path, []byte(data), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(path); err == nil {
				t.Fatal("invalid config accepted")
			}
		})
	}
}
