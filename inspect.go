package proper

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"strings"

	"github.com/andrewarchi/proper/proptypes"
)

func InspectDir(fset *token.FileSet, dir string) (map[string][]proptypes.PropType, error) {
	pkgs, err := parser.ParseDir(fset, dir, func(info os.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, 0)
	if err != nil {
		return nil, err
	}
	types := make(map[string][]proptypes.PropType)
	for _, pkg := range pkgs {
		for name, f := range pkg.Files {
			types[name] = inspectFile(fset, f)
		}
	}
	return types, nil
}

func inspectFile(fset *token.FileSet, f *ast.File) []proptypes.PropType {
	var propTypes []proptypes.PropType
	for _, decl := range f.Decls {
		if typeDecl, ok := decl.(*ast.GenDecl); ok && typeDecl.Tok == token.TYPE {
			fmt.Println(fset.Position(typeDecl.Pos()))
			for _, s := range typeDecl.Specs {
				if spec, ok := s.(*ast.TypeSpec); ok {
					fmt.Println("#", spec.Name)
					propType := inspectExpr(spec.Type)
					if propType != nil {
						fmt.Println("PROPTYPE: ", propType.Format(0))
						propTypes = append(propTypes, propType)
					} else {
						fmt.Println("PROPTYPE: nil")
					}
				}
			}
			fmt.Println()
		}
	}
	return propTypes
}

func inspectExpr(expr ast.Expr) proptypes.PropType {
	fmt.Println(reflect.TypeOf(expr).Elem().Name())
	switch typ := expr.(type) {
	case *ast.Ident:
	case *ast.ParenExpr:
	case *ast.SelectorExpr:
	case *ast.StarExpr:
		return inspectExpr(typ.X)
	case *ast.ArrayType:
		return proptypes.ArrayOf(inspectExpr(typ.Elt))
	case *ast.MapType:
		return inspectMap(typ)
	case *ast.StructType:
		return inspectStruct(typ)
	case *ast.ChanType, *ast.FuncType, *ast.InterfaceType:
		return nil // cannot be marshalled to json
	default:
		fmt.Println("Other")
	}
	return nil
}

func inspectMap(m *ast.MapType) proptypes.PropType {
	return proptypes.Shape(nil) // only if key is string
}

func inspectStruct(typ *ast.StructType) proptypes.PropType {
	var shape proptypes.ShapeMap
	if typ.Fields != nil {
		for _, field := range typ.Fields.List {
			propType := inspectExpr(field.Type)
			for _, name := range field.Names {
				fmt.Print(field)
				if !name.IsExported() { // undetectable in json marshall
					continue
				}
				fieldName := name.Name
				omitEmpty := false
				if tag, ok := lookupTag(field, "json"); ok {
					tagName, options := parseJSONTag(tag)
					if tagName == "-" {
						continue
					}
					if tagName != "" {
						fieldName = tagName
					}
					omitEmpty = options.Contains("omitempty")
				}
				shape = append(shape, proptypes.ShapeEntry{Name: fieldName, Type: propType})
				fmt.Println(fieldName, omitEmpty)
			}
		}
	}
	return proptypes.Shape(shape)
}
