package sample

import (
	. "github.com/pingcap/tidb-tools/sqlgen/sqlgen"
)

var counter map[string]int

type NamedRule struct {
	name string
	rule func() Rule
}

func (nr NamedRule) Gen() (res string, ok bool) {
	if r, ok1 := counter[nr.name]; ok1 {
		if r <= 0 {
			ok = false
			return
		} else {
			counter[nr.name] -= 1
			return nr.rule().Gen()
		}
	} else {
		return nr.rule().Gen()
	}
}