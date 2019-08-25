package sqlgen

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// BuildLib is used to build a random generator library.
func BuildLib(yaccFilePath, prodName, packageName, outputFilePath string) {
	yaccFilePath = absolute(yaccFilePath)
	outputFilePath = absolute(filepath.Join(outputFilePath, packageName))
	prods, err := ParseYacc(yaccFilePath)
	if err != nil {
		log.Fatal(err)
	}
	prodMap := BuildProdMap(prods)

	Must(os.Mkdir(outputFilePath, 0755))
	Must(os.Chdir(outputFilePath))

	allProds := writeGenerate(prodName, prodMap, packageName)
	writeUtil(packageName)
	writeDeclarations(allProds, packageName)
	writeLimit(allProds, packageName)
	writeTest(packageName)
}

func writeGenerate(prodName string, prodMap map[string]*Production, packageName string) map[string]struct{} {
	var allProds map[string]struct{}
	openAndWrite(prodName+".go", packageName, func(w *bufio.Writer) {
		var sb strings.Builder
		visitor := func(p *Production) {
			sb.WriteString(convertProdToCode(p))
		}
		ps, err := breadthFirstSearch(prodName, prodMap, visitor)
		allProds = ps
		if err != nil {
			log.Fatal(err)
		}
		MustWrite(w, fmt.Sprintf(templateMain, sb.String(), prodName))
	})
	return allProds
}

func writeUtil(packageName string) {
	openAndWrite("util.go", packageName, func(w *bufio.Writer) {
		MustWrite(w, utilSnippet)
	})
}

func writeDeclarations(allProds map[string]struct{}, packageName string) {
	openAndWrite("declarations.go", packageName, func(w *bufio.Writer) {
		MustWrite(w, `
import (
	. "github.com/pingcap/tidb-tools/sqlgen/sqlgen"
)

`)
		for p := range allProds {
			p = convertHead(p)
			MustWrite(w, fmt.Sprintf("var %s Rule\n", p))
		}
	})
}

func writeLimit(allProds map[string]struct{}, packageName string) {
	openAndWrite("limits.go", packageName, func(w *bufio.Writer) {
		var sb strings.Builder
		for p := range allProds {
			p = convertHead(p)
			sb.WriteString(fmt.Sprintf("\t\"%s\": 127,\n", p))
		}
		sb.WriteString("\t/* Custom Limits Here... */")
		MustWrite(w, fmt.Sprintf(templateLimit, sb.String()))
	})
}

func writeTest(packageName string) {
	openAndWrite(packageName+"_test.go", packageName, func(w *bufio.Writer) {
		MustWrite(w, testSnippet)
	})
}


func openAndWrite(path string, pkgName string, doWrite func(*bufio.Writer)) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	writer := bufio.NewWriter(file)
	MustWrite(writer, fmt.Sprintf("package %s\n", pkgName))
	doWrite(writer)
	writer.Flush()
}

const templateMain = `
import (
	"log"
	"math/rand"
	"time"

	. "github.com/pingcap/tidb-tools/sqlgen/sqlgen"
)

// Generate is used to generate a string according to bnf grammar.
var Generate = generate()

func generate() func() string {
	%s
	rand.Seed(time.Now().UnixNano())
	retFn := func() string {
		if res, ok := %s.Gen(); ok {
			return res
		} else {
			log.Println("Invalid SQL")
			return ""
		}
	}

	return retFn
}
`

const templateLimit = `
var counter = map[string]int{
%s
}
`

const utilSnippet = `
import (
	. "github.com/pingcap/tidb-tools/sqlgen/sqlgen"
)

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
`

const templateO = `
	%s = NamedRule{
		name: "%s",
		rule: func() Rule {
			return Or(
				%s
			)
		},
	}
`

const templateR = `
	%s = NamedRule{
		name: "%s",
		rule: func() Rule {
			return SelfRec(
				Range{0, 255},
				[]OrOpt{
					%s
				},
				[]func(Rule) OrOpt{
					%s
				},
			)
		},
	}
`

