//go:build ignore

// gen builds the pm2go code graph and knowledge graph.
//
//	go run docs/knowledge-base/graph/gen.go
//
// Outputs (next to this file): code-graph.json, knowledge-graph.json, GRAPH.md.
// Inputs: Go sources under servers/pm2go, pm2go-work/*.md (issue status),
// knowledge-graph.curated.json (hand-written concepts and relations).
package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const modPrefix = "github.com/rpsoftech/golang-servers/servers/pm2go"

type Node struct {
	ID       string            `json:"id"`
	Kind     string            `json:"kind"`
	Name     string            `json:"name"`
	Package  string            `json:"package,omitempty"`
	File     string            `json:"file,omitempty"`
	Line     int               `json:"line,omitempty"`
	Exported bool              `json:"exported,omitempty"`
	Doc      string            `json:"doc,omitempty"`
	Extra    map[string]string `json:"extra,omitempty"`
}

type Edge struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Kind   string `json:"kind"`
	Approx bool   `json:"approx,omitempty"`
}

type Graph struct {
	Generated string         `json:"generated_by"`
	Stats     map[string]int `json:"stats"`
	Nodes     []Node         `json:"nodes"`
	Edges     []Edge         `json:"edges"`
}

func firstSentence(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Join(strings.Fields(s), " ")
	if i := strings.Index(s, ". "); i > 0 {
		s = s[:i+1]
	}
	if len(s) > 220 {
		s = s[:220] + "..."
	}
	return s
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type fnInfo struct {
	id, pkg, name, recv string
	decl                *ast.FuncDecl
	imports             map[string]string // alias -> import path
}

func main() {
	root := "servers/pm2go"
	fset := token.NewFileSet()
	nodes := map[string]*Node{}
	var edges []Edge
	edgeSeen := map[string]bool{}
	addEdge := func(from, to, kind string, approx bool) {
		k := from + "|" + to + "|" + kind
		if from == to || edgeSeen[k] {
			return
		}
		edgeSeen[k] = true
		edges = append(edges, Edge{from, to, kind, approx})
	}
	addNode := func(n Node) { nodes[n.ID] = &n }

	pkgName := map[string]string{} // rel dir -> package name
	var funcs []*fnInfo
	plain := map[string]string{}     // "reldir.Name" -> id
	methods := map[string][]string{} // method name -> ids
	typeIDs := map[string]string{}   // "reldir.Name" -> id
	var structs []struct {
		id, rel string
		st      *ast.StructType
		imports map[string]string
	}
	routeRe := regexp.MustCompile(`^(GET|POST|PUT|DELETE|PATCH) /`)

	var dirs []string
	must(filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "ui" || d.Name() == "deploy" || d.Name() == "testdata" {
				return filepath.SkipDir
			}
			dirs = append(dirs, p)
		}
		return nil
	}))
	sort.Strings(dirs)
	stats := map[string]int{}

	for _, dir := range dirs {
		ents, _ := os.ReadDir(dir)
		var goFiles []string
		for _, e := range ents {
			n := e.Name()
			if strings.HasSuffix(n, ".go") && !strings.HasSuffix(n, "_test.go") {
				goFiles = append(goFiles, filepath.Join(dir, n))
			}
		}
		if len(goFiles) == 0 {
			continue
		}
		rel := strings.TrimPrefix(strings.TrimPrefix(dir, root), "/")
		if rel == "" {
			rel = "."
		}
		pkgID := "pkg:" + rel
		var pkgDoc string
		for _, f := range goFiles {
			af, err := parser.ParseFile(fset, f, nil, parser.ParseComments)
			must(err)
			pkgName[rel] = af.Name.Name
			if af.Doc != nil && pkgDoc == "" {
				pkgDoc = firstSentence(af.Doc.Text())
			}
			imports := map[string]string{}
			for _, im := range af.Imports {
				path, _ := strconv.Unquote(im.Path.Value)
				alias := filepath.Base(path)
				if im.Name != nil {
					alias = im.Name.Name
				}
				imports[alias] = path
				if strings.HasPrefix(path, modPrefix) {
					r := strings.TrimPrefix(strings.TrimPrefix(path, modPrefix), "/")
					if r == "" {
						r = "."
					}
					addEdge(pkgID, "pkg:"+r, "imports", false)
				}
			}
			fileID := "file:" + filepath.ToSlash(f)
			loc := fset.Position(af.End()).Line
			addNode(Node{ID: fileID, Kind: "file", Name: filepath.Base(f), Package: rel, File: filepath.ToSlash(f), Extra: map[string]string{"lines": strconv.Itoa(loc)}})
			addEdge(pkgID, fileID, "contains", false)
			stats["files"]++
			for _, decl := range af.Decls {
				switch d := decl.(type) {
				case *ast.GenDecl:
					for _, sp := range d.Specs {
						ts, ok := sp.(*ast.TypeSpec)
						if !ok {
							continue
						}
						id := "type:" + rel + "." + ts.Name.Name
						kind := "type"
						switch ts.Type.(type) {
						case *ast.StructType:
							kind = "struct"
						case *ast.InterfaceType:
							kind = "interface"
						}
						doc := ""
						if ts.Doc != nil {
							doc = firstSentence(ts.Doc.Text())
						} else if d.Doc != nil && len(d.Specs) == 1 {
							doc = firstSentence(d.Doc.Text())
						}
						pos := fset.Position(ts.Pos())
						addNode(Node{ID: id, Kind: kind, Name: ts.Name.Name, Package: rel, File: filepath.ToSlash(f), Line: pos.Line, Exported: ts.Name.IsExported(), Doc: doc})
						addEdge(fileID, id, "contains", false)
						typeIDs[rel+"."+ts.Name.Name] = id
						stats["types"]++
						if st, ok := ts.Type.(*ast.StructType); ok {
							structs = append(structs, struct {
								id, rel string
								st      *ast.StructType
								imports map[string]string
							}{id, rel, st, imports})
						}
					}
				case *ast.FuncDecl:
					name := d.Name.Name
					recv := ""
					if d.Recv != nil && len(d.Recv.List) > 0 {
						recv = typeName(d.Recv.List[0].Type)
					}
					id := "func:" + rel + "." + name
					kind := "func"
					if recv != "" {
						id = "method:" + rel + "." + recv + "." + name
						kind = "method"
					}
					pos := fset.Position(d.Pos())
					doc := ""
					if d.Doc != nil {
						doc = firstSentence(d.Doc.Text())
					}
					extra := map[string]string{}
					if recv != "" {
						extra["receiver"] = recv
					}
					addNode(Node{ID: id, Kind: kind, Name: name, Package: rel, File: filepath.ToSlash(f), Line: pos.Line, Exported: d.Name.IsExported(), Doc: doc, Extra: extra})
					addEdge(fileID, id, "contains", false)
					if recv != "" {
						methods[name] = append(methods[name], id)
						if tid, ok := typeIDs[rel+"."+recv]; ok {
							addEdge(tid, id, "has_method", false)
						} else {
							pending = append(pending, [2]string{rel + "." + recv, id})
						}
					} else {
						plain[rel+"."+name] = id
					}
					stats["funcs"]++
					funcs = append(funcs, &fnInfo{id: id, pkg: rel, name: name, recv: recv, decl: d, imports: imports})
				}
			}
		}
		addNode(Node{ID: pkgID, Kind: "package", Name: rel, Package: rel, Doc: pkgDoc, Extra: map[string]string{"import_path": modPrefix + "/" + strings.TrimPrefix(rel, "./")}})
		stats["packages"]++
	}
	for _, p := range pending {
		if tid, ok := typeIDs[p[0]]; ok {
			addEdge(tid, p[1], "has_method", false)
		}
	}

	// struct field type references
	fieldType := map[string]map[string]string{} // type id -> field name -> type id
	for _, s := range structs {
		for _, fld := range s.st.Fields.List {
			t := refType(fld.Type, s.rel, s.imports, typeIDs)
			if t == "" {
				continue
			}
			addEdge(s.id, t, "field_type", false)
			if fieldType[s.id] == nil {
				fieldType[s.id] = map[string]string{}
			}
			for _, nm := range fld.Names {
				fieldType[s.id][nm.Name] = t
			}
		}
	}

	// call edges and routes
	for _, f := range funcs {
		if f.decl.Body == nil {
			continue
		}
		env := map[string]string{} // identifier -> type id
		if f.decl.Recv != nil && len(f.decl.Recv.List) > 0 && len(f.decl.Recv.List[0].Names) > 0 {
			if t := refType(f.decl.Recv.List[0].Type, f.pkg, f.imports, typeIDs); t != "" {
				env[f.decl.Recv.List[0].Names[0].Name] = t
			}
		}
		for _, fld := range f.decl.Type.Params.List {
			if t := refType(fld.Type, f.pkg, f.imports, typeIDs); t != "" {
				for _, nm := range fld.Names {
					env[nm.Name] = t
				}
			}
		}
		ast.Inspect(f.decl.Body, func(n ast.Node) bool {
			if as, ok := n.(*ast.AssignStmt); ok && as.Tok == token.DEFINE && len(as.Lhs) == len(as.Rhs) {
				for i, rhs := range as.Rhs {
					id, ok := as.Lhs[i].(*ast.Ident)
					if !ok {
						continue
					}
					e := rhs
					if u, ok := e.(*ast.UnaryExpr); ok && u.Op == token.AND {
						e = u.X
					}
					if cl, ok := e.(*ast.CompositeLit); ok {
						if t := refType(cl.Type, f.pkg, f.imports, typeIDs); t != "" {
							env[id.Name] = t
						}
					}
				}
			}
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			switch fn := call.Fun.(type) {
			case *ast.Ident:
				if id, ok := plain[f.pkg+"."+fn.Name]; ok {
					addEdge(f.id, id, "calls", false)
				}
			case *ast.SelectorExpr:
				if x, ok := fn.X.(*ast.Ident); ok {
					if path, ok := f.imports[x.Name]; ok && strings.HasPrefix(path, modPrefix) {
						r := strings.TrimPrefix(strings.TrimPrefix(path, modPrefix), "/")
						if r == "" {
							r = "."
						}
						if id, ok := plain[r+"."+fn.Sel.Name]; ok {
							addEdge(f.id, id, "calls", false)
						}
						return true
					}
				}
				if t := exprType(fn.X, env, fieldType); t != "" {
					// t is "type:<dir>.<Name>"
					mid := "method:" + strings.TrimPrefix(t, "type:") + "." + fn.Sel.Name
					if _, ok := nodes[mid]; ok {
						addEdge(f.id, mid, "calls", false)
						return true
					}
				}
				if ids := methods[fn.Sel.Name]; len(ids) == 1 {
					addEdge(f.id, ids[0], "calls", true)
				}
				// HTTP routes: mux.HandleFunc("GET /api/x", handler)
				if fn.Sel.Name == "HandleFunc" && len(call.Args) >= 2 {
					if bl, ok := call.Args[0].(*ast.BasicLit); ok && bl.Kind == token.STRING {
						pat, _ := strconv.Unquote(bl.Value)
						if routeRe.MatchString(pat) || strings.HasPrefix(pat, "/") {
							rid := "route:" + pat
							addNode(Node{ID: rid, Kind: "route", Name: pat, Package: f.pkg})
							addEdge(f.id, rid, "registers", false)
							stats["routes"]++
							if hid := handlerOf(call.Args[1], f, env, fieldType, nodes, methods); hid != "" {
								addEdge(rid, hid, "handled_by", false)
							}
						}
					}
				}
			}
			return true
		})
	}

	// emit code graph
	cg := Graph{Generated: "docs/knowledge-base/graph/gen.go", Stats: stats}
	for _, n := range nodes {
		cg.Nodes = append(cg.Nodes, *n)
	}
	sort.Slice(cg.Nodes, func(i, j int) bool { return cg.Nodes[i].ID < cg.Nodes[j].ID })
	sort.Slice(edges, func(i, j int) bool {
		a, b := edges[i], edges[j]
		if a.From != b.From {
			return a.From < b.From
		}
		if a.To != b.To {
			return a.To < b.To
		}
		return a.Kind < b.Kind
	})
	cg.Edges = edges
	stats["nodes"], stats["edges"] = len(cg.Nodes), len(cg.Edges)
	writeJSON("docs/knowledge-base/graph/code-graph.json", cg)

	kg := buildKnowledge(nodes, edges)
	writeJSON("docs/knowledge-base/graph/knowledge-graph.json", kg)
	writeMarkdown(cg, kg)
	fmt.Printf("code graph: %d nodes, %d edges; knowledge graph: %d nodes, %d edges\n", len(cg.Nodes), len(cg.Edges), len(kg.Nodes), len(kg.Edges))
}

