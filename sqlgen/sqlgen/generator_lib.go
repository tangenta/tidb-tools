// Copyright 2019 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package sqlgen

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
)

var GenPlugins []Plugin

type Plugin interface{}

type ProductionListener interface {
	Plugin
	BeforeProductionGen(fn *Fn)
	AfterProductionGen(fn *Fn, result *Result)
	ProductionCancel(fn *Fn)
}

func And(fn ...Fn) Fn {
	return Fn{Weight: 1, F: func() Result {
		return collectResult(fn...)
	}}
}

func Opt(fn Fn) Fn {
	if RandomBool() {
		return fn
	}
	return EmptyStringFn()
}

func RandomNum(low, high int64) string {
	num := rand.Int63n(high - low + 1)
	return strconv.FormatInt(num + low, 10)
}

func RandomFloat(low, high float64) string {
	f := low + rand.Float64() * (high - low)
	return fmt.Sprintf("%f", f)
}

func RandomBool() bool {
	return rand.Intn(2) == 0
}

func If(condition bool, fn Fn) Fn {
	if condition {
		return fn
	}
	return NoneFn()
}

func Or(fns ...Fn) Fn {
	fns = filterNoneFns(fns)
	return Fn{Weight: 1, F: func() Result {
		for len(fns) > 0 {
			randNum := randomSelectByFactor(fns)
			chosenFn := fns[randNum]
			rs := evaluateFn(chosenFn)
			if rs.Tp == PlainString {
				return rs
			}
			fns[0], fns[randNum] = fns[randNum], fns[0]
			fns = fns[1:]
		}
		return InvalidResult()
	}}
}

func Repeat(fn Fn, cnt int) Fn {
	if cnt == 0 {
		return EmptyStringFn()
	}
	fns := make([]Fn, 0, cnt)
	for i := 0; i < cnt; i++ {
		fns[i] = fn
	}
	return And(fns...)
}

func RepeatRange(low, high int, fn Fn) Fn {
	return Repeat(fn, low + rand.Intn(high - low))
}

func Join(sep Fn, fns ...Fn) Fn {
	newFns := make([]Fn, 0, len(fns)*2-1)
	for i, f := range fns {
		newFns = append(newFns, f)
		if i != len(fns) - 1 {
			newFns = append(newFns, sep)
		}
	}
	return And(newFns...)
}

func filterNoneFns(fns []Fn) []Fn {
	for i := 0; i < len(fns); i++ {
		if fns[i].Name == NoneFn().Name {
			fns[i], fns[len(fns)-1] = fns[len(fns)-1], fns[i]
			fns = fns[:len(fns)-1]
		}
	}
	return fns
}

func collectResult(fns ...Fn) Result {
	var doneF []Fn
	var resStr strings.Builder
	for i, f := range fns {
		res := evaluateFn(f)
		switch res.Tp {
		case PlainString:
			doneF = append(doneF, f)
			resStr.WriteString(strings.Trim(res.Value, " "))
			if i != len(fns) {
				resStr.WriteString(" ")
			}
		case Invalid:
			for _, df := range doneF {
				forEachProdListener(func(p ProductionListener) {
					p.ProductionCancel(&df)
				})
			}
			return InvalidResult()
		default:
			log.Fatalf("Unsupport result type '%v'", res.Tp)
		}
	}
	return StrResult(resStr.String())
}

func evaluateFn(fn Fn) Result {
	if len(fn.Name) == 0 {
		return fn.F()
	}
	forEachProdListener(func(p ProductionListener) {
		p.BeforeProductionGen(&fn)
	})
	res := fn.F()
	forEachProdListener(func(p ProductionListener) {
		p.AfterProductionGen(&fn, &res)
	})
	return res
}

func randomSelectByFactor(fns []Fn) int {
	num := rand.Intn(sumRandFactor(fns))
	acc := 0
	for i, f := range fns {
		acc += f.Weight
		if acc > num {
			return i
		}
	}
	return len(fns) - 1
}

func forEachProdListener(fn func(ProductionListener)) {
	for _, p := range GenPlugins {
		if lp, ok := p.(ProductionListener); ok {
			fn(lp)
		}
	}
}

func sumRandFactor(fs []Fn) int {
	total := 0
	for _, f := range fs {
		total += f.Weight
	}
	return total
}