const templateS = `
	%s = NamedRule{
		name: "%s",
		rule: func() Rule {
			return Const("%s")
		},
	}
`

func convertProdToCode(p *Production) string {
	prodHead := convertHead(p.head)
	if len(p.bodyList) == 1 {
		allLiteral := true
		seqs := p.bodyList[0].seq
		for _, s := range seqs {
			if !isLiteral(s) {
				allLiteral = false
				break
			}
		}

		trimmedSeqs := trimmedStrs(seqs)
		if allLiteral {
			return fmt.Sprintf(templateS, prodHead, prodHead, strings.Join(trimmedSeqs, " "))
		}
	}

	if p.isSelfRec {
		return buildSelfRecProd(prodHead, p)
	} else {
		return buildNormProd(prodHead, p)
	}
}

func buildSelfRecProd(ph string, p *Production) string {
	var nor strings.Builder
	var rec strings.Builder
	for _, body := range p.bodyList {
		if body.isSelfRec {
			rec.WriteString(fmt.Sprintf("func(_r Rule) OrOpt { return Opt(1%s) },\n\t\t\t\t\t",
				consRecBody(ph, &body)))
		} else {
			nor.WriteString(fmt.Sprintf("Opt(1%s),\n\t\t\t\t\t", consNormBody(&body)))
		}
	}
	nor.WriteString("/* Custom Rules Here... */")
	rec.WriteString("/* Custom Rules Here... */")

	return fmt.Sprintf(templateR, ph, p.head, nor.String(), rec.String())
}

func buildNormProd(ph string, p *Production) string {
	var bodyListStr strings.Builder
	for _, body := range p.bodyList {
		bodyListStr.WriteString(fmt.Sprintf("Opt(1%s),\n\t\t\t\t", consNormBody(&body)))
	}
	bodyListStr.WriteString("/* Custom Rules Here... */")

	return fmt.Sprintf(templateO, ph, p.head, bodyListStr.String())
}

func consNormBody(body *Body) string {
	var bodyStr strings.Builder
	for _, s := range body.seq {
		if isLit, ok := literal(s); ok {
			bodyStr.WriteString(fmt.Sprintf(", Const(\"%s\")", isLit))
		} else {
			bodyStr.WriteString(fmt.Sprintf(", %s", convertHead(s)))
		}
	}
	return bodyStr.String()
}

func consRecBody(hd string, body *Body) string {
	var bodyStr strings.Builder
	for _, s := range body.seq {
		if isLit, ok := literal(s); ok {
			bodyStr.WriteString(fmt.Sprintf(", Const(\"%s\")", isLit))
		} else {
			s = convertHead(s)
			if s == hd {
				bodyStr.WriteString(", _r")
			} else {
				bodyStr.WriteString(fmt.Sprintf(", %s", s))
			}
		}
	}
	return bodyStr.String()
}

func trimmedStrs(origin []string) []string {
	ret := make([]string, len(origin))
	for i, s := range origin {
		if lit, ok := literal(s); ok {
			ret[i] = lit
		}
	}
	return ret
}

// convertHead to avoid keyword clash.
func convertHead(str string) string {
	if strings.HasPrefix(str, "$@") {
		return "num" + strings.TrimPrefix(str, "$@")
	}

	switch str {
	case "type":
		return "utype"
	case "%empty":
		return "empty"
	default:
		return str
	}
}

func MustWrite(oFile *bufio.Writer, str string) {
	_, err := oFile.WriteString(str)
	if err != nil {
		log.Fatal(err)
	}
}

func Must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func absolute(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		log.Fatal(err)
	}
	return abs
}

const testSnippet = `
import (
	"fmt"
	"testing"
)

func TestA(t *testing.T) {
	for i := 0; i < 10; i++ {
		fmt.Println(Generate())
	}
}
`
