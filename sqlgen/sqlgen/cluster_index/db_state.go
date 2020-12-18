package cluster_index

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	"github.com/cznic/mathutil"
	. "github.com/pingcap/tidb-tools/sqlgen/sqlgen"
)

type State struct {
	ctrl *ControlOption

	tables []*Table

	curTable     *Table
	curColumn    *Column
	curIndex     *Index
	selectedCols []*Column

	globalTableIdx int
	globalColumnIdx int
	globalIndexIdx int
}

type Table struct {
	name string
	columns []*Column
	indices []*Index

	containsPK bool // to ensure at most 1 pk in each table
	values [][]string
}

type Column struct {
	name string
	tp ColumnType

	isUnsigned bool
	arg1 int // optional
	arg2 int // optional
	args []string // for ColumnTypeSet and ColumnTypeEnum
}

type Index struct {
	name string
	tp IndexType
	columns []*Column
	columnPrefix []int
}

func NewState() *State {
	return &State{
		ctrl: DefaultControlOption(),
		globalTableIdx:  -1,
		globalColumnIdx: -1,
		globalIndexIdx:  -1,
	}
}

func (s *State) UpdateCtrlOption(fn func(option *ControlOption)) {
	fn(s.ctrl)
}

func (s *State) NewTable() {
	s.curTable = &Table{}
	s.tables = append(s.tables, s.curTable)
}

func (s *State) GenTableName() string {
	s.globalTableIdx++
	tblName := fmt.Sprintf("tbl_%d", s.globalTableIdx)
	s.curTable.name = tblName
	return tblName
}

func (s *State) NewColumn() {
	s.curColumn = &Column{}
	s.curTable.columns = append(s.curTable.columns, s.curColumn)
}

func (s *State) GenColumnName() string {
	s.globalColumnIdx++
	colName := fmt.Sprintf("col_%d", s.globalColumnIdx)
	s.curColumn.name = colName
	return colName
}

func (s *State) NewIndex() {
	s.curIndex = &Index{}
	s.curTable.indices = append(s.curTable.indices, s.curIndex)
}

func (s *State) GenIndexTypeString() string {
	maxVal := int(IndexTypeMax)
	if s.curTable.containsPK {
		maxVal = int(IndexTypePrimary)
	}
	tp := IndexType(rand.Intn(maxVal))
	s.curIndex.tp = tp
	switch tp {
	case IndexTypeNonUnique:
		return ""
	case IndexTypeUnique:
		return "unique"
	case IndexTypePrimary:
		s.curTable.containsPK = true
		return "primary"
	default:
		return "invalid"
	}
}

func (s *State) GenIndexName() string {
	s.globalIndexIdx++
	idxName := fmt.Sprintf("idx_%d", s.globalIndexIdx)
	s.curIndex.name = idxName
	return idxName
}

func (s *State) GenIndexColumns() string {
	totalCols := s.curTable.columns
	for {
		chosenIdx := rand.Intn(len(totalCols))
		chosenCol := totalCols[chosenIdx]
		totalCols[0], totalCols[chosenIdx] = totalCols[chosenIdx], totalCols[0]
		totalCols = totalCols[1:]

		s.curIndex.columns = append(s.curIndex.columns, chosenCol)
		prefixLen := 0
		if chosenCol.tp.IsStringType() && rand.Intn(4) == 0 {
			prefixLen = rand.Intn(5)
		}
		s.curIndex.columnPrefix = append(s.curIndex.columnPrefix, prefixLen)
		if len(totalCols) == 0 || RandomBool() {
			break
		}
	}
	var sb strings.Builder
	for i, col := range s.curIndex.columns {
		sb.WriteString(col.name)
		if s.curIndex.columnPrefix[i] != 0 {
			sb.WriteString("(")
			sb.WriteString(strconv.Itoa(s.curIndex.columnPrefix[i]))
			sb.WriteString(")")
		}
		if i != len(s.curIndex.columns)-1 {
			sb.WriteString(",")
		}
	}
	return sb.String()
}

