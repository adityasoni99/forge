package blueprint

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type BlueprintYAML struct {
	Name        string              `yaml:"name"`
	Version     string              `yaml:"version"`
	Description string              `yaml:"description"`
	Start       string              `yaml:"start"`
	Nodes       map[string]NodeYAML `yaml:"nodes"`
	Edges       []EdgeYAML          `yaml:"edges"`
}

type NodeYAML struct {
	Type   string                 `yaml:"type"`
	Config map[string]interface{} `yaml:"config"`
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

func buildNode(id string, ny NodeYAML, executor AgentExecutor) (Node, error) {
	switch ny.Type {
	case "agentic":
		prompt, _ := ny.Config["prompt"].(string)
		return NewAgenticNode(id, prompt, ny.Config, executor), nil
	case "deterministic":
		command, _ := ny.Config["command"].(string)
		if command == "" {
			return nil, fmt.Errorf("deterministic node missing 'command' in config")
		}
		return NewDeterministicNode(id, command), nil
	case "gate":
		checkNode, _ := ny.Config["check_node"].(string)
		if checkNode == "" {
			return nil, fmt.Errorf("gate node missing 'check_node' in config")
		}
		return NewGateNode(id, checkNode), nil
	default:
		return nil, fmt.Errorf("unknown node type: %q", ny.Type)
	}
}
