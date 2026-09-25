package config

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

type Base struct {
	File    string
	Author  string `yaml:"author,omitempty"`
	Email   string `yaml:"email,omitempty"`
	Version string `yaml:"version,omitempty"`
	Name    string `yaml:"name,omitempty"`
	Listen  string `yaml:"listen,omitempty"`
	Log     string `yaml:"log,omitempty"`
	DSN     string `yaml:"dsn,omitempty"`
}

func NewBase(configName, projectName string) (*Base, error) {
	f, err := Find(configName, projectName)
	if err != nil {
		return nil, err
	}
	var conf = Base{}
	if err := Load(f, &conf); err != nil {
		return nil, err
	}
	return &conf, nil
}

func Find(configName, projectName string) (string, error) {
	start := time.Now()
	files := []string{
		fmt.Sprintf("%s.yml", configName),
		fmt.Sprintf("config/%s.yml", configName),
		fmt.Sprintf("/etc/%s.yml", configName),
		fmt.Sprintf("/etc/%s/%s.yml", projectName, configName),
		fmt.Sprintf("/etc/%s/config.yml", projectName),
		fmt.Sprintf("/etc/%s.yml", projectName),
		fmt.Sprintf("config.yml"),
		fmt.Sprintf("config/config.yml"),
	}
	for _, f := range files {
		slog.Info(
			"config",
			"action", "trying_file",
			"duration", time.Since(start),
			"file", f,
		)
		if _, err := os.Stat(f); err == nil {
			slog.Info(
				"config",
				"action", "found_file",
				"duration", time.Since(start),
				"file", f,
			)
			return f, nil
		}
	}
	return "", fmt.Errorf("Config file not found!")
}

func Load(file string, conf any) error {
	yamlFile, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(yamlFile, conf)
}
