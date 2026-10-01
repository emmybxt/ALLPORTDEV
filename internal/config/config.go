package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

type Service struct {
	Name      string            `yaml:"name"`
	Command   string            `yaml:"command"`
	Dir       string            `yaml:"dir"`
	Env       map[string]string `yaml:"env"`
	Autostart *bool             `yaml:"autostart"`
}

func (s Service) StartsAutomatically() bool {
	return s.Autostart == nil || *s.Autostart
}

type Config struct {
	Services []Service `yaml:"services"`
}

func Load(path string) (Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}
	defer f.Close()
	var cfg Config
	decoder := yaml.NewDecoder(f)
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return cfg, fmt.Errorf("read config: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return cfg, fmt.Errorf("config must contain exactly one YAML document")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return cfg, err
	}
	return cfg, cfg.validate(filepath.Dir(abs))
}

func (c *Config) validate(base string) error {
	if len(c.Services) == 0 {
		return fmt.Errorf("configure at least one service")
	}
	names := make(map[string]bool)
	for i := range c.Services {
		s := &c.Services[i]
		if err := validateService(*s); err != nil {
			return err
		}
		if names[s.Name] {
			return fmt.Errorf("duplicate service name %q", s.Name)
		}
		names[s.Name] = true
		if !filepath.IsAbs(s.Dir) {
			s.Dir = filepath.Join(base, s.Dir)
		}
		info, err := os.Stat(s.Dir)
		if err != nil {
			return fmt.Errorf("service %s directory: %w", s.Name, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("service %s: %s is not a directory", s.Name, s.Dir)
		}
	}
	return nil
}

func validateService(s Service) error {
	if strings.TrimSpace(s.Name) == "" {
		return fmt.Errorf("service name is required")
	}
	if strings.ContainsAny(s.Name, " \t\r\n\x1b") {
		return fmt.Errorf("service name %q must not contain whitespace or escape characters", s.Name)
	}
	if strings.TrimSpace(s.Command) == "" {
		return fmt.Errorf("service %s: command is required", s.Name)
	}
	for key, value := range s.Env {
		if key == "" || strings.ContainsAny(key, "=\x00") || strings.ContainsRune(value, 0) {
			return fmt.Errorf("service %s: invalid environment variable %q", s.Name, key)
		}
	}
	return nil
}