var pending [][2]string

// exprType resolves a receiver expression to a type id using known variable
// types and struct field types. It returns "" when unknown.
func exprType(e ast.Expr, env map[string]string, fieldType map[string]map[string]string) string {
	switch x := e.(type) {
	case *ast.Ident:
		return env[x.Name]
	case *ast.StarExpr:
		return exprType(x.X, env, fieldType)
	case *ast.ParenExpr:
		return exprType(x.X, env, fieldType)
	case *ast.SelectorExpr:
		if t := exprType(x.X, env, fieldType); t != "" {
			return fieldType[t][x.Sel.Name]
		}
	}
	return ""
}

// handlerOf finds the handler method behind a route registration argument:
// a method value (a.servers), a wrapper call around one (req(a.servers)), or
// "" for inline closures.
func handlerOf(e ast.Expr, f *fnInfo, env map[string]string, fieldType map[string]map[string]string, nodes map[string]*Node, methods map[string][]string) string {
	switch x := e.(type) {
	case *ast.SelectorExpr:
		if t := exprType(x.X, env, fieldType); t != "" {
			id := "method:" + strings.TrimPrefix(t, "type:") + "." + x.Sel.Name
			if _, ok := nodes[id]; ok {
				return id
			}
		}
		if ids := methods[x.Sel.Name]; len(ids) == 1 {
			return ids[0]
		}
	case *ast.CallExpr:
		for _, a := range x.Args {
			if id := handlerOf(a, f, env, fieldType, nodes, methods); id != "" {
				return id
			}
		}
	}
	return ""
}

