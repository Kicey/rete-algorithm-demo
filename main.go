package main

import (
	"fmt"
	"reflect"
)

// Fact Definitions (Airline Domain)
type Account struct {
	ID          string
	RewardMiles int
	Status      string
}

type Flight struct {
	Miles    int
	Category string
	Airline  string
}

func main() {
	fmt.Println("--- Building Rete Network ---")

	// Create an empty network
	reteRoot := &ReteNode{}

	// Object type nodes
	accountNode := &ObjectTypeNode{ObjectType: reflect.TypeOf(Account{})}
	flightNode := &ObjectTypeNode{ObjectType: reflect.TypeOf(Flight{})}

	// Append object nodes to the root Rete node
	reteRoot.Children = append(reteRoot.Children, accountNode)
	reteRoot.Children = append(reteRoot.Children, flightNode)

	// Build Rule 1: IF reward miles > 100k, THEN status = Gold
	// This only requires Account class checking.
	alphaMem1 := &AlphaMemory{}
	rule1Alpha := &AlphaNode{
		Description: "Account.RewardMiles > 100000",
		Condition: func(f Fact) bool {
			if a, ok := f.(Account); ok {
				return a.RewardMiles > 100000
			}
			return false
		},
		Memory: alphaMem1,
	}

	rule1Terminal := &TerminalNode{
		RuleName: "Assign Gold Status",
		Action: func(facts []Fact) {
			if acc, ok := facts[0].(Account); ok {
				fmt.Printf(">> ACTION EXECUTED: Account %s upgraded to Gold status!\n", acc.ID)
			}
		},
	}

	// Link them up
	alphaMem1.Children = append(alphaMem1.Children, rule1Terminal)
	accountNode.Children = append(accountNode.Children, rule1Alpha)

	// Build Rule 2: IF flight miles >= 500, THEN reward flight miles
	alphaMem2 := &AlphaMemory{}
	rule2Alpha := &AlphaNode{
		Description: "Flight.Miles >= 500",
		Condition: func(f Fact) bool {
			if fl, ok := f.(Flight); ok {
				return fl.Miles >= 500
			}
			return false
		},
		Memory: alphaMem2,
	}

	rule2Terminal := &TerminalNode{
		RuleName: "Reward Flight Miles",
		Action: func(facts []Fact) {
			if fl, ok := facts[0].(Flight); ok {
				fmt.Printf(">> ACTION EXECUTED: Flight over 500 miles. Ready to award %d miles.\n", fl.Miles)
			}
		},
	}

	alphaMem2.Children = append(alphaMem2.Children, rule2Terminal)
	flightNode.Children = append(flightNode.Children, rule2Alpha)

	// Build Rule 7: IF status is Gold AND flight airline is not partner, THEN reward 100% bonus miles
	// This involves a JOIN (Beta Node)
	// We reuse alphaMem1 (Account>100k) as the Left Input (Account part)
	// We need a new AlphaNode for the Right Input (Flight part: Not Partner)

	alphaMem3 := &AlphaMemory{}
	rule7AlphaRight := &AlphaNode{
		Description: "Flight.Airline != Partner",
		Condition: func(f Fact) bool {
			if fl, ok := f.(Flight); ok {
				return fl.Airline != "Partner"
			}
			return false
		},
		Memory: alphaMem3,
	}
	flightNode.Children = append(flightNode.Children, rule7AlphaRight)

	betaMem7 := &BetaMemory{}
	betaNode7 := &BetaNode{
		Description: "Join Account(Gold) & Flight(!Partner)",
		JoinCondition: func(left []Fact, right Fact) bool {
			// This simplified condition always joins an account to a flight since the alpha filters did the work.
			// Usually here you'd match an ID like Account.ID == Flight.AccountID
			return true
		},
		Memory: betaMem7,
	}

	rule7Terminal := &TerminalNode{
		RuleName: "+100% Bonus Miles for Gold Status",
		Action: func(facts []Fact) {
			acc := facts[0].(Account)
			fl := facts[1].(Flight)
			fmt.Printf(">> ACTION EXECUTED: Gold member %s earns 100%% bonus miles. Bonus: %d\n", acc.ID, fl.Miles)
		},
	}

	// Link up the Beta Join
	// We want AlphaMem1 to be the Left Input and AlphaMem3 to be the Right input
	// Helper for Left Input bridging (Alpha to Beta)
	leftBridge := &AlphaToBetaLeftBridge{Target: betaNode7}
	alphaMem1.Children = append(alphaMem1.Children, leftBridge)

	// Right input directly from AlphaMem3
	alphaMem3.Children = append(alphaMem3.Children, betaNode7)
	betaMem7.Children = append(betaMem7.Children, rule7Terminal)

	fmt.Println("--- Network Builder Complete --- \n")

	// Rete Runtime evaluation
	fmt.Println("--- Rete Runtime Cycle: Asserting Facts ---")

	// Create facts based on the article example
	joeAccount := Account{ID: "Joe123", RewardMiles: 150000, Status: "Unknown"}
	sfFlight := Flight{Miles: 2419, Category: "Economy", Airline: "Original"}

	// Assert facts into Rete Network Root
	fmt.Printf("\n[Input Fact] Entering Account... \n")
	reteRoot.Assert(joeAccount)

	fmt.Printf("\n[Input Fact] Entering Flight... \n")
	reteRoot.Assert(sfFlight)

	fmt.Println("\n--- Operations Finished ---")
}
