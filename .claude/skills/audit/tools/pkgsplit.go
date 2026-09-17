// Command pkgsplit reports what a proposed package split would cost, and whether the boundary
// it implies is already being broken.
//
// Given a package and a set of its files to peel off, it type-checks the package and reports
// every unexported package-level identifier declared on one side of the proposed line and used
// on the other. Two numbers come back, and they answer different questions:
//
//   - **Forward edges** — declared in the moved set, used by the rest. This is the price of the
//     split: each one has to be exported, and each export is a rename across every use site.
//   - **Back edges** — declared in the rest, used by the moved set. This is the finding. A file
//     meant to be shared that reaches into one particular screen is a boundary that has already
//     stopped holding; the split only makes the compiler say so. Running this on a partition
//     nobody intends to execute, purely to read the back edges, is a legitimate use.
//
// It is type-aware, so a local variable that happens to share a name with a package-level
// declaration is not counted, and a method is attributed to the file its receiver is declared in.
//
// It lives in .claude/skills/audit/tools rather than in tools/, and builds into .scratch/,
// because it needs golang.org/x/tools/go/packages and the game's own module does not take a
// dependency for the benefit of an audit. The go tool ignores a directory whose name begins with
// a dot, so `go vet ./...` never sees this file.
//
// Usage:
//
//	pkgsplit -pkg ./internal/screens -move "ground.go,travel.go,clock.go"
//	pkgsplit -pkg ./internal/screens -move "combat.go,combat_deck.go" -tags scenario
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