func typeName(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.StarExpr:
		return typeName(t.X)
	case *ast.Ident:
		return t.Name
	case *ast.IndexExpr:
		return typeName(t.X)
	}
	return ""
}

func refType(e ast.Expr, rel string, imports map[string]string, typeIDs map[string]string) string {
	switch t := e.(type) {
	case *ast.StarExpr:
		return refType(t.X, rel, imports, typeIDs)
	case *ast.ArrayType:
		return refType(t.Elt, rel, imports, typeIDs)
	case *ast.MapType:
		return refType(t.Value, rel, imports, typeIDs)
	case *ast.Ident:
		return typeIDs[rel+"."+t.Name]
	case *ast.SelectorExpr:
		if x, ok := t.X.(*ast.Ident); ok {
			if path, ok := imports[x.Name]; ok && strings.HasPrefix(path, modPrefix) {
				r := strings.TrimPrefix(strings.TrimPrefix(path, modPrefix), "/")
				if r == "" {
					r = "."
				}
				return typeIDs[r+"."+t.Sel.Name]
			}
		}
	}
	return ""
}

func writeJSON(path string, v any) {
	b, err := json.MarshalIndent(v, "", " ")
	must(err)
	must(os.WriteFile(path, append(b, '\n'), 0o644))
}

// ---------- knowledge graph ----------

