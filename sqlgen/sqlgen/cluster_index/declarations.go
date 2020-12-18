package cluster_index

import (
	. "github.com/pingcap/tidb-tools/sqlgen/sqlgen"
)

var (
	start         Fn
	switchSysVars Fn
	createTable   Fn
	definitions   Fn
	colDefs       Fn
	colDef        Fn
	idxDefs       Fn
	idxDef        Fn
	insertInto    Fn
	query         Fn
	commonSelect  Fn
	predicate Fn
	randColVals Fn
)