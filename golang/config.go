package main

import (
	"fmt"
	"time"

	"github.com/jinzhu/configor"
	"github.com/k0kubun/pp"
)

// Enable verbose mode or set env `CONFIGOR_VERBOSE_MODE` to true when running your application
// Enable debug mode or set env `CONFIGOR_DEBUG_MODE` to true when running your application
func loadConfigFile(paths ...string) (*Config, error) {
	var config *Config
	opts := &configor.Config{
		Debug:              true,
		Verbose:            true,
		AutoReload:         false,
		AutoReloadInterval: time.Minute,
		AutoReloadCallback: func(config interface{}) {
			fmt.Printf("%v changed", config)
		},
		ErrorOnUnmatchedKeys: false,
	}
	err := configor.New(opts).Load(&config, paths...)
	if debug {
		pp.Println(config)
	}
	return config, err
}

type Config struct {
	Providers ConfigProviders `json:"providers" yaml:"providers"`
	Store     ConfigStore     `json:"store" yaml:"store"`
	Zoekt     ConfigZoekt     `json:"zoekt" yaml:"zoekt"`
}

type ConfigProviders struct {
	Github    ConfigProvidersGithub    `json:"github" yaml:"github"`
	HubDocker ConfigProvidersHubDocker `json:"hub-docker" yaml:"hub-docker"`
}

type ConfigProvidersGithub struct {
	Cache  ConfigProvidersGithubCache  `json:"cache" yaml:"cache"`
	Queue  ConfigProvidersGithubQueue  `json:"queue" yaml:"queue"`
	Search ConfigProvidersGithubSearch `json:"search" yaml:"search"`
	Tokens []string                    `json:"tokens" yaml:"tokens"`
}

type ConfigProvidersGithubCache struct {
	Path string `json:"path" yaml:"path"`
}

type ConfigProvidersGithubQueue struct {
	Jobs int `json:"jobs" yaml:"jobs"`
}

type ConfigProvidersGithubSearch struct {
	Keywords []string `json:"keywords" yaml:"keywords"`
}

type ConfigProvidersHubDocker struct {
	Cache  ConfigProvidersHubDockerCache  `json:"cache" yaml:"cache"`
	Queue  ConfigProvidersHubDockerQueue  `json:"queue" yaml:"queue"`
	Search ConfigProvidersHubDockerSearch `json:"search" yaml:"search"`
}

type ConfigProvidersHubDockerCache struct {
	Path string `json:"path" yaml:"path"`
}

type ConfigProvidersHubDockerQueue struct {
	Jobs int     `json:"jobs" yaml:"jobs"`
	Size float64 `json:"size" yaml:"size"`
}

type ConfigProvidersHubDockerSearch struct {
	Keywords []string `json:"keywords" yaml:"keywords"`
}

type ConfigStore struct {
	Path string `json:"path" yaml:"path"`
}

type ConfigZoekt struct {
	Branch string `json:"branch" yaml:"branch"`
	Path   string `json:"path" yaml:"path"`
}
