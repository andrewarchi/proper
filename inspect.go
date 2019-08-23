package proper

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/andrewarchi/proper/proptypes"
)

type testPropType struct{ val interface{} }

func (t testPropType) Format(indent int) string {
	if t.val == nil {
		return "<<nil>>"
	}
	if id, ok := t.val.(*ast.Ident); ok {
		o := id.Obj
		if o == nil {
			return fmt.Sprintf("<<%[1]T %q: nil Obj>>", t.val, id.Name)
		}
		return fmt.Sprintf("<<%[1]T %q: %v, %q, %v, %v, %v>>", t.val, id.Name, o.Kind, o.Name, o.Decl, o.Data, o.Type)
	}
	return fmt.Sprintf("<<%[1]T %[1]v>>", t.val)
}

var _ proptypes.PropType = testPropType{nil}

func InspectDirRecursive(fset *token.FileSet, root string) (map[string][]proptypes.PropType, error) {
	types := make(map[string][]proptypes.PropType)
	err := filepath.Walk(root, func(path string, f os.FileInfo, err error) error {
		if f.IsDir() {
			return inspectDir(fset, path, types)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return types, nil
}

func InspectDir(fset *token.FileSet, dir string) (map[string][]proptypes.PropType, error) {
	types := make(map[string][]proptypes.PropType)
	if err := inspectDir(fset, dir, types); err != nil {
		return nil, err
	}
	return types, nil
}

func inspectDir(fset *token.FileSet, dir string, types map[string][]proptypes.PropType) error {
	pkgs, err := parser.ParseDir(fset, dir, func(info os.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, 0)
	if err != nil {
		return err
	}
	for _, pkg := range pkgs {
		for name, f := range pkg.Files {
			types[name] = inspectFile(fset, f)
		}
	}
	return nil
}

func inspectFile(fset *token.FileSet, f *ast.File) []proptypes.PropType {
	var propTypes []proptypes.PropType
	for _, decl := range f.Decls {
		if typeDecl, ok := decl.(*ast.GenDecl); ok && typeDecl.Tok == token.TYPE {
			fmt.Println(fset.Position(typeDecl.Pos()))
			for _, s := range typeDecl.Specs {
				if spec, ok := s.(*ast.TypeSpec); ok {
					fmt.Printf("const %s = ", spec.Name)
					if propType, ok := inspectExpr(spec.Type); ok {
						fmt.Print(propType.Format(0))
						propTypes = append(propTypes, propType)
					} else {
						fmt.Print("null")
					}
					fmt.Println(";")
				}
			}
			fmt.Println()
		}
	}
	return propTypes
}

func inspectExpr(expr ast.Expr) (proptypes.PropType, bool) {
	switch typ := expr.(type) {
	case *ast.Ident:
		return inspectIdent(typ)
	case *ast.ParenExpr:
	case *ast.SelectorExpr:
	case *ast.StarExpr:
		return inspectExpr(typ.X)
	case *ast.ArrayType:
		return inspectArray(typ.Elt)
	case *ast.InterfaceType:
		// Interfaces are dynamic, so it would be difficult to detect all the types
		// that satisfy the interface in the codebase. Ideally, this would be a
		// OneOfType of all types satisfying the interface rather than Object.
		return proptypes.Object, true
	case *ast.MapType:
		if t, ok := inspectExpr(typ.Value); ok {
			return proptypes.ObjectOf(t), true
		}
		return proptypes.Object, true
	case *ast.StructType:
		return inspectStruct(typ)
	case *ast.ChanType, *ast.FuncType:
		// Channels and funcs cannot encoded in json. Attempting to encode such a
		// value causes jsonMarshal to return a json.UnsupportedTypeError.
		return nil, false
	default:
		fmt.Println("Other")
		return testPropType{fmt.Sprint("UNMATCHED", reflect.TypeOf(expr).Elem().Name())}, true
	}
	return testPropType{expr}, true
}

func inspectIdent(id *ast.Ident) (proptypes.PropType, bool) {
	if id.Obj == nil {
		// TODO(aa) Is there a better way of detecting Go primitives?
		// List of primitives taken from https://golang.org/ref/spec#Types.
		switch id.Name {
		case "bool":
			return proptypes.Bool, true
		case "uint8", "uint16", "uint32", "uint64",
			"int8", "int16", "int32", "int64",
			"float32", "float64",
			"byte", "rune",
			"uint", "int", "uintptr":
			return proptypes.Number, true
		case "complex64", "complex128":
			return nil, false
		case "string":
			return proptypes.String, true
		}
	}
	return testPropType{id}, true
}

// inspectArray returns a prop type representing an array. As a special case,
// []byte is encoded to json as a string. For any invalid element types, an
// untyped array is returned.
func inspectArray(elem ast.Expr) (proptypes.PropType, bool) {
	if e, ok := elem.(*ast.Ident); ok && e.Name == "byte" && e.Obj == nil {
		return proptypes.String, true
	}
	if t, ok := inspectExpr(elem); ok {
		return proptypes.ArrayOf(t), true
	}
	return proptypes.Array, true
}

func inspectStruct(typ *ast.StructType) (proptypes.PropType, bool) {
	var shape proptypes.ShapeMap
	if typ.Fields != nil {
		for _, field := range typ.Fields.List {
			propType, ok := inspectExpr(field.Type)
			if !ok {
				continue
			}
			for _, name := range field.Names {
				if !name.IsExported() { // undetectable in json marshal
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
				_ = omitEmpty
				shape = append(shape, proptypes.ShapeEntry{Name: fieldName, Type: propType})
			}
		}
	}
	return proptypes.Shape(shape), true
}
