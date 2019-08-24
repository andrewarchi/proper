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

// PropTypeDecl is a declaration of a prop type in JavaScript.
type PropTypeDecl struct {
	Name *ast.Ident
	Pos  token.Pos
	Typ  proptypes.PropType
}

// Format formats a prop type declaration as a JavaScript const declaration.
func (d *PropTypeDecl) Format(fset *token.FileSet) string {
	t := "null"
	if d.Typ != nil {
		t = d.Typ.Format(0)
	}
	return fmt.Sprintf("// %v\nconst %s = %s;", fset.Position(d.Pos), d.Name, t)
}

var _ proptypes.PropType = (*selectorPropType)(nil)
var _ proptypes.PropType = (*testPropType)(nil)

type selectorPropType struct {
	sel *ast.SelectorExpr
}

func (s *selectorPropType) Format(indent int) string {
	panic("unimplemented")
}

type testPropType struct{ val interface{} }

func (t *testPropType) Format(indent int) string {
	if t.val == nil {
		return "<<nil>>"
	}
	if id, ok := t.val.(*ast.Ident); ok {
		o := id.Obj
		if o == nil {
			return fmt.Sprintf("<<%T %q: nil Obj>>", t.val, id.Name)
		}
		return fmt.Sprintf("<<%T %q: %v, %q, %v, %v, %v>>", t.val, id.Name, o.Kind, o.Name, o.Decl, o.Data, o.Type)
	}
	return fmt.Sprintf("<<%[1]T %[1]v>>", t.val)
}

func InspectDirRecursive(root string, fset *token.FileSet, types map[string][]*PropTypeDecl) error {
	return filepath.Walk(root, func(path string, f os.FileInfo, err error) error {
		if f.IsDir() {
			return InspectDir(path, fset, types)
		}
		return nil
	})
}

func InspectDir(dir string, fset *token.FileSet, types map[string][]*PropTypeDecl) error {
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

func inspectFile(fset *token.FileSet, f *ast.File) []*PropTypeDecl {
	var propDecls []*PropTypeDecl
	for _, decl := range f.Decls {
		if typeDecl, ok := decl.(*ast.GenDecl); ok && typeDecl.Tok == token.TYPE {
			for _, s := range typeDecl.Specs {
				if spec, ok := s.(*ast.TypeSpec); ok {
					propType, _ := inspectExpr(spec.Type)
					propDecls = append(propDecls, &PropTypeDecl{spec.Name, typeDecl.Pos(), propType})
				}
			}
		}
	}
	return propDecls
}

func inspectExpr(expr ast.Expr) (proptypes.PropType, bool) {
	switch typ := expr.(type) {
	case *ast.Ident:
		return inspectIdent(typ)
	case *ast.ParenExpr:
	case *ast.SelectorExpr:
		return inspectSelector(typ)
	case *ast.StarExpr:
		return inspectExpr(typ.X)
	case *ast.ArrayType:
		return inspectArray(typ.Elt)
	case *ast.InterfaceType:
		// Interfaces are dynamic, so it would be difficult to detect all the types
		// that satisfy the interface in the codebase. Ideally, this would be a
		// OneOfType specifying all types satisfying the interface rather than Any.
		return proptypes.Any, true
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
		return &testPropType{fmt.Sprint("UNMATCHED", reflect.TypeOf(expr).Elem().Name())}, true
	}
	return &testPropType{expr}, true
}

func inspectSelector(expr *ast.SelectorExpr) (proptypes.PropType, bool) {
	if x, ok := expr.X.(*ast.Ident); ok && x != nil && x.Obj != nil {
		fmt.Println("XOBJ", x.Obj)
	}
	if expr.Sel != nil && expr.Sel.Obj != nil {
		fmt.Println("SELOBJ", expr.Sel.Obj)
	}
	fmt.Println("SELECTOR", expr)
	if expr.X != nil && expr.Sel != nil {
		if id, ok := expr.X.(*ast.Ident); ok && id.Obj == nil && expr.Sel.Obj == nil {
			sel := expr.Sel.Name
			switch id.Name {
			case "time":
				switch sel {
				case "Time":
					return proptypes.String, true
				}
			case "bson":
				switch sel {
				case "ObjectId":
					return proptypes.String, true
				}
			}
		}
	}
	return nil, false
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
	return &testPropType{id}, true
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
				if !name.IsExported() { // Undetectable by json encoder
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
					// TODO(aa) add PropType support for omitempty
					omitEmpty = options.Contains("omitempty")
					_ = omitEmpty
				}
				shape = append(shape, proptypes.ShapeEntry{Name: fieldName, Type: propType})
			}
		}
	}
	return proptypes.Shape(shape), true
}