type KNode struct {
	ID      string            `json:"id"`
	Kind    string            `json:"kind"`
	Name    string            `json:"name"`
	Summary string            `json:"summary,omitempty"`
	Code    []string          `json:"code,omitempty"`
	Docs    []string          `json:"docs,omitempty"`
	Status  string            `json:"status,omitempty"`
	Extra   map[string]string `json:"extra,omitempty"`
}

type KGraph struct {
	Generated string  `json:"generated_by"`
	Note      string  `json:"note"`
	Nodes     []KNode `json:"nodes"`
	Edges     []Edge  `json:"edges"`
}

func buildKnowledge(code map[string]*Node, _ []Edge) KGraph {
	var cur struct {
		Nodes           []KNode             `json:"nodes"`
		Edges           []Edge              `json:"edges"`
		IssueComponents map[string][]string `json:"issue_components"`
	}
	b, err := os.ReadFile("docs/knowledge-base/graph/knowledge-graph.curated.json")
	must(err)
	must(json.Unmarshal(b, &cur))
	kg := KGraph{
		Generated: "docs/knowledge-base/graph/gen.go (curated part: knowledge-graph.curated.json)",
		Note:      "Node id prefixes: comp: component, ext: external system, con: concept, flow: flow, doc: document, issue: GitHub issue, code: code reference (ids of code-graph.json). Edge kinds are free text verbs.",
	}
	ids := map[string]bool{}
	for _, n := range cur.Nodes {
		kg.Nodes = append(kg.Nodes, n)
		ids[n.ID] = true
	}
	kg.Edges = append(kg.Edges, cur.Edges...)

	// code reference nodes and implemented_in edges
	refSeen := map[string]bool{}
	for _, n := range cur.Nodes {
		for _, c := range n.Code {
			rid := "code:" + c
			if !refSeen[rid] {
				refSeen[rid] = true
				kn := KNode{ID: rid, Kind: "code", Name: c}
				if cn, ok := code[c]; ok {
					kn.Summary = cn.Doc
					kn.Extra = map[string]string{"file": cn.File, "line": strconv.Itoa(cn.Line)}
				} else if !strings.HasPrefix(c, "dir:") {
					fmt.Fprintf(os.Stderr, "warning: curated code ref %q not found in code graph\n", c)
				}
				kg.Nodes = append(kg.Nodes, kn)
			}
			kg.Edges = append(kg.Edges, Edge{From: n.ID, To: rid, Kind: "implemented_in"})
		}
	}

	// documents
	docDir := "docs/knowledge-base"
	docs, _ := filepath.Glob(docDir + "/pm2go*.md")
	more, _ := filepath.Glob(docDir + "/pm2-go-feature-analysis.md")
	docs = append(docs, more...)
	more, _ = filepath.Glob(docDir + "/graph/GRAPH.md")
	docs = append(docs, more...)
	for _, d := range docs {
		rel := strings.TrimPrefix(d, docDir+"/")
		id := "doc:" + rel
		title := rel
		if f, err := os.ReadFile(d); err == nil {
			if m := regexp.MustCompile(`(?m)^# (.+)$`).FindStringSubmatch(string(f)); m != nil {
				title = m[1]
			}
		}
		kg.Nodes = append(kg.Nodes, KNode{ID: id, Kind: "doc", Name: title, Extra: map[string]string{"path": d}})
		ids[id] = true
	}
	for _, n := range cur.Nodes {
		for _, d := range n.Docs {
			if ids["doc:"+d] {
				kg.Edges = append(kg.Edges, Edge{From: n.ID, To: "doc:" + d, Kind: "documented_in"})
			}
		}
	}

	// issues from the per-issue work files
	files, _ := filepath.Glob(docDir + "/pm2go-work/[0-9][0-9][0-9]-*.md")
	sort.Strings(files)
	hdr := func(s, key string) string {
		m := regexp.MustCompile(`(?m)^` + key + `: *(.*)$`).FindStringSubmatch(s)
		if m == nil {
			return ""
		}
		return strings.TrimSpace(m[1])
	}
	for _, f := range files {
		if strings.Contains(f, "/000-") {
			continue
		}
		raw, _ := os.ReadFile(f)
		s := string(raw)
		tm := regexp.MustCompile(`(?m)^# #(\d+): (.+)$`).FindStringSubmatch(s)
		if tm == nil {
			continue
		}
		num, title := tm[1], tm[2]
		id := "issue:" + num
		size := hdr(s, "Size")
		kg.Nodes = append(kg.Nodes, KNode{ID: id, Kind: "issue", Name: "#" + num + " " + title, Status: hdr(s, "Status"),
			Extra: map[string]string{"issue": "https://github.com/rpsoftech/GolangServers/issues/" + num, "pr": hdr(s, "PR"), "branch": hdr(s, "Branch"), "size_lane_needs": size, "work_file": f}})
		for _, need := range regexp.MustCompile(`#(\d+)`).FindAllStringSubmatch(strings.SplitN(size, "Needs first:", 2)[len(strings.SplitN(size, "Needs first:", 2))-1], -1) {
			kg.Edges = append(kg.Edges, Edge{From: id, To: "issue:" + need[1], Kind: "depends_on"})
		}
		for _, c := range cur.IssueComponents[num] {
			if ids[c] {
				kg.Edges = append(kg.Edges, Edge{From: id, To: c, Kind: "changes"})
			}
		}
		kg.Edges = append(kg.Edges, Edge{From: id, To: "doc:pm2go-work/" + filepath.Base(f), Kind: "tracked_in"})
	}
	// work files as docs too
	for _, f := range files {
		kg.Nodes = append(kg.Nodes, KNode{ID: "doc:pm2go-work/" + filepath.Base(f), Kind: "doc", Name: filepath.Base(f), Extra: map[string]string{"path": f}})
	}
	// keep only edges with existing endpoints
	have := map[string]bool{}
	for _, n := range kg.Nodes {
		have[n.ID] = true
	}
	var okEdges []Edge
	for _, e := range kg.Edges {
		if have[e.From] && have[e.To] {
			okEdges = append(okEdges, e)
		} else {
			fmt.Fprintf(os.Stderr, "dropped edge %s -%s-> %s\n", e.From, e.Kind, e.To)
		}
	}
	kg.Edges = okEdges
	return kg
}

