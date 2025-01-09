package graph_builder

import (
	"fmt"
	"log"
	"rag_server/graph"
)

type GraphBuilder[S interface{}] struct {
	// Nodes is a map of node names to node objects
	Nodes map[string]node[S]

	// Edges is a map of node names to edge objects
	Edges []edge[S]

	// EntryPoint is the starting node of the graph
	EntryPoint string
}

type node[S interface{}] struct {
	name   string
	action *graph.NodeFn[S]
}

type edge[S interface{}] struct {
	type_     graph.EdgeType
	source    string
	target    string
	condition *graph.EdgeFn[S]
}

// NewGraph creates a new graph_builder for simple graph
func NewGraph() *GraphBuilder[[]graph.Message] {
	return NewStateGraph[[]graph.Message]()
}

// NewStateGraph creates a new graph_builder for state graph
func NewStateGraph[S interface{}]() *GraphBuilder[S] {
	gb := &GraphBuilder[S]{
		Nodes: make(map[string]node[S]),
		Edges: []edge[S]{},
	}

	return gb
}

// AddNode adds a node to the graph
func (gb *GraphBuilder[S]) AddNode(name string, nodeFn graph.NodeFn[S]) {
	if _, ok := gb.Nodes[name]; ok {
		log.Fatalf("node with name %s already exists", name)
	}

	gb.Nodes[name] = node[S]{
		name:   name,
		action: &nodeFn,
	}
}

// AddEdge adds an edge to the graph
func (gb *GraphBuilder[S]) AddEdge(source, target string) {
	gb.Edges = append(gb.Edges, edge[S]{
		type_:  graph.SIMPLE,
		source: source,
		target: target,
	})
}

// AddConditionalEdge adds a conditional edge to the graph
func (gb *GraphBuilder[S]) AddConditionalEdge(source string, fn graph.EdgeFn[S]) {
	gb.Edges = append(gb.Edges, edge[S]{
		type_:     graph.CONDITIONAL,
		source:    source,
		condition: &fn,
	})
}

// SetEntryPoint sets the starting node of the graph
func (gb *GraphBuilder[S]) SetEntryPoint(name string) {
	gb.EntryPoint = name
}

// Compile builds the graph and checks for errors
func (gb *GraphBuilder[S]) Compile() (*graph.Graph[S], error) {
	final := graph.NewGraph[S]()

	// First compile all nodes as will be needed as ref by Edges
	var err error
	if final.Nodes, err = gb.compileNodes(); err != nil {
		return nil, err
	}

	// Second compile all Edges
	var edgesByNodeName map[string][]*graph.Edge[S]
	if edgesByNodeName, err = gb.compileEdges(final); err != nil {
		return nil, err
	}

	// Bind Edges to Nodes
	for name, edges := range edgesByNodeName {
		n, nodeExists := final.Nodes[name]
		if !nodeExists {
			return nil, fmt.Errorf("node %s not found", name)
		}

		n.Edges = edges
	}

	// Ensure EntryPoint was set
	if gb.EntryPoint == "" {
		return nil, fmt.Errorf("entry point not set")
	}

	// Setup graph.StateList.GetCurrent to EntryPoint of the graph
	entryPoint, entryPointExists := final.Nodes[gb.EntryPoint]
	if !entryPointExists {
		return nil, fmt.Errorf("entry point %s not found in graph", gb.EntryPoint)
	}
	final.EntryPoint = entryPoint

	return final, nil
}

func (gb *GraphBuilder[S]) compileNodes() (map[string]*graph.Node[S], error) {
	nodes := make(map[string]*graph.Node[S])

	// Create a map of graph.Node objects from the defined nodes
	for name, n := range gb.Nodes {
		nodeObj := &graph.Node[S]{
			Name:   name,
			Edges:  []*graph.Edge[S]{},
			Action: *(n.action),
		}

		if n.action == nil && name != graph.END {
			return nil, fmt.Errorf("node %s does not have a NodeFn", name)
		}

		nodes[name] = nodeObj
	}

	// Adding END node if not found
	if _, ok := nodes[graph.END]; !ok {
		nodes[graph.END] = &graph.Node[S]{
			Name: graph.END,
		}
	}

	return nodes, nil
}

func (gb *GraphBuilder[S]) compileEdges(g *graph.Graph[S]) (map[string][]*graph.Edge[S], error) {
	finalEdges := make(map[string][]*graph.Edge[S])

	for _, e := range gb.Edges {
		source, _ := g.GetNodeByName(e.source)
		target, _ := g.GetNodeByName(e.target)

		if source == nil {
			return nil, fmt.Errorf("source node %s not found", e.source)
		}

		if target == nil && e.type_ == graph.SIMPLE {
			return nil, fmt.Errorf("target node %s not found", e.target)
		} else if e.condition == nil && e.type_ == graph.CONDITIONAL {
			return nil, fmt.Errorf("condition function not found")
		}

		finalEdges[e.source] = append(finalEdges[e.source], &graph.Edge[S]{
			Type:      e.type_,
			Target:    target,
			Condition: e.condition,
		})
	}

	return finalEdges, nil
}
