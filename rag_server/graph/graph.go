package graph

import (
	"context"
	"fmt"
	"log"
)

var (
	START = "START"
	END   = "END"
)

type Graph[S interface{}] struct {
	// Nodes is a map of node names to node objects
	Nodes map[string]*Node[S]

	// EntryPoint is the starting node of the graph
	EntryPoint *Node[S]

	// GetCurrent state of the graph
	StateList []StateItem[S]
}

type StateItem[S interface{}] struct {
	Node  string
	State S
}

type Node[S interface{}] struct {
	Name   string
	Action NodeFn[S]
	Edges  []*Edge[S]
}

// Edge is a struct that represents a transition between nodes
type Edge[S interface{}] struct {
	Type      EdgeType
	Target    *Node[S]
	Condition *EdgeFn[S]
}

// NodeFn represents a pointer to a function that processes a State and a context to produce a list of GetMessages or an error.
type NodeFn[S interface{}] func(S, context.Context) (S, error)

// EdgeFn represents a function that performs a transition between nodes by evaluating a state and context to return a target node name or an error.
type EdgeFn[S interface{}] func(S, context.Context) (string, error)

// NewGraph creates a new graph object
func NewGraph[S interface{}]() *Graph[S] {

	g := &Graph[S]{
		Nodes:     make(map[string]*Node[S]),
		StateList: []StateItem[S]{},
	}

	return g
}

func (g *Graph[S]) GetNodeByName(name string) (*Node[S], error) {
	if node, ok := g.Nodes[name]; ok {
		return node, nil
	}

	return nil, fmt.Errorf("node %s not found", name)
}

func (g *Graph[S]) getCurrent() (*Node[S], error) {
	nodeName := g.StateList[1:][0].Node

	if nodeName == "" {
		return nil, fmt.Errorf("Current node is empty")
	}

	return g.GetNodeByName(nodeName)
}

// Stream sends a message to the Current node
func (g *Graph[S]) Stream(input S, config context.Context) ([]StateItem[S], error) {
	log.Printf("Streaming started...")

	currentStep := StateItem[S]{
		Node:  "START",
		State: input,
	}
	g.StateList = append(g.StateList, currentStep)
	log.Printf("Input state: %v", currentStep)

	var current *Node[S]
	current = g.EntryPoint

	stepCount := 0
	maxStep := 10

	for stepCount < maxStep {
		stepCount++

		if current == nil {
			return g.StateList, fmt.Errorf("current node is nil")
		}

		if current.Name == END {
			log.Println("END node reached.\n streaming stopped.")
			// Do nothing
			return g.StateList, nil
		}

		if current.Action == nil {
			return g.StateList, fmt.Errorf("node %s have no action", current.Name)
		}

		// Execute the Current node
		currentState, err := current.Action(currentStep.State, config)
		nextStep := StateItem[S]{
			Node:  current.Name,
			State: currentState,
		}
		g.StateList = append(g.StateList, nextStep)
		log.Printf("Next state: %v", nextStep)

		if err != nil {
			return g.StateList, fmt.Errorf("error in node %s: %v", current.Name, err)
		}

		if current.Edges == nil {
			return g.StateList, fmt.Errorf("node %s does not have any edges", current.Name)
		}

		// Iterate over the edges of the Current node
		// and find the first target node matching
		var target *Node[S]
		for _, edge := range current.Edges {
			target, err = g.getEdgeTarget(input, edge, current, config)

			if err != nil {
				return g.StateList, fmt.Errorf("error in edge from %s: %v", current.Name, err)
			}
		}

		if target == nil {
			return g.StateList, fmt.Errorf("reached dead end after node %s", current.Name)
		}

		// updating loop variables
		current = target
		currentStep = nextStep
	}

	return g.StateList, fmt.Errorf("reached max step limit")
}

func (g *Graph[S]) getEdgeTarget(input S, edge *Edge[S], current *Node[S], config context.Context) (*Node[S], error) {
	if edge.Type == SIMPLE {

		if edge.Target == nil {
			return nil, fmt.Errorf("edge target is nil")
		}

		return edge.Target, nil
	}

	if edge.Type == CONDITIONAL {
		if edge.Condition == nil {
			return nil, fmt.Errorf("EdgeFn not found for conditional edge from %s", current.Name)
		}

		targetName, err := (*edge.Condition)(input, config)
		if err != nil {
			return nil, fmt.Errorf("error in edge from %s: %v", current.Name, err)
		}

		if targetName == "" {
			return nil, nil
		}

		target, err := g.GetNodeByName(targetName)
		if err != nil {
			return nil, fmt.Errorf("%s node not found: %v", targetName, err)
		}
		return target, nil
	}

	return nil, fmt.Errorf("unsupported edge type")
}
