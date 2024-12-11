package main

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// dependencyInfo guarda las declaraciones top-level del paquete
type dependencyInfo struct {
	funcs  map[string]*ast.FuncDecl
	types  map[string]*ast.GenDecl
	vars   map[string]*ast.GenDecl
	consts map[string]*ast.GenDecl
}

// gatherTopLevelDecls recorre las declaraciones top-level de un archivo y las agrega a dependencyInfo
func gatherTopLevelDecls(file *ast.File, info *dependencyInfo) {
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			info.funcs[d.Name.Name] = d
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					info.types[s.Name.Name] = d
				case *ast.ValueSpec:
					for _, name := range s.Names {
						if d.Tok.String() == "var" {
							info.vars[name.Name] = d
						}
						if d.Tok.String() == "const" {
							info.consts[name.Name] = d
						}
					}
				}
			}
		}
	}
}

// findFunc busca la función solicitada en el mapa de funcs
func findFunc(funcName string, info *dependencyInfo) *ast.FuncDecl {
	return info.funcs[funcName]
}

var builtins = map[string]bool{
	"len": true, "cap": true, "append": true, "make": true,
	"new": true, "delete": true, "complex": true, "real": true, "imag": true,
	"panic": true, "recover": true, "close": true, "print": true, "println": true,
}

func isBuiltin(name string) bool {
	return builtins[name]
}

// extractUsedIdents extrae todos los *ast.Ident usados en el cuerpo de la función
func extractUsedIdents(f *ast.FuncDecl) []*ast.Ident {
	var idents []*ast.Ident
	if f.Body == nil {
		return idents
	}
	ast.Inspect(f.Body, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok {
			idents = append(idents, id)
		}
		return true
	})
	return idents
}

// getFuncParamsAndLocals extrae nombres de parámetros y variables locales de la función
func getFuncParamsAndLocals(f *ast.FuncDecl) map[string]bool {
	localSymbols := make(map[string]bool)
	if f.Type.Params != nil {
		for _, p := range f.Type.Params.List {
			for _, n := range p.Names {
				localSymbols[n.Name] = true
			}
		}
	}
	if f.Type.Results != nil {
		for _, r := range f.Type.Results.List {
			for _, n := range r.Names {
				localSymbols[n.Name] = true
			}
		}
	}
	// Variables locales
	if f.Body != nil {
		ast.Inspect(f.Body, func(n ast.Node) bool {
			if decl, ok := n.(*ast.AssignStmt); ok {
				if decl.Tok == token.DEFINE {
					for _, lhs := range decl.Lhs {
						if id, ok := lhs.(*ast.Ident); ok {
							localSymbols[id.Name] = true
						}
					}
				}
			}
			if decl, ok := n.(*ast.DeclStmt); ok {
				if genD, ok := decl.Decl.(*ast.GenDecl); ok {
					for _, spec := range genD.Specs {
						if vSpec, ok := spec.(*ast.ValueSpec); ok {
							for _, name := range vSpec.Names {
								localSymbols[name.Name] = true
							}
						}
					}
				}
			}
			return true
		})
	}
	return localSymbols
}

// dependenciesOfDecl given a top-level declaration (which could be a func, type, var, const),
// find what other top-level names it depends on.
func dependenciesOfDecl(d ast.Decl, info *dependencyInfo) []string {
	var f *ast.FuncDecl
	switch decl := d.(type) {
	case *ast.FuncDecl:
		f = decl
	default:
		// Only functions have bodies from which we extract usage.
		// Types, vars, consts usually don't reference functions/types in a "body" context.
		// However, types can reference other types. In a simple scenario, let's assume we only need to handle func bodies.
		return nil
	}

	localSymbols := getFuncParamsAndLocals(f)
	usedIdents := extractUsedIdents(f)
	var queue []string
	for _, id := range usedIdents {
		if id.Name == "" || isBuiltin(id.Name) || localSymbols[id.Name] {
			continue
		}
		// If there's no object, it's likely a top-level reference
		if id.Obj == nil {
			queue = append(queue, id.Name)
		}
	}
	return queue
}

