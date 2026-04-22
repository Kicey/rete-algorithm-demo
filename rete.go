package main

import (
	"fmt"
	"reflect"
)

// Fact represents a single piece of data in our system.
// Any struct can be a fact.
type Fact interface{}

// Condition represents a simple evaluation on a single fact (used in Alpha Nodes).
type Condition func(fact Fact) bool

// Action represents the outcome when a rule is fully matched.
type Action func(facts []Fact)

// -----------------------
// Node Definitions
// -----------------------

// ReteNode is the root of the network. All facts enter here.
type ReteNode struct {
	Children []*ObjectTypeNode
}

func (r *ReteNode) Assert(fact Fact) {
	fmt.Printf("[ReteNode] Asserting new fact: %+v\n", fact)
	for _, child := range r.Children {
		child.Assert(fact)
	}
}

// ObjectTypeNode filters facts by their specific Go type.
type ObjectTypeNode struct {
	ObjectType reflect.Type
	Children   []*AlphaNode
}

func (ot *ObjectTypeNode) Assert(fact Fact) {
	if reflect.TypeOf(fact) == ot.ObjectType {
		fmt.Printf("[ObjectTypeNode] Matched type: %s\n", ot.ObjectType.Name())
		for _, child := range ot.Children {
			child.Assert(fact)
		}
	}
}

// AlphaNode checks a specific condition on a fact (Intra-element condition).
type AlphaNode struct {
	Description string // For logging
	Condition   Condition
	Memory      *AlphaMemory
}

func (an *AlphaNode) Assert(fact Fact) {
	if an.Condition(fact) {
		fmt.Printf("[AlphaNode] Condition '%s' passed for fact: %+v\n", an.Description, fact)
		an.Memory.Add(fact)
	} else {
		fmt.Printf("[AlphaNode] Condition '%s' failed for fact: %+v\n", an.Description, fact)
	}
}

// AlphaMemory stores facts that have passed all previous alpha tests.
type AlphaMemory struct {
	Facts    []Fact
	Children []ReteNodeVisitor
}

func (am *AlphaMemory) Add(fact Fact) {
	am.Facts = append(am.Facts, fact)
	fmt.Printf("[AlphaMemory] Stored fact: %+v. Total facts here: %d\n", fact, len(am.Facts))
	for _, child := range am.Children {
		child.RightActivate(fact)
	}
}

// AlphaToBetaLeftBridge allows an AlphaMemory to act as the Left input for a BetaNode
type AlphaToBetaLeftBridge struct {
	Target *BetaNode
}

func (ab *AlphaToBetaLeftBridge) RightActivate(fact Fact) {
	// Send the single fact as an array representing left memory facts
	ab.Target.LeftActivate([]Fact{fact})
}

// BetaNode joins facts from different branches.
// It has a Left input (from a BetaMemory or AlphaMemory) and a Right input (from an AlphaMemory).
type BetaNode struct {
	Description   string
	JoinCondition func(left []Fact, right Fact) bool
	Memory        *BetaMemory
}

func (bn *BetaNode) LeftActivate(leftFacts []Fact) {
	fmt.Printf("[BetaNode-LeftActivate] '%s' checking left facts: %+v\n", bn.Description, leftFacts)
	// Add to memory first
	bn.Memory.LeftFacts = append(bn.Memory.LeftFacts, leftFacts)

	// Iterate through right facts to find matches
	for _, rightFact := range bn.Memory.RightFacts {
		if bn.JoinCondition(leftFacts, rightFact) {
			bn.Memory.Add(append(leftFacts, rightFact))
		}
	}
}

func (bn *BetaNode) RightActivate(rightFact Fact) {
	fmt.Printf("[BetaNode-RightActivate] '%s' checking right fact: %+v\n", bn.Description, rightFact)
	// Add to memory first
	bn.Memory.RightFacts = append(bn.Memory.RightFacts, rightFact)

	// Iterate through left facts to find matches
	for _, leftFacts := range bn.Memory.LeftFacts {
		if bn.JoinCondition(leftFacts, rightFact) {
			// Creating a new slice containing all facts joined together
			joined := append([]Fact{}, leftFacts...)
			joined = append(joined, rightFact)
			bn.Memory.Add(joined)
		}
	}
}

// BetaMemory stores successfully joined facts (tuples).
type BetaMemory struct {
	LeftFacts  [][]Fact // Tuples from the left input
	RightFacts []Fact   // Single facts from the right input
	Joined     [][]Fact // The newly joined tuples
	Children   []ReteNodeTupleVisitor
}

func (bm *BetaMemory) Add(joinedFacts []Fact) {
	bm.Joined = append(bm.Joined, joinedFacts)
	fmt.Printf("[BetaMemory] Stored joined facts: %+v. Total tuples here: %d\n", joinedFacts, len(bm.Joined))
	for _, child := range bm.Children {
		child.LeftActivate(joinedFacts)
	}
}

func (bm *BetaMemory) LeftActivate(leftFacts []Fact) {
	bm.LeftFacts = append(bm.LeftFacts, leftFacts)
	// Usually passed to BetaNode instead
}

// Interfaces for nodes to accept right or left facts
type ReteNodeVisitor interface {
	RightActivate(fact Fact)
}

type ReteNodeTupleVisitor interface {
	LeftActivate(facts []Fact)
}

// TerminalNode (Action Node) executes the rule's consequent.
type TerminalNode struct {
	RuleName string
	Action   Action
}

func (tn *TerminalNode) RightActivate(fact Fact) {
	fmt.Printf("[TerminalNode] Rule '%s' activated! Executing action...\n", tn.RuleName)
	tn.Action([]Fact{fact})
}

func (tn *TerminalNode) LeftActivate(facts []Fact) {
	fmt.Printf("[TerminalNode] Rule '%s' activated! Executing action...\n", tn.RuleName)
	tn.Action(facts)
}
