package sample

import (
	"log"
	"math/rand"
	"time"

	. "github.com/pingcap/tidb-tools/sqlgen/sqlgen"
)

// Generate is used to generate a string according to bnf grammar.
var Generate = generate()

func generate() func() string {
	
	start = NamedRule{
		name: "start",
		rule: func() Rule {
			return Or(
				Opt(1, A),
				Opt(1, B),
				/* Custom Rules Here... */
			)
		},
	}

	A = NamedRule{
		name: "A",
		rule: func() Rule {
			return Or(
				Opt(1, Const("a")),
				Opt(1, Const("a"), B),
				/* Custom Rules Here... */
			)
		},
	}

	B = NamedRule{
		name: "B",
		rule: func() Rule {
			return SelfRec(
				Range{0, 255},
				[]OrOpt{
					Opt(1, Const("b")),
					/* Custom Rules Here... */
				},
				[]func(Rule) OrOpt{
					func(_r Rule) OrOpt { return Opt(1, A, _r) },
					/* Custom Rules Here... */
				},
			)
		},
	}

	rand.Seed(time.Now().UnixNano())
	retFn := func() string {
		if res, ok := start.Gen(); ok {
			return res
		} else {
			log.Println("Invalid SQL")
			return ""
		}
	}

	return retFn
}
