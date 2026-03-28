package blueprint

import "fmt"

type Graph struct {
	nodes     map[string]Node
	edges     []Edge
	adjacency map[string][]Edge
	startNode string
}

func NewGraph() *Graph {
	return &Graph{
		nodes:     make(map[string]Node),
		adjacency: make(map[string][]Edge),
	}
}

func (g *Graph) AddNode(n Node) error {
	if _, exists := g.nodes[n.ID()]; exists {
		return fmt.Errorf("duplicate node: %s", n.ID())
	}
	g.nodes[n.ID()] = n
	return nil
}

func (g *Graph) GetNode(id string) (Node, bool) {
	n, ok := g.nodes[id]
	return n, ok
}

func (g *Graph) NodeCount() int {
	return len(g.nodes)
}

func (g *Graph) SetStartNode(id string) error {
	if _, ok := g.nodes[id]; !ok {
		return fmt.Errorf("start node %q not found in graph", id)
	}
	g.startNode = id
	return nil
}

func (g *Graph) StartNode() string {
	return g.startNode
}

func (g *Graph) AddEdge(e Edge) error {
	if _, ok := g.nodes[e.From]; !ok {
		return fmt.Errorf("edge source %q not found", e.From)
	}
	if _, ok := g.nodes[e.To]; !ok {
		return fmt.Errorf("edge target %q not found", e.To)
	}
	g.edges = append(g.edges, e)
	g.adjacency[e.From] = append(g.adjacency[e.From], e)
	return nil
}

// NextNodes returns target node IDs reachable from nodeID.
// For unconditional edges, pass condition="".
// For gate edges, pass condition="pass" or "fail".
func (g *Graph) NextNodes(nodeID, condition string) []string {
	var result []string
	for _, e := range g.adjacency[nodeID] {
		if condition == "" && e.Condition == "" {
			result = append(result, e.To)
		} else if e.Condition == condition {
			result = append(result, e.To)
		}
	}
	return result
}

func (g *Graph) Validate() error {
	if g.startNode == "" {
		return fmt.Errorf("no start node set")
	}
	if _, ok := g.nodes[g.startNode]; !ok {
		return fmt.Errorf("start node %q not in graph", g.startNode)
	}
	visited := make(map[string]bool)
	g.dfs(g.startNode, visited)
	for id := range g.nodes {
		if !visited[id] {
			return fmt.Errorf("node %q unreachable from start %q", id, g.startNode)
		}
	}
	return nil
}

func (g *Graph) dfs(nodeID string, visited map[string]bool) {
	if visited[nodeID] {
		return
	}
	visited[nodeID] = true
	for _, e := range g.adjacency[nodeID] {
		g.dfs(e.To, visited)
	}
}
