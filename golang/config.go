package main

import (
	"fmt"
	"time"

	"github.com/jinzhu/configor"
	"github.com/k0kubun/pp"
)

// Enable verbose mode or set env `CONFIGOR_VERBOSE_MODE` to true when running your application
// Enable debug mode or set env `CONFIGOR_DEBUG_MODE` to true when running your application
func loadConfigFile(paths ...string) error {
	// var config *Config
	config = &Config{}
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
	err := configor.New(opts).Load(config, paths...)
	if debug {
		pp.Println(config)
	}
	return err
}

type Config struct {
	Providers *ConfigProviders `json:"providers,omitempty" yaml:"providers,omitempty" toml:"providers,omitempty"`
	Store     *ConfigStore     `json:"store,omitempty" yaml:"store,omitempty" toml:"store,omitempty"`
	Zoekt     *ConfigZoekt     `json:"zoekt,omitempty" yaml:"zoekt,omitempty" toml:"zoekt,omitempty"`
}

type ConfigProviders struct {
	Github    *ConfigProvidersGithub    `json:"github,omitempty" yaml:"github,omitempty" toml:"github,omitempty"`
	HubDocker *ConfigProvidersHubDocker `json:"hub-docker,omitempty" yaml:"hub-docker,omitempty" toml:"hub-docker,omitempty"`
}

type ConfigProvidersGithub struct {
	Cache  *ConfigProvidersGithubCache  `json:"cache,omitempty" yaml:"cache,omitempty" toml:"cache,omitempty"`
	Queue  *ConfigProvidersGithubQueue  `json:"queue,omitempty" yaml:"queue,omitempty" toml:"queue,omitempty"`
	Search *ConfigProvidersGithubSearch `json:"search,omitempty" yaml:"search,omitempty" toml:"search,omitempty"`
	Tokens []string                     `json:"tokens,omitempty" yaml:"tokens,omitempty" toml:"tokens,omitempty"`
}

type ConfigProvidersGithubCache struct {
	Path string `json:"path,omitempty" yaml:"path,omitempty" toml:"path,omitempty"`
}

type ConfigProvidersGithubQueue struct {
	Jobs int `json:"jobs,omitempty" yaml:"jobs,omitempty" toml:"jobs,omitempty"`
}

type ConfigProvidersGithubSearch struct {
	Keywords []string `json:"keywords,omitempty" yaml:"keywords,omitempty" toml:"keywords,omitempty"`
}

type ConfigProvidersHubDocker struct {
	Cache  *ConfigProvidersHubDockerCache  `json:"cache,omitempty" yaml:"cache,omitempty" toml:"cache,omitempty"`
	Queue  *ConfigProvidersHubDockerQueue  `json:"queue,omitempty" yaml:"queue,omitempty" toml:"queue,omitempty"`
	Search *ConfigProvidersHubDockerSearch `json:"search,omitempty" yaml:"search,omitempty" toml:"search,omitempty"`
}

type ConfigProvidersHubDockerCache struct {
	Path string `json:"path,omitempty" yaml:"path,omitempty" toml:"path,omitempty"`
}

type ConfigProvidersHubDockerQueue struct {
	Jobs int     `json:"jobs,omitempty" yaml:"jobs,omitempty" toml:"jobs,omitempty"`
	Size float64 `json:"size,omitempty" yaml:"size,omitempty" toml:"size,omitempty"`
}

type ConfigProvidersHubDockerSearch struct {
	Keywords []string `json:"keywords,omitempty" yaml:"keywords,omitempty" toml:"keywords,omitempty"`
}

type ConfigStore struct {
	Path string `json:"path,omitempty" yaml:"path,omitempty" toml:"path,omitempty"`
}

type ConfigZoekt struct {
	Branch string `json:"branch,omitempty" yaml:"branch,omitempty" toml:"branch,omitempty"`
	Path   string `json:"path,omitempty" yaml:"path,omitempty" toml:"path,omitempty"`
}
