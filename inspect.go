package proper

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
)

func InspectDir(fset *token.FileSet, dir string) {
	pkgs, err := parser.ParseDir(fset, dir, nil, parser.AllErrors) // 0
	if err != nil {
		panic(err)
	}
	for _, pkg := range pkgs {
		for name, f := range pkg.Files {
			if !strings.HasSuffix(name, "_test.go") {
				inspectFile(fset, f)
			}
		}
	}
}

func inspectFile(fset *token.FileSet, f *ast.File) {
	for _, decl := range f.Decls {
		if typeDecl, ok := decl.(*ast.GenDecl); ok && typeDecl.Tok == token.TYPE {
			fmt.Println(fset.Position(typeDecl.Pos()))
			for _, s := range typeDecl.Specs {
				if spec, ok := s.(*ast.TypeSpec); ok {
					fmt.Println("#", spec.Name)
					var jsTyp string
					switch typ := spec.Type.(type) {
					case *ast.Ident:
						fmt.Println("Ident")
					case *ast.ParenExpr:
						fmt.Println("ParenExpr")
					case *ast.SelectorExpr:
						fmt.Println("SelectorExpr")
					case *ast.StarExpr:
						fmt.Println("StarExpr")
					case *ast.ArrayType:
						jsTyp = "PropTypes.array"
					case *ast.ChanType:
						// unsupported
					case *ast.FuncType:
						jsTyp = "PropTypes.func"
					case *ast.InterfaceType:
						jsTyp = "PropTypes.object"
					case *ast.MapType:
						fmt.Println("MapType")
					case *ast.StructType:
						inspectStruct(typ)
					default:
						fmt.Println("Other:", reflect.TypeOf(spec.Type).Elem().Name())
					}
					if jsTyp != "" {
						fmt.Println(jsTyp)
					}
				}
			}
			fmt.Println()
		}
	}
}

func inspectStruct(typ *ast.StructType) {
	if typ.Fields != nil {
		for _, field := range typ.Fields.List {
			indirect := false
			switch typ := field.Type.(type) {
			case *ast.Ident:
				fmt.Println("Ident", typ.Name, typ.Obj)
			case *ast.SelectorExpr:
				fmt.Println("SelectorExpr", typ.X, typ.Sel)
			case *ast.StarExpr:
				indirect = true
				fmt.Println("StarExpr", typ.X)
			default: // not a conclusive list yet
				fmt.Println("Unknown:", reflect.TypeOf(field.Type).Elem().Name())
			}
			for _, name := range field.Names {
				fmt.Print(field)
				if !name.IsExported() { // undetectable in json marshall
					continue
				}
				fieldName := name.Name
				omitEmpty := false
				if tag, ok := lookupTag(field); ok {
					tagName, options := parseTag(tag)
					if tagName == "-" {
						continue
					}
					if tagName != "" {
						fieldName = tagName
					}
					omitEmpty = options.Contains("omitempty")
				}
				fmt.Println(fieldName, omitEmpty, indirect)
			}
		}
	}
}