// ---------- markdown ----------

func mid(id string) string { return regexp.MustCompile(`[^A-Za-z0-9]`).ReplaceAllString(id, "_") }

func writeMarkdown(cg Graph, kg KGraph) {
	var sb strings.Builder
	w := func(f string, a ...any) { fmt.Fprintf(&sb, f, a...) }
	w("# pm2go graphs\n\n")
	w("Generated by `go run docs/knowledge-base/graph/gen.go` (do not edit by hand; edit `knowledge-graph.curated.json` for concepts and relations). Machine-readable versions: [`code-graph.json`](code-graph.json) and [`knowledge-graph.json`](knowledge-graph.json).\n\n")
	w("Use these before reading code: find the component, then open only the files it points to.\n\n")
	w("- Code graph: %d nodes, %d edges (%d packages, %d files, %d types, %d funcs and methods, %d HTTP routes).\n", cg.Stats["nodes"], cg.Stats["edges"], cg.Stats["packages"], cg.Stats["files"], cg.Stats["types"], cg.Stats["funcs"], cg.Stats["routes"])
	w("- Knowledge graph: %d nodes, %d edges.\n\n", len(kg.Nodes), len(kg.Edges))

	// architecture
	w("## Architecture (knowledge graph, components and external systems)\n\n```mermaid\nflowchart LR\n")
	arch := map[string]bool{}
	for _, n := range kg.Nodes {
		if n.Kind == "component" || n.Kind == "external" {
			arch[n.ID] = true
			shape := "[\"%s\"]"
			if n.Kind == "external" {
				shape = "([\"%s\"])"
			}
			w("  %s"+shape+"\n", mid(n.ID), strings.ReplaceAll(n.Name, "\"", "'"))
		}
	}
	for _, e := range kg.Edges {
		if arch[e.From] && arch[e.To] && e.Kind != "documents" {
			w("  %s -->|%s| %s\n", mid(e.From), e.Kind, mid(e.To))
		}
	}
	w("```\n\n")

	// flows
	w("## Key flows\n\n")
	for _, n := range kg.Nodes {
		if n.Kind == "flow" {
			w("- **%s**: %s\n", n.Name, n.Summary)
		}
	}

	// package imports
	w("\n## Package imports (code graph)\n\n```mermaid\nflowchart LR\n")
	for _, n := range cg.Nodes {
		if n.Kind == "package" {
			w("  %s[\"%s\"]\n", mid(n.ID), n.Name)
		}
	}
	for _, e := range cg.Edges {
		if e.Kind == "imports" {
			w("  %s --> %s\n", mid(e.From), mid(e.To))
		}
	}
	w("```\n\n")

	// issues
	w("## Issue dependencies and status (knowledge graph)\n\n```mermaid\nflowchart LR\n")
	st := map[string]string{}
	for _, n := range kg.Nodes {
		if n.Kind == "issue" {
			short := strings.TrimPrefix(n.ID, "issue:")
			label := n.Name
			if len(label) > 44 {
				label = label[:44] + "..."
			}
			w("  %s[\"%s<br/>%s\"]\n", mid(n.ID), strings.ReplaceAll(label, "\"", "'"), n.Status)
			st[short] = n.Status
		}
	}
	for _, e := range kg.Edges {
		if e.Kind == "depends_on" {
			w("  %s --> %s\n", mid(e.To), mid(e.From))
		}
	}
	for _, n := range kg.Nodes {
		if n.Kind == "issue" {
			switch n.Status {
			case "done":
				w("  style %s fill:#1f6f3a,color:#fff\n", mid(n.ID))
			case "in review", "in progress":
				w("  style %s fill:#8a6d1a,color:#fff\n", mid(n.ID))
			}
		}
	}
	w("```\n\nArrow means \"must land before\". Green done, amber in progress or review, default open.\n\n")

	// routes
	w("## HTTP routes and handlers (code graph)\n\n| Route | Handler |\n|-------|---------|\n")
	handler := map[string]string{}
	for _, e := range cg.Edges {
		if e.Kind == "handled_by" {
			handler[e.From] = e.To
		}
	}
	var routes []string
	for _, n := range cg.Nodes {
		if n.Kind == "route" {
			routes = append(routes, n.ID)
		}
	}
	sort.Strings(routes)
	node := map[string]Node{}
	for _, n := range cg.Nodes {
		node[n.ID] = n
	}
	for _, r := range routes {
		h := "(inline)"
		if hid, ok := handler[r]; ok {
			hn := node[hid]
			h = fmt.Sprintf("`%s` (%s:%d)", hn.Name, hn.File, hn.Line)
		}
		w("| `%s` | %s |\n", strings.TrimPrefix(r, "route:"), h)
	}

	// package table
	w("\n## Packages\n\n| Package | Files | Types | Funcs | Summary |\n|---------|-------|-------|-------|---------|\n")
	count := map[string]map[string]int{}
	for _, n := range cg.Nodes {
		if count[n.Package] == nil {
			count[n.Package] = map[string]int{}
		}
		count[n.Package][n.Kind]++
	}
	for _, n := range cg.Nodes {
		if n.Kind == "package" {
			c := count[n.Package]
			w("| `%s` | %d | %d | %d | %s |\n", n.Name, c["file"], c["struct"]+c["interface"]+c["type"], c["func"]+c["method"], n.Doc)
		}
	}

	w(`
## How agents should use this

1. Read this file, then open the **component** in knowledge-graph.json: its "code" refs name the packages, types and files to read, and "docs" names the document that explains the behaviour.
2. To find who calls a function: in code-graph.json look for edges with kind "calls" and "to" equal to its id. Edges with "approx": true were matched by a unique method name only; verify them.
3. To find what an issue touches: knowledge-graph.json edges "changes" from issue:N, plus the issue's own file under docs/knowledge-base/pm2go-work/.
4. Node ids: pkg:<dir>, file:<path>, type:<dir>.<Name>, func:<dir>.<Name>, method:<dir>.<Type>.<Name>, route:<METHOD /path>.
5. After changing structure (new package, type, route, issue, component): rerun the generator and commit the three generated files.

### Query recipes (jq)

~~~bash
# all callers of a function
jq -r '.edges[] | select(.kind=="calls" and .to=="method:daemon.Manager.Reload") | .from' docs/knowledge-base/graph/code-graph.json
# everything an issue changes
jq -r '.edges[] | select(.from=="issue:103" and .kind=="changes") | .to' docs/knowledge-base/graph/knowledge-graph.json
# open issues that depend on nothing open
jq -r '.nodes[] | select(.kind=="issue" and .status=="open") | .id' docs/knowledge-base/graph/knowledge-graph.json
~~~
`)
	must(os.WriteFile("docs/knowledge-base/graph/GRAPH.md", []byte(sb.String()), 0o644))
}
