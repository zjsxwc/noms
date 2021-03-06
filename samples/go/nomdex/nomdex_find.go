// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"
	"io"
	"os"

	"github.com/attic-labs/noms/cmd/util"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/outputpager"
	flag "github.com/juju/gnuflag"
)

var find = &util.Command{
	Run:       runFind,
	UsageLine: "find --db <database spec> <query>",
	Short:     "Print objects from prebuilt indexes",
	Long:      "Print object from prebuild indexes (long desc)",
	Flags:     setupFindFlags,
	Nargs:     1,
}

func setupFindFlags() *flag.FlagSet {
	flagSet := flag.NewFlagSet("find", flag.ExitOnError)
	flagSet.StringVar(&indexPath, "index", "", "dataset containing index")
	outputpager.RegisterOutputpagerFlags(flagSet)
	return flagSet
}

func runFind(args []string) int {
	query := args[0]
	if indexPath == "" {
		fmt.Fprintf(os.Stderr, "Missing required 'index' arg\n")
		flag.Usage()
		return 1
	}

	db, index, err := openIndex(indexPath)
	if printError(err, "Unable to open database/index\n\terror: ") {
		return 1
	}
	defer db.Close()

	expr, err := parseQuery(query)
	if err != nil {
		fmt.Printf("err: %s\n", err)
		return 1
	}

	pgr := outputpager.Start()
	defer pgr.Stop()

	ranges := expr.ranges()
	printObjects(pgr.Writer, index, ranges)

	return 0
}

func printObjects(w io.Writer, index types.Map, ranges queryRangeSlice) {
	cnt := 0
	first := true
	printObjectForRange := func(index types.Map, r queryRange) {
		index.IterFrom(r.lower.value, func(k, v types.Value) bool {
			if first && r.lower.value != nil && !r.lower.include && r.lower.value.Equals(k) {
				return false
			}
			if r.upper.value != nil {
				if !r.upper.include && r.upper.value.Equals(k) {
					return true
				}
				if r.upper.value.Less(k) {
					return true
				}
			}
			s := v.(types.Set)
			s.IterAll(func(v types.Value) {
				types.WriteEncodedValue(w, v)
				fmt.Fprintf(w, "\n")
				cnt++
			})
			return false
		})
	}
	for _, r := range ranges {
		printObjectForRange(index, r)
	}
	fmt.Fprintf(w, "Found %d objects\n", cnt)
}

func openIndex(idxPath string) (datas.Database, types.Map, error) {
	db, value, err := spec.GetPath(idxPath)
	if err != nil {
		return nil, types.Map{}, err
	}

	var index types.Map
	s, ok := value.(types.Struct)
	if ok && datas.IsCommitType(s.Type()) {
		index, ok = s.Get("value").(types.Map)
		if !ok {
			return nil, types.Map{}, fmt.Errorf("Value of commit is not a valid index")
		}
	} else {
		index, ok = value.(types.Map)
		if !ok {
			return nil, types.Map{}, fmt.Errorf("%s is not a valid index", outDsArg)
		}
	}

	// Todo: make this type be Map<String | Number>, Set<Value>> once Issue #2326 gets resolved and
	// IsSubtype() returns the correct value.
	typ := types.MakeMapType(
		types.MakeUnionType(types.StringType, types.NumberType),
		types.ValueType)

	if !types.IsSubtype(typ, index.Type()) {
		err := fmt.Errorf("%s does not point to a suitable index type:", idxPath)
		return nil, types.Map{}, err
	}

	return db, index, nil
}