func (s *State) GenColTypes() string {
	col := s.curColumn
	col.tp = ColumnType(rand.Intn(int(ColumnTypeMax)))
	switch col.tp {
	// https://docs.pingcap.com/tidb/stable/data-type-numeric
	case ColumnTypeFloat|ColumnTypeDouble:
		col.arg1 = rand.Intn(256)
		upper := mathutil.Min(col.arg1, 30)
		col.arg2 = rand.Intn(upper+1)
	case ColumnTypeDecimal:
		col.arg1 = rand.Intn(66)
		upper := mathutil.Min(col.arg1, 30)
		col.arg2 = rand.Intn(upper+1)
	case ColumnTypeBit:
		col.arg1 = 1 + rand.Intn(64)
	case ColumnTypeChar,ColumnTypeVarchar,ColumnTypeText,ColumnTypeBlob,ColumnTypeBinary:
		col.arg1 = 1 + rand.Intn(4294967295)
	case ColumnTypeEnum,ColumnTypeSet:
		col.args = []string{"Alice", "Bob", "Charlie", "David"}
	}
	if col.tp.IsIntegerType() {
		col.isUnsigned = RandomBool()
	}
	return ColumnTypeString(col)
}

func ColumnTypeString(c *Column) string {
	var sb strings.Builder
	sb.WriteString(c.tp.String())
	if c.arg1 != 0 {
		sb.WriteString("(")
		sb.WriteString(strconv.Itoa(c.arg1))
		if c.arg2 != 0 {
			sb.WriteString(",")
			sb.WriteString(strconv.Itoa(c.arg2))
		}
		sb.WriteString(")")
	} else if len(c.args) != 0 {
		sb.WriteString("(")
		for i, a := range c.args {
			sb.WriteString("'")
			sb.WriteString(a)
			sb.WriteString("'")
			if i != len(c.args)-1 {
				sb.WriteString(",")
			}
		}
		sb.WriteString(")")
	}
	if c.isUnsigned {
		sb.WriteString(" unsigned")
	}
	return sb.String()
}

func (s *State) GetRandTableName() string {
	randTbl := s.tables[rand.Intn(len(s.tables))]
	s.curTable = randTbl
	return randTbl.name
}

func (s *State) GetRandColumns(defau1t string) string {
	if RandomBool() {
		// insert into t values (...)
		s.selectedCols = s.curTable.columns
		return defau1t
	}
	// insert into t (cols..) values (...)
	totalCols := s.curTable.columns
	for {
		chosenIdx := rand.Intn(len(totalCols))
		chosenCol := totalCols[chosenIdx]
		totalCols[0], totalCols[chosenIdx] = totalCols[chosenIdx], totalCols[0]
		totalCols = totalCols[1:]

		s.selectedCols = append(s.selectedCols, chosenCol)
		if len(totalCols) == 0 || RandomBool() {
			break
		}
	}
	var sb strings.Builder
	sb.WriteString("(")
	for i, c := range s.selectedCols {
		sb.WriteString(c.name)
		if i != len(s.selectedCols)-1 {
			sb.WriteString(",")
		}
	}
	sb.WriteString(")")
	return sb.String()
}

func (s *State) GetRandColumn() string {
	totalCols := s.curTable.columns
	col := totalCols[rand.Intn(len(totalCols))]
	s.selectedCols = []*Column{col}
	return col.name
}

func (s *State) GenRandValue() string {
	var sb strings.Builder
	var row []string
	for i, c := range s.selectedCols {
		v := c.RandomValue()
		row = append(row, v)
		sb.WriteString(v)
		if i != len(s.selectedCols)-1 {
			sb.WriteString(",")
		}
	}
	s.curTable.values = append(s.curTable.values, row)
	sb.WriteString("")
	return sb.String()
}
