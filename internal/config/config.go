package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Context struct {
	Name                 string `yaml:"name"`
	URL                  string `yaml:"url"`
	Token                string `yaml:"token"`
	Insecure             bool   `yaml:"insecure"`
	DefaultEnvironmentID int    `yaml:"defaultEnvironmentId,omitempty"`
}

type Config struct {
	CurrentContext string    `yaml:"current-context"`
	Contexts       []Context `yaml:"contexts"`
}

func ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".portainerctl", "config.yaml"), nil
}

func Load() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("malformed config file: %w", err)
	}
	return &cfg, nil
}

func Save(cfg *Config) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func (c *Config) Current() (*Context, error) {
	if url := os.Getenv("PORTAINERCTL_URL"); url != "" {
		return &Context{
			Name:     "env",
			URL:      url,
			Token:    os.Getenv("PORTAINERCTL_TOKEN"),
			Insecure: os.Getenv("PORTAINERCTL_INSECURE") == "true",
		}, nil
	}
	if len(c.Contexts) == 0 {
		return nil, fmt.Errorf("no contexts configured — run: portainerctl config add-context")
	}
	for _, ctx := range c.Contexts {
		if ctx.Name == c.CurrentContext {
			return &ctx, nil
		}
	}
	return &c.Contexts[0], nil
}

func (c *Config) AddContext(ctx Context) {
	for i, existing := range c.Contexts {
		if existing.Name == ctx.Name {
			c.Contexts[i] = ctx
			return
		}
	}
	c.Contexts = append(c.Contexts, ctx)
}

func (c *Config) RemoveContext(name string) bool {
	for i, ctx := range c.Contexts {
		if ctx.Name == name {
			c.Contexts = append(c.Contexts[:i], c.Contexts[i+1:]...)
			if c.CurrentContext == name {
				c.CurrentContext = ""
				if len(c.Contexts) > 0 {
					c.CurrentContext = c.Contexts[0].Name
				}
			}
			return true
		}
	}
	return false
}
