package cli

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
)

// ConfigTarget describes a single target in the configuration file.
type ConfigTarget map[string]string

// ConfigFile holds configuration data for various targets.
type ConfigFile struct {
	Targets map[string]ConfigTarget `yaml:"targets"`
}

// New instantiates a configFile and immediately populates it with the
// supplied data.
func NewConfigFile(data io.Reader, defaultTarget ConfigTarget) (*ConfigFile, error) {
	configFile := &ConfigFile{
		Targets: map[string]ConfigTarget{
			"default": defaultTarget,
		},
	}
	err := configFile.Load(data)
	if err != nil {
		return nil, err
	}
	return configFile, err
}

// String returns a yaml-formatted representation of the content of config.
func (config *ConfigFile) String() string {
	yaml, _ := yaml.Marshal(config)
	return string(yaml)
}

// Create inserts a new target.
func (config *ConfigFile) Create(name string, storeType string) *ConfigFile {
	targets := config.Targets
	if _, ok := targets[name]; !ok {
		config.Targets[name] = ConfigTarget{
			"type": storeType,
		}
	}
	return config
}

// ConfigTarget finds the requested target, creating one if needed.
func (config *ConfigFile) Target(name string) (*ConfigTarget, error) {
	targets := config.Targets
	if targeted, ok := targets[name]; ok {
		return &targeted, nil
	}
	return nil, fmt.Errorf("%s target not found", name)
}

// Delete removes a target by name from the configuration struct.
func (config *ConfigFile) Delete(name string) *ConfigFile {
	delete(config.Targets, name)
	return config
}

// Load reads a provided data source that is expected to contain yaml that can
// be directly unmarshalled into File field of ConfigFile.
func (config *ConfigFile) Load(data io.Reader) error {
	bytes, err := ioutil.ReadAll(data)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(bytes, &config); err != nil {
		return err
	}
	return nil
}

// Save renders the current configuration as YAML and writes it to a consumer
// specified io.Writer.
func (config *ConfigFile) Save(dest io.Writer) error {
	yaml, _ := yaml.Marshal(config)
	// validate number of bytes written too?
	if _, err := dest.Write(yaml); err != nil {
		return err
	}
	return nil
}

// Set assigns a configuration value to the target.
func (target *ConfigTarget) Set(key string, value string) *ConfigTarget {
	(*target)[key] = value
	return target
}

// Delete removes a configuration value.
func (target *ConfigTarget) Delete(key string) *ConfigTarget {
	delete(*target, key)
	return target
}

// Get retrieves a configuration value from a target without consumers knowing
// where it was stored.
func (target *ConfigTarget) Get(key string) string {
	return (*target)[key]
}
