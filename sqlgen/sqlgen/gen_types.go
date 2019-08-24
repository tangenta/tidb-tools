package sqlgen

import (
	"math/rand"
	"strings"
)

type Rule interface {
	Gen() (res string, ok bool)
}

type RuleImpl struct {
	fn func() (res string, ok bool)
}

func (i RuleImpl) Gen() (res string, ok bool) {
	return i.fn()
}

func Cat(rules ...Rule) Rule {
	return RuleImpl{func() (res string, ok bool) {
		var ss []string
		for _, rule := range rules {
			if s, ok1 := rule.Gen(); ok1 {
				ss = append(ss, s)
			} else {
				ok = false
				return
			}
		}
		return strings.Join(ss, " "), true
	}}
}

type OrOpt struct {
	Frequency int
	Opt       Rule
}

func Opt(frequency int, opts ...Rule) OrOpt {
	return OrOpt{Frequency: frequency, Opt: Cat(opts...)}
}

func (o OrOpt) Gen() (res string, ok bool) {
	return o.Opt.Gen()
}

func Or(rules ...OrOpt) Rule {
	return RuleImpl{func() (res string, ok bool) {
		rules := rules
		for len(rules) > 0 && !ok {
			fSum := 0
			for _, r := range rules {
				fSum += r.Frequency
			}
			pivot := rand.Intn(fSum)
			var idx int
			for i, rule := range rules {
				if pivot-rule.Frequency < 0 {
					idx = i
					break
				} else {
					pivot -= rule.Frequency
				}
			}
			res, ok = rules[idx].Gen()
			rules = append(rules[:idx], rules[idx+1:]...)
		}
		return
	}}
}

type Range struct {
	Low, High int
}

/**
  Self Recursive BNF Generator Definition

    moreThan,
    lessThan: the limit of rec-definition call times.
        Times of rec-definition call will be [moreThan, lessThan].
        When moreThan == 0 and lessThan == -1, will no longer limit
        the call times.

    tmls: Sequence of normal definitions.
    recs: Sequence of recursive definitions.
        A recursive definition is a function, accepting an Opt which
        represents a placeholder of self call.
*/
func SelfRec(rge Range, tmls []OrOpt, recs []func(rule Rule) OrOpt) Rule {
	var r func(cur int) Rule
	r = func(cur int) Rule {
		return RuleImpl{func() (res string, ok bool) {
			var rs []OrOpt
			if rge.Low <= cur && rge.High-cur != 0 {
				for _, rc := range recs {
					rs = append(rs, rc(r(cur+1)))
				}
				return Or(append(tmls, rs...)...).Gen()
			} else if cur < rge.Low {
				for _, rc := range recs {
					rs = append(rs, rc(r(cur+1)))
				}
				return Or(rs...).Gen()
			} else {
				return Or(tmls...).Gen()
			}
		}}
	}

	return r(1)
}

type Const string

func (c Const) Gen() (res string, ok bool) {
	return string(c), true
}
