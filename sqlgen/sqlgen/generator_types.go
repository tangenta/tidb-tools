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
	"strings"
)

// ResultType is used to determine whether a Result is valid.
type ResultType int

const (
	// PlainString indicates the result is a plain string
	PlainString ResultType = iota
	// Invalid indicates the result is invalid.
	Invalid
)

// Result stands for the result of Function evaluation.
type Result struct {
	Tp    ResultType
	Value string
}

func InvalidResult() Result {
	return innerInvalidResult
}

var innerInvalidResult = Result{Tp: Invalid}

// InvalidFunc return a functions that returns invalid result.
func InvalidFunc(msg string) func() Result {
	return func() Result {
		return Result{Tp: Invalid, Value: msg}
	}
}

// StrResult returns a PlainString Result.
func StrResult(str string) Result {
	return Result{Tp: PlainString, Value: str}
}

// Fn is a callable object.
type Fn struct {
	Name   string
	F      func() Result
	Weight int
}

func NewFn(name string, fn func() Fn) Fn {
	return Fn{
		Name: name,
		F: func() Result {
			return fn().F()
		},
		Weight: 1,
	}
}

func NewConstFn(name string, fn Fn) Fn {
	return Fn{
		Name: name,
		F: func() Result {
			return fn.F()
		},
		Weight: 1,
	}
}

func (f Fn) SetW(weight int) Fn {
	return Fn{
		Name:   f.Name,
		F:      f.F,
		Weight: weight,
	}
}

// Str is a Fn which simply returns str.
func Str(str string) Fn {
	return Fn{
		Weight: 1,
		F: func() Result {
			return StrResult(str)
		}}
}

func Strf(str string, fns ...Fn) Fn {
	if len(fns) == 0 {
		return Str(str)
	}
	ss := strings.Split(str, "[%fn]")
	if len(ss) != len(fns) + 1 {
		return InvalidFn("[param count mismatched] str: %s", str)
	}
	strs := make([]Fn, 0, 2 * len(ss) - 1)
	for i := 0; i < len(fns); i++ {
		strs = append(strs, Str(ss[i]))
		strs = append(strs, fns[i])
		if i == len(fns) - 1 {
			strs = append(strs, Str(ss[i+1]))
		}
	}
	return And(strs...)
}

func Strs(strs ...string) Fn {
	return Fn{
		Weight: 1,
		F: func() Result {
			return StrResult(strings.Join(strs, " "))
		},
	}
}

// EmptyStringFn is a Fn which simply returns empty string.
func EmptyStringFn() Fn {
	return innerEmptyFn
}

var innerEmptyFn = Fn{
	Weight: 1,
	F: func() Result {
		return Result{Tp: PlainString, Value: ""}
	},
}

func NoneFn() Fn {
	return innerNoneFn
}

var innerNoneFn = Fn{
	Weight: 1,
	Name:   "_$none_fn",
}

func InvalidFn(msg string, params ...interface{}) Fn {
	msg = fmt.Sprintf(msg, params...)
	return Fn{
		F: func() Result {
			return Result{Tp: Invalid, Value: msg}
		},
		Weight: 1,
	}
}