// resolveAllDependencies tries to find all dependencies (funcs/types/vars/consts)
// needed by a set of initial declarations. It iteratively adds new dependencies found
// by scanning the newly added declarations, until no new dependencies appear.
func resolveAllDependencies(initialDecl ast.Decl, info *dependencyInfo) []ast.Decl {
	foundDecls := make([]ast.Decl, 0)
	declMap := make(map[ast.Decl]bool)

	// Start with the initial function
	workQueue := []ast.Decl{initialDecl}
	declMap[initialDecl] = true

	for len(workQueue) > 0 {
		d := workQueue[0]
		workQueue = workQueue[1:]

		newDeps := dependenciesOfDecl(d, info)
		for _, name := range newDeps {
			// Check if we have this dependency top-level
			// It could be a func, type, var, const
			if fd, ok := info.funcs[name]; ok {
				if !declMap[fd] {
					declMap[fd] = true
					workQueue = append(workQueue, fd)
					foundDecls = append(foundDecls, fd)
				}
			}
			if td, ok := info.types[name]; ok {
				if !declMap[td] {
					declMap[td] = true
					// types are GenDecl, they don't have a "body" to explore,
					// but we still add them to the result set
					foundDecls = append(foundDecls, td)
					// Types might reference other types in their fields (in a more complex scenario),
					// but resolving that would require more advanced analysis.
				}
			}
			if vd, ok := info.vars[name]; ok {
				if !declMap[vd] {
					declMap[vd] = true
					foundDecls = append(foundDecls, vd)
				}
			}
			if cd, ok := info.consts[name]; ok {
				if !declMap[cd] {
					declMap[cd] = true
					foundDecls = append(foundDecls, cd)
				}
			}
		}
	}

	return foundDecls
}

func main() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Directory to analyze [default: .]: ")
	dir, _ := reader.ReadString('\n')
	dir = strings.TrimSpace(dir)
	if dir == "" {
		dir = "."
	}

	fmt.Print("Name of the function to extract: ")
	funcName, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading function name:", err)
		return
	}
	funcName = strings.TrimSpace(funcName)

	fmt.Print("Output file [default: output.go]: ")
	outputFile, _ := reader.ReadString('\n')
	outputFile = strings.TrimSpace(outputFile)
	if outputFile == "" {
		outputFile = "output.go"
	}

	fset := token.NewFileSet()
	pkgs := make(map[string]*ast.Package)

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		fmt.Println("Error reading directory:", err)
		return
	}

	var goFiles []string
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".go") && f.Name() != filepath.Base(os.Args[0]) {
			goFiles = append(goFiles, filepath.Join(dir, f.Name()))
		}
	}

	if len(goFiles) == 0 {
		fmt.Println("No Go files found in the directory.")
		return
	}

	// Parse all files
	for _, gf := range goFiles {
		astFile, err := parser.ParseFile(fset, gf, nil, parser.ParseComments)
		if err != nil {
			fmt.Println("Error parsing file:", gf, err)
			return
		}
		pkgName := astFile.Name.Name
		pkg, ok := pkgs[pkgName]
		if !ok {
			pkg = &ast.Package{
				Name:  pkgName,
				Files: make(map[string]*ast.File),
			}
			pkgs[pkgName] = pkg
		}
		pkg.Files[gf] = astFile
	}

	// Assume a single package
	if len(pkgs) > 1 {
		fmt.Println("Warning: multiple packages found, using the first one.")
	}
	var thePkg *ast.Package
	for _, p := range pkgs {
		thePkg = p
		break
	}

	info := &dependencyInfo{
		funcs:  make(map[string]*ast.FuncDecl),
		types:  make(map[string]*ast.GenDecl),
		vars:   make(map[string]*ast.GenDecl),
		consts: make(map[string]*ast.GenDecl),
	}
	for _, f := range thePkg.Files {
		gatherTopLevelDecls(f, info)
	}

	targetFunc := findFunc(funcName, info)
	if targetFunc == nil {
		fmt.Printf("Function '%s' not found.\n", funcName)
		return
	}

	// Resolve all dependencies by recursively (iteratively) exploring them
	allDeps := resolveAllDependencies(targetFunc, info)

	// Write to output file
	outF, err := os.Create(outputFile)
	if err != nil {
		fmt.Println("Error creating output file:", err)
		return
	}
	defer outF.Close()

	printerConfig := &printer.Config{Mode: printer.TabIndent, Tabwidth: 4}

	// Write package name
	fmt.Fprintf(outF, "package %s\n\n", thePkg.Name)

	// Write the main function
	fmt.Fprintln(outF, "// --- Requested Function ---")
	if err := printerConfig.Fprint(outF, fset, targetFunc); err != nil {
		fmt.Println("Error printing target function:", err)
		return
	}

	// Write all dependencies found
	if len(allDeps) > 0 {
		fmt.Fprintln(outF, "\n// --- Top-level Dependencies Found ---")
		for _, d := range allDeps {
			if err := printerConfig.Fprint(outF, fset, d); err != nil {
				fmt.Println("Error printing dependency:", err)
			}
			fmt.Fprintln(outF)
		}
	} else {
		fmt.Fprintln(outF, "\n// No additional top-level dependencies found.")
	}

	fmt.Printf("Extraction completed. Code written to '%s'.\n", outputFile)
}
