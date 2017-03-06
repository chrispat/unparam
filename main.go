// Copyright (c) 2017, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/token"
	"go/types"
	"os"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"

	"github.com/kisielk/gotool"
)

func main() {
	flag.Parse()
	warns, err := unusedParams(flag.Args()...)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, warn := range warns {
		fmt.Println(warn)
	}
}

func unusedParams(args ...string) ([]string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	paths := gotool.ImportPaths(args)
	var conf loader.Config
	if _, err := conf.FromArgs(paths, false); err != nil {
		return nil, err
	}
	lprog, err := conf.Load()
	if err != nil {
		return nil, err
	}
	wantPkg := make(map[*types.Package]bool)
	for _, info := range lprog.InitialPackages() {
		wantPkg[info.Pkg] = true
	}
	prog := ssautil.CreateProgram(lprog, 0)
	prog.Build()

	funcSigns := make(map[string]bool)
	addSign := func(t types.Type) {
		sign, ok := t.(*types.Signature)
		if !ok || sign.Params().Len() == 0 {
			return
		}
		funcSigns[signString(sign)] = true
	}
	for _, pkg := range prog.AllPackages() {
		for _, mb := range pkg.Members {
			switch mb.Token() {
			case token.FUNC:
				params := mb.Type().(*types.Signature).Params()
				for i := 0; i < params.Len(); i++ {
					addSign(params.At(i).Type())
				}
				continue
			case token.TYPE:
			default:
				continue
			}
			switch x := mb.Type().Underlying().(type) {
			case *types.Struct:
				for i := 0; i < x.NumFields(); i++ {
					addSign(x.Field(i).Type())
				}
			case *types.Interface:
				for i := 0; i < x.NumMethods(); i++ {
					addSign(x.Method(i).Type())
				}
			case *types.Signature:
				addSign(x)
			}
		}
	}

	var potential []*ssa.Parameter
	for fn := range ssautil.AllFunctions(prog) {
		if fn.Pkg == nil { // builtin?
			continue
		}
		if len(fn.Blocks) == 0 { // stub
			continue
		}
		if !wantPkg[fn.Pkg.Pkg] { // not part of given pkgs
			continue
		}
		if dummyImpl(fn.Blocks[0]) { // panic implementation
			continue
		}
		for i, par := range fn.Params {
			if i == 0 && fn.Signature.Recv() != nil { // receiver
				continue
			}
			switch par.Object().Name() {
			case "", "_": // unnamed
				continue
			}
			if len(*par.Referrers()) > 0 { // used
				continue
			}
			potential = append(potential, par)
		}

	}
	sort.Slice(potential, func(i, j int) bool {
		return potential[i].Pos() < potential[j].Pos()
	})
	warns := make([]string, 0, len(potential))
	for _, par := range potential {
		sign := par.Parent().Signature
		if funcSigns[signString(sign)] { // could be required
			continue
		}
		line := prog.Fset.Position(par.Pos()).String()
		if strings.HasPrefix(line, wd) {
			line = line[len(wd)+1:]
		}
		warns = append(warns, fmt.Sprintf("%s: %s is unused",
			line, par.Name()))
	}
	return warns, nil
}

var rxHarmlessCall = regexp.MustCompile(`(?i)\blog(ger)?\b|\bf?print`)

// dummyImpl reports whether a block is a dummy implementation. This is
// true if the block will almost immediately panic, throw or return
// constants only.
func dummyImpl(blk *ssa.BasicBlock) bool {
	for _, instr := range blk.Instrs {
		switch x := instr.(type) {
		case *ssa.Alloc, *ssa.Store, *ssa.UnOp, *ssa.BinOp,
			*ssa.MakeInterface, *ssa.MakeMap, *ssa.Extract,
			*ssa.IndexAddr, *ssa.FieldAddr, *ssa.Slice,
			*ssa.Lookup, *ssa.ChangeType, *ssa.TypeAssert,
			*ssa.Convert, *ssa.ChangeInterface:
			// non-trivial expressions in panic/log/print
			// calls
		case *ssa.Return:
			for _, val := range x.Results {
				if _, ok := val.(*ssa.Const); !ok {
					return false
				}
			}
			return true
		case *ssa.Panic:
			return true
		case *ssa.Call:
			if rxHarmlessCall.MatchString(x.Call.Value.String()) {
				continue
			}
			return x.Call.Value.Name() == "throw" // runtime's panic
		default:
			return false
		}
	}
	return false
}

func tupleJoin(buf *bytes.Buffer, t *types.Tuple) {
	buf.WriteByte('(')
	for i := 0; i < t.Len(); i++ {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(t.At(i).Type().String())
	}
	buf.WriteByte(')')
}

var signBuf = new(bytes.Buffer)

// signString is similar to Signature.String(), but it ignores
// param/result names.
func signString(sign *types.Signature) string {
	signBuf.Reset()
	tupleJoin(signBuf, sign.Params())
	tupleJoin(signBuf, sign.Results())
	return signBuf.String()
}
