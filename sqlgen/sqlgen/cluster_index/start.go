package cluster_index

import (
	"log"
	"math/rand"
	"time"

	. "github.com/pingcap/tidb-tools/sqlgen/sqlgen"
)

func NewGenerator(state *State) func() string {
	rand.Seed(time.Now().UnixNano())
	retFn := func() string {
		res := start.F()
		switch res.Tp {
		case PlainString:
			return res.Value
		case Invalid:
			log.Println("Invalid SQL")
			return ""
		default:
			log.Fatalf("Unsupported result type '%v'", res.Tp)
			return ""
		}
	}

	start = NewFn("start", func() Fn {
		return Or(
			switchSysVars,
			If(len(state.tables) < state.ctrl.MaxTableNum,
				createTable,
			).SetW(2),
			If(len(state.tables) > 0,
				insertInto,
			),
			If(len(state.tables) > 0,
				query,
			),
		)
	})

	switchSysVars = NewFn("switchSysVars", func() Fn {
		return Or(
			Str("set @@global.tidb_row_format_version = 2;"),
			Str("set @@global.tidb_row_format_version = 1;"),
			Str("set @@tidb_enable_clustered_index = 0;"),
			Str("set @@tidb_enable_clustered_index = 1;"),
		)
	})

	createTable = NewFn("createTable", func() Fn {
		state.NewTable()
		return And(
			Str("create table"),
			Str(state.GenTableName()),
			Str("("),
			definitions,
			Str(");"),
		)
	})

	definitions = NewFn("definitions", func() Fn {
		colDefs = NewFn("colDefs", func() Fn {
			return Or(
				colDef,
				And(colDef, Str(","), colDefs).SetW(2),
			)
		})
		colDef = NewFn("colDefs", func() Fn {
			state.NewColumn()
			return And(Str(state.GenColumnName()), Str(state.GenColTypes()))
		})
		idxDefs = NewFn("idxDefs", func() Fn {
			return Or(
				idxDef,
				And(idxDef, Str(","), idxDefs).SetW(2),
			)
		})
		idxDef = NewFn("idxDef", func() Fn {
			state.NewIndex()
			return And(
				Str(state.GenIndexTypeString()),
				Str("key"),
				Str(state.GenIndexName()),
				Str("("),
				Str(state.GenIndexColumns()),
				Str(")"),
			)
		})
		return Or(
			And(colDefs, Str(","), idxDefs).SetW(4),
			colDefs,
		)
	})

	insertInto = NewFn("insertInto", func() Fn {
		return And(
			Str("insert into"),
			Str(state.GetRandTableName()),
			Str(state.GetRandColumns("")),
			Str("values"),
			Str("("),
			Str(state.GenRandValue()),
			Str(")"),
		)
	})

	query = NewFn("query", func() Fn {
		commonSelect = NewFn("commonSelect", func() Fn {
			tbl := state.GetRandTableName()
			return And(Str("select"),
				Str(state.GetRandColumns("*")),
				Str("from"),
				Str(tbl),
				Str("where"),
				predicate)
		})
		predicate = NewFn("predicate", func() Fn {
			return Or(
				And(Str(state.GetRandColumn()), Str("="), Str(state.GenRandValue())),
				And(Str(state.GetRandColumn()), Str("in"), Str("("), randColVals, Str(")")),
			)
		})

		return commonSelect
	})

	randColVals = NewFn("randColVals", func() Fn {
		return Or(
			Str(state.GenRandValue()),
			And(Str(state.GenRandValue()), Str(","), Str(state.GenRandValue())),
		)
	})

	return retFn
}