func main() {
	pkgPath := flag.String("pkg", "", "package to analyse, as an import path or ./relative/dir")
	move := flag.String("move", "", "comma-separated base filenames to peel off into a new package")
	tags := flag.String("tags", "", "build tags, so a file selected by one can be analysed")
	positions := flag.Bool("positions", false, "print file:line:col<TAB>name for each forward edge, for gopls rename")
	flag.Parse()

	if *pkgPath == "" || *move == "" {
		fmt.Fprintln(os.Stderr, "usage: pkgsplit -pkg ./internal/screens -move a.go,b.go [-tags scenario]")
		os.Exit(2)
	}

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedSyntax | packages.NeedTypes |
			packages.NeedTypesInfo | packages.NeedCompiledGoFiles,
	}
	if *tags != "" {
		cfg.BuildFlags = []string{"-tags", *tags}
	}

	pkgs, err := packages.Load(cfg, *pkgPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load:", err)
		os.Exit(1)
	}
	if len(pkgs) == 0 {
		fmt.Fprintln(os.Stderr, "load: no packages matched", *pkgPath)
		os.Exit(1)
	}
	pkg := pkgs[0]
	for _, e := range pkg.Errors {
		fmt.Fprintln(os.Stderr, "err:", e)
	}

	moving := map[string]bool{}
	for _, f := range strings.Split(*move, ",") {
		if f = strings.TrimSpace(f); f != "" {
			moving[f] = true
		}
	}
	// A filename that matches nothing is a typo, and silently analysing the wrong partition is
	// worse than refusing: the whole output would look like a clean split.
	present := map[string]bool{}
	for _, f := range pkg.CompiledGoFiles {
		present[filepath.Base(f)] = true
	}
	var unknown []string
	for f := range moving {
		if !present[f] {
			unknown = append(unknown, f)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		fmt.Fprintf(os.Stderr, "not in %s: %s\n", pkg.PkgPath, strings.Join(unknown, " "))
		os.Exit(1)
	}

	scope := pkg.Types.Scope()
	fileOf := func(o types.Object) string {
		return filepath.Base(pkg.Fset.Position(o.Pos()).Filename)
	}

	// forward: declared among the moved files, used by the rest. back: the reverse.
	forward := map[string]map[string]int{}
	back := map[string]map[string]int{}
	forwardNames := map[string]bool{}

	// Every object this package declares, so -positions can find a forward edge's declaration.
	var decls []types.Object
	for _, o := range pkg.TypesInfo.Defs {
		if o != nil && o.Pkg() == pkg.Types {
			decls = append(decls, o)
		}
	}

	for _, f := range pkg.Syntax {
		user := filepath.Base(pkg.Fset.Position(f.Pos()).Filename)
		inMove := moving[user]

		ast.Inspect(f, func(n ast.Node) bool {
			id, ok := n.(*ast.Ident)
			if !ok {
				return true
			}
			o := pkg.TypesInfo.Uses[id]
			// Only this package's own unexported declarations can force an export.
			if o == nil || o.Pkg() != pkg.Types || o.Exported() {
				return true
			}
			// Package-level declarations, plus methods — which live in a type's scope rather
			// than the package's but still cross a package line.
			fn, isFunc := o.(*types.Func)
			isMethod := isFunc && fn.Type().(*types.Signature).Recv() != nil
			if o.Parent() != scope && !isMethod {
				return true
			}
			if moving[fileOf(o)] == inMove {
				return true
			}

			name := o.Name()
			if isMethod {
				name += " (method)"
			}
			side := forward
			if inMove {
				side = back
			} else {
				forwardNames[o.Name()] = true
			}
			if side[name] == nil {
				side[name] = map[string]int{}
			}
			side[name][user]++
			return true
		})
	}

	if *positions {
		// Every forward edge has to be exported, and gopls rename needs the declaration's
		// position. Emitted sorted and deduplicated so the caller can drive it straight.
		var lines []string
		seen := map[string]bool{}
		for _, o := range decls {
			if !forwardNames[o.Name()] {
				continue
			}
			pos := pkg.Fset.Position(o.Pos())
			// A method is only ours to export if its receiver type is itself moving. Several
			// types share method names (tick, done, draw), so the receiver is part of the
			// identity: renaming the wrong one breaks an interface nobody was touching.
			label := o.Name()
			if fn, ok := o.(*types.Func); ok {
				sig := fn.Type().(*types.Signature)
				if sig.Recv() == nil {
					if !moving[filepath.Base(pos.Filename)] {
						continue
					}
				} else {
					recv := sig.Recv().Type()
					if ptr, isPtr := recv.(*types.Pointer); isPtr {
						recv = ptr.Elem()
					}
					named, isNamed := recv.(*types.Named)
					if !isNamed {
						continue
					}
					if !moving[filepath.Base(pkg.Fset.Position(named.Obj().Pos()).Filename)] {
						continue
					}
					label = named.Obj().Name() + "." + o.Name()
				}
			} else if !moving[filepath.Base(pos.Filename)] {
				continue
			}
			if seen[label] {
				continue
			}
			seen[label] = true
			lines = append(lines, fmt.Sprintf("%s:%d:%d\t%s\t%s", pos.Filename, pos.Line, pos.Column, o.Name(), label))
		}
		sort.Strings(lines)
		for _, l := range lines {
			fmt.Println(l)
		}
		return
	}

	report("declared in MOVED files, used by the REST (must be exported)", forward)
	report("declared in the REST, used by MOVED files (the boundary is already broken)", back)
}

// report prints one side of the cut, sorted by how heavily each symbol is used — the heaviest
// are both the most expensive to rename and the most likely to be genuinely shared.
func report(title string, m map[string]map[string]int) {
	type row struct {
		name  string
		uses  int
		files []string
	}
	rows := make([]row, 0, len(m))
	total := 0
	for name, byFile := range m {
		r := row{name: name}
		for f, c := range byFile {
			r.files = append(r.files, f)
			r.uses += c
		}
		sort.Strings(r.files)
		total += r.uses
		rows = append(rows, r)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].uses != rows[j].uses {
			return rows[i].uses > rows[j].uses
		}
		return rows[i].name < rows[j].name
	})

	fmt.Printf("\n== %s: %d symbols, %d reference sites ==\n", title, len(rows), total)
	for _, r := range rows {
		files := r.files
		if len(files) > 4 {
			files = append(append([]string{}, files[:4]...), fmt.Sprintf("+%d more", len(r.files)-4))
		}
		fmt.Printf("  %-34s %4d uses  %s\n", r.name, r.uses, strings.Join(files, " "))
	}
}
