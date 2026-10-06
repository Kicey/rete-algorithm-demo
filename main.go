package main

import "fmt"

type Account struct{ ID, Status string }
type Flight struct {
	ID, AccountID string
	Miles         int
	Airline       string
}

func NewEngine() *Engine {
	alpha := func(test func(any) bool) *Alpha {
		return &Alpha{Test: test, Memory: make(map[int]Fact)}
	}
	e := &Engine{
		facts:    make(map[int]Fact),
		pending:  make(map[string]func()),
		left:     &Beta{Memory: make(map[string]Tuple)},
		bonus:    &Beta{Memory: make(map[string]Tuple)},
		gold:     alpha(func(v any) bool { return v.(Account).Status == "Gold" }),
		eligible: alpha(func(v any) bool { return v.(Flight).Miles >= 500 }),
		other:    alpha(func(v any) bool { return v.(Flight).Airline != "Partner" }),
	}
	terminal := func(rule string, action func(Tuple)) func(Tuple, bool) {
		return func(t Tuple, add bool) {
			key := rule + "/" + t.Key
			if add {
				e.pending[key] = func() { action(t) }
			} else {
				delete(e.pending, key)
			}
		}
	}
	base := terminal("base", func(t Tuple) {
		f := t.Facts[0].Value.(Flight)
		fmt.Printf("base %s: %d miles\n", f.ID, f.Miles)
	})
	e.bonus.Out = append(e.bonus.Out, terminal("bonus", func(t Tuple) {
		a := t.Facts[0].Value.(Account)
		f := t.Facts[1].Value.(Flight)
		fmt.Printf("bonus %s/%s: %d miles\n", a.ID, f.ID, f.Miles)
	}))
	j := &Join{
		Left: e.left, Right: e.other, Next: e.bonus,
		Test: func(t Tuple, f Fact) bool {
			return t.Facts[0].Value.(Account).ID == f.Value.(Flight).AccountID
		},
	}
	e.gold.Out = append(e.gold.Out, func(f Fact, add bool) {
		e.left.Push(singleton(f), add)
	})
	e.left.Out = append(e.left.Out, j.LeftEvent)
	e.eligible.Out = append(e.eligible.Out,
		func(f Fact, add bool) { base(singleton(f), add) }, e.other.Push)
	e.other.Out = append(e.other.Out, j.RightEvent)
	return e
}

func main() {
	e := NewEngine()
	e.Insert(Account{"A1", "Gold"})
	silver := e.Insert(Account{"A2", "Silver"})
	f1 := e.Insert(Flight{"F1", "A1", 2419, "Original"})
	e.Insert(Flight{"F2", "A2", 800, "Original"})
	e.Insert(Flight{"F3", "A1", 300, "Original"})
	e.Insert(Flight{"F4", "A1", 900, "Partner"})
	e.Report("before retraction")
	e.Retract(f1)
	e.Report("after retraction")
	e.Fire()
	e.Retract(silver)
	e.Insert(Account{"A2", "Gold"})
	e.Report("after account replacement")
	e.Fire()
}
