package main

import (
	"math/rand"
	"reflect"
	"sort"
	"strconv"
	"testing"
)

// Recompute both rules directly from working memory as an independent oracle.
func expected(e *Engine) (map[string]bool, map[string]bool) {
	bonus := make(map[string]bool)
	active := make(map[string]bool)
	for fh, fact := range e.facts {
		f, ok := fact.Value.(Flight)
		if !ok || f.Miles < 500 {
			continue
		}
		active["base/"+strconv.Itoa(fh)] = true
		if f.Airline == "Partner" {
			continue
		}
		for ah, af := range e.facts {
			a, ok := af.Value.(Account)
			if ok && a.Status == "Gold" && a.ID == f.AccountID {
				key := strconv.Itoa(ah) + "/" + strconv.Itoa(fh)
				bonus[key] = true
				active["bonus/"+key] = true
			}
		}
	}
	return bonus, active
}

func check(t *testing.T, e *Engine) {
	t.Helper()
	wantBonus, wantPending := expected(e)
	gotBonus := make(map[string]bool)
	gotPending := make(map[string]bool)
	for k, tuple := range e.bonus.Memory {
		gotBonus[k] = true
		if len(tuple.Facts) != 2 {
			t.Fatalf("invalid tuple %v", tuple)
		}
		for _, fact := range tuple.Facts {
			if !reflect.DeepEqual(e.facts[fact.Handle], fact) {
				t.Fatalf("tuple retains an invalid fact: %v", fact)
			}
		}
	}
	for k := range e.pending {
		gotPending[k] = true
	}
	if !reflect.DeepEqual(gotBonus, wantBonus) {
		t.Fatalf("bonus mismatch: got %v want %v", gotBonus, wantBonus)
	}
	if !reflect.DeepEqual(gotPending, wantPending) {
		t.Fatalf("pending mismatch: got %v want %v", gotPending, wantPending)
	}
}

func TestAllArrivalOrders(t *testing.T) {
	values := []any{Account{"A1", "Gold"}, Flight{"F1", "A1", 500, "Original"},
		Account{"A2", "Gold"}, Flight{"F2", "A3", 900, "Original"}}
	var permute func([]int)
	permute = func(order []int) {
		if len(order) == len(values) {
			e := NewEngine()
			for _, i := range order {
				e.Insert(values[i])
				check(t, e)
			}
			if len(e.bonus.Memory) != 1 {
				t.Fatalf("order %v produced %d matches", order, len(e.bonus.Memory))
			}
			return
		}
		for i := range values {
			used := false
			for _, j := range order {
				used = used || i == j
			}
			if !used {
				permute(append(append([]int(nil), order...), i))
			}
		}
	}
	permute(nil)
}

func TestRandomChangesAgainstFullRecomputation(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	e := NewEngine()
	for step := 0; step < 1000; step++ {
		if len(e.facts) > 0 && rng.Intn(2) == 0 {
			handles := make([]int, 0, len(e.facts))
			for handle := range e.facts {
				handles = append(handles, handle)
			}
			sort.Ints(handles)
			e.Retract(handles[rng.Intn(len(handles))])
		} else if rng.Intn(2) == 0 {
			e.Insert(Account{strconv.Itoa(rng.Intn(5)), []string{"Gold", "Silver"}[rng.Intn(2)]})
		} else {
			e.Insert(Flight{strconv.Itoa(step), strconv.Itoa(rng.Intn(5)),
				[]int{0, 499, 500, 501, 2419}[rng.Intn(5)],
				[]string{"Partner", "Original"}[rng.Intn(2)]})
		}
		check(t, e)
	}
}

func TestCancellationAndMatchLifetime(t *testing.T) {
	e := NewEngine()
	account := Account{"A1", "Gold"}
	a := e.Insert(account)
	account.Status = "Silver" // Changing the caller's copy must not change the fact.
	f := e.Insert(Flight{"F1", "A1", 800, "Original"})
	check(t, e)
	fired := 0
	record := func() {
		for key := range e.pending {
			e.pending[key] = func() { fired++ }
		}
	}
	record()
	e.Retract(f)
	e.Retract(f) // Unknown handles are harmless.
	e.Fire()
	if fired != 0 || len(e.bonus.Memory) != 0 {
		t.Fatal("retracted match fired")
	}
	e.Insert(Flight{"F2", "A1", 800, "Original"})
	record()
	e.Fire()
	e.Fire()
	if fired != 2 || len(e.bonus.Memory) != 1 {
		t.Fatal("match was refired or not retained")
	}
	e.Retract(a)
	if len(e.bonus.Memory) != 0 {
		t.Fatal("account retraction retained dependent tuple")
	}
	e.Insert(Account{"A1", "Gold"})
	if len(e.pending) != 1 {
		t.Fatal("recreated match should activate only the bonus rule")
	}
}

func TestReplacementReevaluatesPredicates(t *testing.T) {
	e := NewEngine()
	a := e.Insert(Account{"A1", "Gold"})
	f := e.Insert(Flight{"F1", "A1", 500, "Original"})
	check(t, e)
	for _, replacement := range []Flight{
		{"F1", "A2", 500, "Original"},
		{"F1", "A1", 499, "Original"},
		{"F1", "A1", 500, "Partner"},
		{"F1", "A1", 500, "Original"},
	} {
		e.Retract(f)
		check(t, e)
		f = e.Insert(replacement)
		check(t, e)
	}
	e.Retract(a)
	a = e.Insert(Account{"A1", "Silver"})
	check(t, e)
	e.Retract(a)
	e.Insert(Account{"A1", "Gold"})
	check(t, e)
}

func TestJoinedSiblingsKeepIndependentTuples(t *testing.T) {
	// Spare capacity exposes accidental writes through a shared tuple slice.
	prefix := make([]Fact, 2, 4)
	prefix[0] = Fact{1, Account{"A1", "Gold"}}
	prefix[1] = Fact{2, Account{"A2", "Silver"}}
	left := &Beta{Memory: make(map[string]Tuple)}
	right := &Alpha{Memory: map[int]Fact{
		3: {3, Flight{"F1", "A1", 500, "Original"}},
		4: {4, Flight{"F2", "A1", 800, "Original"}},
	}}
	joined := &Beta{Memory: make(map[string]Tuple)}
	j := &Join{Left: left, Right: right, Next: joined,
		Test: func(Tuple, Fact) bool { return true }}
	left.Out = append(left.Out, j.LeftEvent)
	tuple := Tuple{"1/2", prefix}
	left.Push(tuple, true)
	if len(joined.Memory) != 2 {
		t.Fatal("expected two distinct joined tuples")
	}
	for handle, fact := range right.Memory {
		got := joined.Memory["1/2/"+strconv.Itoa(handle)].Facts
		want := []Fact{prefix[0], prefix[1], fact}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("joined tuple shares overwritten storage: got %v want %v", got, want)
		}
	}
	left.Push(tuple, false)
	if len(joined.Memory) != 0 {
		t.Fatal("retracted prefix retained dependent joined tuples")
	}
}
