package blueprint

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type BlueprintYAML struct {
	Name        string              `yaml:"name"`
	Version     string              `yaml:"version"`
	Description string              `yaml:"description"`
	WhenToUse   string              `yaml:"when_to_use,omitempty"`
	Start       string              `yaml:"start"`
	Hooks       []HookYAML          `yaml:"hooks,omitempty"`
	Nodes       map[string]NodeYAML `yaml:"nodes"`
	Edges       []EdgeYAML          `yaml:"edges"`
}

// HookYAML declares a blueprint-level hook (parsed for documentation and future wiring).
type HookYAML struct {
	Event  string                 `yaml:"event"`
	Action string                 `yaml:"action"`
	Config map[string]interface{} `yaml:"config,omitempty"`
}

type NodeYAML struct {
	Type            string                 `yaml:"type"`
	Description     string                 `yaml:"description,omitempty"`
	ConcurrencySafe *bool                  `yaml:"concurrency_safe,omitempty"`
	AllowedTools    []string               `yaml:"allowed_tools,omitempty"`
	MaxRetries      int                    `yaml:"max_retries,omitempty"`
	Config          map[string]interface{} `yaml:"config"`
}

type EdgeYAML struct {
	From      string `yaml:"from"`
	To        string `yaml:"to"`
	Condition string `yaml:"condition,omitempty"`
}

func ParseBlueprintYAML(data []byte) (*BlueprintYAML, error) {
	var bp BlueprintYAML
	if err := yaml.Unmarshal(data, &bp); err != nil {
		return nil, fmt.Errorf("parse blueprint YAML: %w", err)
	}
	if bp.Name == "" {
		return nil, fmt.Errorf("blueprint missing required field: name")
	}
	if bp.Start == "" {
		return nil, fmt.Errorf("blueprint missing required field: start")
	}
	if len(bp.Nodes) == 0 {
		return nil, fmt.Errorf("blueprint has no nodes")
	}
	return &bp, nil
}

func (bp *BlueprintYAML) BuildGraph(executor AgentExecutor) (*Graph, error) {
	g := NewGraph()
	if err := bp.addNodesToGraph(g, executor); err != nil {
		return nil, err
	}
	for _, ey := range bp.Edges {
		if err := g.AddEdge(Edge{From: ey.From, To: ey.To, Condition: ey.Condition}); err != nil {
			return nil, err
		}
	}
	if err := g.SetStartNode(bp.Start); err != nil {
		return nil, err
	}
	return g, nil
}

func (bp *BlueprintYAML) addNodesToGraph(g *Graph, executor AgentExecutor) error {
	for id, ny := range bp.Nodes {
		if err := validateNodeYAML(id, ny); err != nil {
			return err
		}
		node, err := buildNode(id, ny, executor)
		if err != nil {
			return fmt.Errorf("node %q: %w", id, err)
		}
		if err := g.AddNode(node); err != nil {
			return err
		}
	}
	return nil
}

func validateNodeYAML(nodeID string, ny NodeYAML) error {
	if len(ny.AllowedTools) > 0 && ny.Type != "agentic" {
		return fmt.Errorf("node %q: allowed_tools is only valid for agentic nodes", nodeID)
	}
	// concurrency_safe on non-agentic nodes is ignored; deterministic and gate use
	// code-level IsConcurrencySafe() semantics.
	return nil
}

func cloneConfig(src map[string]interface{}) map[string]interface{} {
	if src == nil {
		return make(map[string]interface{})
	}
	dst := make(map[string]interface{}, len(src)+4)
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func buildNode(id string, ny NodeYAML, executor AgentExecutor) (Node, error) {
	switch ny.Type {
	case "agentic":
		prompt, _ := ny.Config["prompt"].(string)
		if prompt == "" {
			return nil, fmt.Errorf("agentic node missing 'prompt' in config")
		}
		cfg := cloneConfig(ny.Config)
		if len(ny.AllowedTools) > 0 {
			tools := make([]string, len(ny.AllowedTools))
			copy(tools, ny.AllowedTools)
			cfg["allowed_tools"] = tools
		}
		if ny.Description != "" {
			cfg["node_description"] = ny.Description
		}
		if ny.MaxRetries != 0 {
			cfg["max_retries"] = ny.MaxRetries
		}
		n := NewAgenticNode(id, prompt, cfg, executor)
		if ny.ConcurrencySafe != nil {
			n.SetConcurrencySafe(*ny.ConcurrencySafe)
		}
		return n, nil
	case "deterministic":
		command, _ := ny.Config["command"].(string)
		if command == "" {
			return nil, fmt.Errorf("deterministic node missing 'command' in config")
		}
		n := NewDeterministicNode(id, command)
		n.description = ny.Description
		n.maxRetries = ny.MaxRetries
		return n, nil
	case "gate":
		checkNode, _ := ny.Config["check_node"].(string)
		if checkNode == "" {
			return nil, fmt.Errorf("gate node missing 'check_node' in config")
		}
		n := NewGateNode(id, checkNode)
		n.description = ny.Description
		n.maxRetries = ny.MaxRetries
		return n, nil
	case "eval":
		prompt, _ := ny.Config["prompt"].(string)
		if prompt == "" {
			return nil, fmt.Errorf("eval node missing 'prompt' in config")
		}
		criteriaRaw, _ := ny.Config["criteria"].([]interface{})
		criteria := make([]string, 0, len(criteriaRaw))
		for _, c := range criteriaRaw {
			if s, ok := c.(string); ok {
				criteria = append(criteria, s)
			}
		}
		threshold := 0.7
		if t, ok := ny.Config["threshold"].(float64); ok {
			threshold = t
		}
		return NewEvalNode(id, prompt, criteria, threshold, executor), nil
	default:
		return nil, fmt.Errorf("unknown node type: %q", ny.Type)
	}
}
