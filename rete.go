package main

import (
	"fmt"
	"sort"
	"strconv"
)

// Fact is an immutable value snapshot identified by an insertion handle.
type Fact struct {
	Handle int
	Value  any
}

// Tuple retains ordered fact identities for a rule prefix.
type Tuple struct {
	Key   string
	Facts []Fact
}

func singleton(f Fact) Tuple {
	return Tuple{strconv.Itoa(f.Handle), []Fact{f}}
}

// Copy the prefix so sibling results never share a writable backing array.
func extend(t Tuple, f Fact) Tuple {
	facts := append([]Fact(nil), t.Facts...)
	return Tuple{t.Key + "/" + strconv.Itoa(f.Handle), append(facts, f)}
}

// Alpha combines a single-fact filter with its retained passing facts.
type Alpha struct {
	Test   func(any) bool
	Memory map[int]Fact
	Out    []func(Fact, bool)
}

func (a *Alpha) Push(f Fact, add bool) {
	if add {
		if !a.Test(f.Value) {
			return
		}
		if _, exists := a.Memory[f.Handle]; exists {
			return
		}
		a.Memory[f.Handle] = f
	} else {
		stored, exists := a.Memory[f.Handle]
		if !exists {
			return
		}
		f = stored
		delete(a.Memory, f.Handle)
	}
	for _, out := range a.Out {
		out(f, add)
	}
}

// Beta retains tuples and propagates insertions and removals to consumers.
type Beta struct {
	Memory map[string]Tuple
	Out    []func(Tuple, bool)
}

func (b *Beta) Push(t Tuple, add bool) {
	if add {
		if _, exists := b.Memory[t.Key]; exists {
			return
		}
		b.Memory[t.Key] = t
	} else {
		stored, exists := b.Memory[t.Key]
		if !exists {
			return
		}
		t = stored
		delete(b.Memory, t.Key)
	}
	for _, out := range b.Out {
		out(t, add)
	}
}

// Join extends a left tuple with a right fact satisfying a pure predicate.
type Join struct {
	Left  *Beta
	Right *Alpha
	Test  func(Tuple, Fact) bool
	Next  *Beta
}

func (j *Join) LeftEvent(t Tuple, add bool) {
	for _, f := range j.Right.Memory {
		if j.Test(t, f) {
			j.Next.Push(extend(t, f), add)
		}
	}
}

func (j *Join) RightEvent(f Fact, add bool) {
	for _, t := range j.Left.Memory {
		if j.Test(t, f) {
			j.Next.Push(extend(t, f), add)
		}
	}
}

// Engine owns working memory and the manually wired demonstration network.
type Engine struct {
	next                  int
	facts                 map[int]Fact
	gold, eligible, other *Alpha
	left, bonus           *Beta
	pending               map[string]func()
}

// Insert allocates a fresh handle, even when business values are identical.
func (e *Engine) Insert(v any) int {
	switch v.(type) {
	case Account, Flight:
	default:
		panic("unsupported fact type")
	}
	e.next++
	f := Fact{e.next, v}
	e.facts[f.Handle] = f
	switch v.(type) {
	case Account:
		e.gold.Push(f, true)
	case Flight:
		e.eligible.Push(f, true)
	}
	return f.Handle
}

// Retract invalidates dependent tuples and cancels their pending actions.
func (e *Engine) Retract(handle int) {
	f, exists := e.facts[handle]
	if !exists {
		return
	}
	delete(e.facts, handle)
	switch f.Value.(type) {
	case Account:
		e.gold.Push(f, false)
	case Flight:
		e.eligible.Push(f, false)
	}
}

// Fire executes pending actions in key order for reproducible demo output.
func (e *Engine) Fire() {
	for len(e.pending) > 0 {
		keys := make([]string, 0, len(e.pending))
		for key := range e.pending {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if action, exists := e.pending[key]; exists {
				delete(e.pending, key)
				action()
			}
		}
	}
}

func (e *Engine) Report(label string) {
	fmt.Printf("%s: eligible=%d nonPartner=%d gold=%d bonus=%d pending=%d\n",
		label, len(e.eligible.Memory), len(e.other.Memory),
		len(e.left.Memory), len(e.bonus.Memory), len(e.pending))
}
