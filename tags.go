package proper

import (
	"go/ast"
	"reflect"
	"strconv"
	"strings"
)

func lookupTag(field *ast.Field, key string) (string, bool) {
	if field == nil || field.Tag == nil {
		return "", false
	}
	tag, err := strconv.Unquote(field.Tag.Value)
	if err != nil {
		return "", false
	}
	return reflect.StructTag(tag).Lookup(key)
}

// tagOptions is the string following a comma in a struct field's "json"
// tag, or the empty string. It does not include the leading comma.
// Copied from: encoding/json.tagOptions.
type tagOptions string

// parseJSONTag splits a struct field's json tag into its name and
// comma-separated options.
// Copied from: encoding/json.parseTag.
func parseJSONTag(tag string) (string, tagOptions) {
	if idx := strings.Index(tag, ","); idx != -1 {
		return tag[:idx], tagOptions(tag[idx+1:])
	}
	return tag, tagOptions("")
}

// Contains reports whether a comma-separated list of options
// contains a particular substr flag. substr must be surrounded by a
// string boundary or commas.
// Copied from: encoding/json.tagOptions.Contains.
func (o tagOptions) Contains(optionName string) bool {
	if len(o) == 0 {
		return false
	}
	s := string(o)
	for s != "" {
		var next string
		i := strings.Index(s, ",")
		if i >= 0 {
			s, next = s[:i], s[i+1:]
		}
		if s == optionName {
			return true
		}
		s = next
	}
	return false
}
