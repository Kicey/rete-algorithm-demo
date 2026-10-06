# Rete algorithm demo in Go

A small, manually wired Rete matcher demonstrating shared alpha filters,
two-sided tuple joins, fact identity, retraction, and a separate agenda.
Only the Go standard library is used. The module requires Go 1.25.4 or later.

## Run the example

From this directory:

```bash
go run .
```

`go run .` compiles and executes the package in the current directory without
installing a binary. The program keeps state in memory and prints reward
proposals; it does not write account balances or contact external services.
Go may write to its build cache.

```text
before retraction: eligible=3 nonPartner=2 gold=1 bonus=1 pending=4
after retraction: eligible=2 nonPartner=1 gold=1 bonus=0 pending=2
base F2: 800 miles
base F4: 900 miles
after account replacement: eligible=2 nonPartner=1 gold=2 bonus=1 pending=1
bonus A2/F2: 800 miles
```

## Rules and network

Facts are `Account{ID, Status}` and `Flight{ID, AccountID, Miles, Airline}`.

- **BaseMiles:** a flight has at least 500 miles; propose its base reward.
- **GoldBonus:** the account is Gold, the flight has at least 500 miles, its
  airline is not Partner, and its `AccountID` equals the account's `ID`;
  propose an additional reward equal to the flight's miles.

```text
Account --> [Status == Gold] --> M_G --> [tuple adapter] --> B_G --+
                                                               | left
                                                               v
                                                             [Join] --> B_B --> T_bonus
                                                               ^
                                                               | right
Flight --> [Miles >= 500] --> M_E --> [Airline != Partner] --> M_N-+
                              |
                              +--> T_base
```

`M_G`, `M_E`, and `M_N` retain filtered facts. `B_G` retains one-account tuples;
`B_B` retains account–flight tuples. Both rules share the miles filter. The
join compares a newly inserted input with the retained opposite side, so
either the account or the flight can arrive first.

| File | Role |
| --- | --- |
| [main.go](main.go) | Fact types, rule predicates, graph construction, actions, and the input sequence |
| [rete.go](rete.go) | Fact handles, tuples, alpha/beta memories, join propagation, working-memory operations, and agenda execution |
| [rete_test.go](rete_test.go) | An independent naïve matcher and behavioral tests |

## Fact and activation lifetime

`Insert(value)` returns a fresh handle. It retains a value snapshot; supported
facts contain only strings and integers. Business IDs define ownership joins,
while handles identify individual assertions. Equal values inserted twice are
distinct facts. Enforce business-ID uniqueness in the application if required.

`Retract(handle)` removes the fact, dependent tuples, and pending activations.
An unknown or already retracted handle is a no-op. To modify a fact, retract
its old handle and insert a replacement snapshot. The example replaces the
Silver `A2` account with Gold, allowing its already retained flight to match.

`Fire()` executes pending actions in sorted activation-key order. Matched
tuples remain retained afterward, so repeated calls do not repeat the same
activation. Retracting and reinserting a fact starts a new match lifetime and
can activate rules again. Retraction does not undo actions already executed.

## Check correctness

```bash
go test ./...
go vet ./...
```

`./...` selects this package and any packages below it. `go test` compiles and
runs the tests; `go vet` checks for suspicious Go constructs without running
the demonstration. Both may use Go's build cache.

The tests compare current joins and pending activations with an independent
full recomputation after every change. They cover all 24 arrival permutations
of four facts, 1,000 seeded insert/retract operations, predicate boundaries,
unrelated accounts, replacement snapshots, cancellation before execution,
repeated firing, and independent storage for sibling tuples.

## Scope

The graph is constructed in code rather than compiled from a rule language.
Only positive conditions are supported. Joins and retractions scan the
opposite memory; there are no join indexes or reverse dependency links.
The demonstration uses sorted agenda keys rather than rule priorities and
prints proposals rather than applying business side effects. It has no truth
maintenance or concurrent access support.

Retaining matches reduces repeated work across changes but costs memory.
For one join, a new right fact checks all retained left tuples; output tuples
can still grow as the product of both input sizes.

## References

- Charles L. Forgy, [Rete: A Fast Algorithm for the Many Pattern/Many Object Pattern Match Problem](https://doi.org/10.1016/0004-3702(82)90020-0), 1982.
- [Drools 6.5: Rete and ReteOO](https://docs.drools.org/6.5.0.Final/drools-docs/html/ch05.html).
- [Drools 6.5: working memory and agenda](https://docs.drools.org/6.5.0.Final/drools-docs/html/ch07.html).
