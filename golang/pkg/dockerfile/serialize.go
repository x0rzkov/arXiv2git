package dockerfile

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

func (c *Command) MarshalJSON() ([]byte, error) {
	rawJSON, err := json.Marshal(c.Command)
	if err != nil {
		return nil, fmt.Errorf("merge json fields: %v", err)
	}
	out := map[string]interface{}{
		"Name": c.Name,
	}
	if err := json.Unmarshal(rawJSON, &out); err != nil {
		return nil, fmt.Errorf("merge json fields: %v", err)
	}
	return json.Marshal(out)
}

func (c *Command) MarshalYAML() ([]byte, error) {
	rawYAML, err := yaml.Marshal(c.Command)
	if err != nil {
		return nil, fmt.Errorf("merge json fields: %v", err)
	}
	out := map[string]interface{}{
		"Name": c.Name,
	}
	if err := yaml.Unmarshal(rawYAML, &out); err != nil {
		return nil, fmt.Errorf("merge json fields: %v", err)
	}
	return yaml.Marshal(out)
}
