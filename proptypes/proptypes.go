// Package proptypes models types from the JavaScript prop-types library.
package proptypes

import (
	"fmt"
	"strings"
)

// ImportName is the variable name by which the prop-types library is imported
// in JavaScript. It is set to PropTypes by default, but can be overridden.
var ImportName = "PropTypes"

// Indent is the characters to use for indentation. It is set to "  " by
// default, but can be overridden.
var Indent = "  "

// PropType represents a prop type from the React prop-types library that can
// be formatted with JavaScript syntax.
type PropType interface {
	Format(indent int) string
}

// ShapeMap describes the shape of a JavaScript object.
type ShapeMap []ShapeEntry

// ShapeEntry is a key-value pair in a ShapeMap.
type ShapeEntry struct {
	Name string
	Type PropType
}

type simple string
type instanceOf struct{ class string }
type oneOf struct{ literals []string }
type oneOfType struct{ types []PropType }
type arrayOf struct{ typ PropType }
type objectOf struct{ typ PropType }
type isRequired struct{ typ PropType }
type shape struct{ shape ShapeMap }
type exact struct{ shape ShapeMap }

const (
	// Any is a prop of any type.
	Any simple = "any"

	// Specific JavaScript primitive types.
	Array  simple = "array"
	Bool   simple = "bool"
	Func   simple = "func"
	Number simple = "number"
	Object simple = "object"
	String simple = "string"
	Symbol simple = "symbol"

	// Node is anything that can be rendered: numbers, strings, elements, or an
	// array or fragment containing these types.
	Node simple = "node"

	// Element is a React element (i.e. <MyComponent />).
	Element simple = "element"

	// ElementType is a React element type (i.e. MyComponent).
	ElementType simple = "elementType"
)

var _ PropType = simple("")

// InstanceOf is a prop that is an instance of a class.
func InstanceOf(class string) PropType { return &instanceOf{class} }

// OneOf is a a prop that is limited to specific values by treating it as an
// enum. The given literals must be formatted in JavaScript syntax, including
// quotes for strings (i.e. "'lorem'" for the JavaScript string value 'lorem').
func OneOf(literals ...string) PropType { return &oneOf{literals} }

// OneOfType is an object that could be one of many types.
func OneOfType(types ...PropType) PropType { return &oneOfType{types} }

// ArrayOf is an array of a certain type.
func ArrayOf(typ PropType) PropType { return &arrayOf{typ} }

// ObjectOf is an object with property values of a certain shape.
func ObjectOf(typ PropType) PropType { return &objectOf{typ} }

// Shape is an object taking on a particular shape.
func Shape(s ShapeMap) PropType { return &shape{s} }

// Exact is an object with warnings on extra properties.
func Exact(s ShapeMap) PropType { return &exact{s} }

// IsRequired is that a prop that is required.
func IsRequired(typ PropType) PropType { return &isRequired{typ} }

func (s simple) Format(indent int) string {
	return ImportName + "." + string(s)
}

func (i *instanceOf) Format(indent int) string {
	return formatTypeFunc("instanceOf", i.class)
}

func (o *oneOf) Format(indent int) string {
	return formatTypeFunc("oneOf", formatArray(o.literals, indent))
}

func (o *oneOfType) Format(indent int) string {
	v := make([]string, len(o.types))
	for i, typ := range o.types {
		if typ == nil {
			v[i] = "null"
		} else {
			v[i] = typ.Format(indent + 1)
		}
	}
	return formatTypeFunc("oneOfType", formatArray(v, indent))
}

func (a *arrayOf) Format(indent int) string {
	t := "null"
	if a.typ != nil {
		t = a.typ.Format(indent)
	}
	return formatTypeFunc("arrayOf", t)
}

func (o *objectOf) Format(indent int) string {
	t := "null"
	if o.typ != nil {
		t = o.typ.Format(indent)
	}
	return formatTypeFunc("objectOf", t)
}

func (s *shape) Format(indent int) string {
	return formatTypeFunc("shape", formatShape(s.shape, indent))
}

func (e *exact) Format(indent int) string {
	return formatTypeFunc("exact", formatShape(e.shape, indent))
}

func (r *isRequired) Format(indent int) string {
	return r.typ.Format(indent) + ".isRequired"
}

func formatTypeFunc(name, value string) string {
	return fmt.Sprintf("%s.%s(%s)", ImportName, name, value)
}

// formatArray formats a slice of strings as a JavaScript array with values on
// separate lines and a trailing comma.
func formatArray(a []string, indent int) string {
	if len(a) == 0 {
		return "[]"
	}
	if len(a) == 1 && !strings.ContainsRune(a[0], '\n') {
		return "[" + a[0] + "]"
	}
	var b strings.Builder
	b.WriteString("[\n")
	in := indentation(indent + 1)
	for _, s := range a {
		b.WriteString(in)
		b.WriteString(s)
		b.WriteString(",\n")
	}
	b.WriteString(indentation(indent))
	b.WriteByte(']')
	return b.String()
}

// formatShape formats a ShapeMap as a JavaScript object with entries on
// separate lines and a trailing comma.
func formatShape(s ShapeMap, indent int) string {
	if len(s) == 0 {
		return "{}"
	}
	var b strings.Builder
	b.WriteString("{\n")
	in := indentation(indent + 1)
	for _, e := range s {
		b.WriteString(in)
		b.WriteString(e.Name)
		b.WriteString(": ")
		if e.Type == nil {
			b.WriteString("null")
		} else {
			b.WriteString(e.Type.Format(indent + 1))
		}
		b.WriteString(",\n")
	}
	b.WriteString(indentation(indent))
	b.WriteByte('}')
	return b.String()
}

func indentation(level int) string {
	return strings.Repeat(Indent, level)
}
