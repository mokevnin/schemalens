package schemalens

import (
	"fmt"

	"github.com/mokevnin/schemalens/internal/config"
)

// DefaultConfigFile is the file the CLI reads from the working directory when
// no configuration is named. LoadConfig reads it for an empty path.
const DefaultConfigFile = "schemalens.yml"

// RuleConfig is the configuration of one rule.
type RuleConfig struct {
	// Enabled switches the rule on or off; nil keeps its default (see
	// RuleInfo.DefaultOn).
	Enabled *bool `yaml:"enabled,omitempty"`
	// Options are rule-specific settings; see the rule's documentation.
	Options map[string]any `yaml:"options,omitempty"`
}

// Config has the schema of schemalens.yml: which rules run, per-rule options
// and the exclusions that silence individual findings. The zero Config enables
// every default-on rule and has no exclusions.
type Config struct {
	Rules      map[string]RuleConfig `yaml:"rules,omitempty"`
	Exclusions []Exclusion           `yaml:"exclusions,omitempty"`
}

// LoadConfig reads and validates the configuration file at path. An empty path
// means DefaultConfigFile in the working directory, which is optional: when it
// does not exist the zero Config is returned. A named file must exist.
func LoadConfig(path string) (*Config, error) {
	c, err := config.Load(path)
	if err != nil {
		return nil, err
	}
	return configFromInternal(c), nil
}

// ParseConfig decodes and validates configuration YAML. Unknown keys, unknown
// rules and exclusions without a reason are errors.
func ParseConfig(data []byte) (*Config, error) {
	c, err := config.Parse(data)
	if err != nil {
		return nil, err
	}
	return configFromInternal(c), nil
}

func configFromInternal(c *config.Config) *Config {
	out := &Config{}
	if len(c.Rules) > 0 {
		out.Rules = make(map[string]RuleConfig, len(c.Rules))
		for id, rc := range c.Rules {
			out.Rules[id] = RuleConfig(rc)
		}
	}
	for _, e := range c.Exclusions {
		out.Exclusions = append(out.Exclusions, Exclusion(e))
	}
	return out
}

// internal converts to the validated internal form; a nil Config is the zero
// Config.
func (c *Config) internal() (*config.Config, error) {
	out := &config.Config{}
	if c != nil {
		if len(c.Rules) > 0 {
			out.Rules = make(map[string]config.RuleConfig, len(c.Rules))
			for id, rc := range c.Rules {
				out.Rules[id] = config.RuleConfig(rc)
			}
		}
		for _, e := range c.Exclusions {
			out.Exclusions = append(out.Exclusions, config.Exclusion(e))
		}
	}
	if err := out.Validate(); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	return out, nil
}
