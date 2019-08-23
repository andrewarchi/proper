package proptypes

import "testing"

func TestFormat(t *testing.T) {
	for _, test := range []struct {
		typ  PropType
		want string
	}{
		{Any, "PropTypes.any"},
		{Array, "PropTypes.array"},
		{Bool, "PropTypes.bool"},
		{Func, "PropTypes.func"},
		{Number, "PropTypes.number"},
		{Object, "PropTypes.object"},
		{String, "PropTypes.string"},
		{Symbol, "PropTypes.symbol"},
		{Node, "PropTypes.node"},
		{Element, "PropTypes.element"},
		{ElementType, "PropTypes.elementType"},
		{InstanceOf("Date"), "PropTypes.instanceOf(Date)"},
		{OneOf(), "PropTypes.oneOf([])"},
		{OneOf("3.14"), `PropTypes.oneOf([3.14])`},
		{OneOf("'lorem'", `"ipsum"`, "123"), `PropTypes.oneOf([
      'lorem',
      "ipsum",
      123,
    ])`},
		{OneOfType(nil), "PropTypes.oneOfType([null])"},
		{OneOfType(Number, String), `PropTypes.oneOfType([
      PropTypes.number,
      PropTypes.string,
    ])`},
		{ArrayOf(Symbol), "PropTypes.arrayOf(PropTypes.symbol)"},
		{ObjectOf(nil), "PropTypes.objectOf(null)"},
		{ObjectOf(String), "PropTypes.objectOf(PropTypes.string)"},
		{Shape(ShapeMap{ShapeEntry{"error", nil}}), `PropTypes.shape({
      error: null,
    })`},
		{Shape(ShapeMap{ShapeEntry{"callback", Func}}), `PropTypes.shape({
      callback: PropTypes.func,
    })`},
		{Shape(ShapeMap{
			ShapeEntry{"name", IsRequired(String)},
			ShapeEntry{"address", Shape(ShapeMap{
				ShapeEntry{"street", String},
				ShapeEntry{"city", String},
				ShapeEntry{"zip", String},
			})},
			ShapeEntry{"email", String},
		}), `PropTypes.shape({
      name: PropTypes.string.isRequired,
      address: PropTypes.shape({
        street: PropTypes.string,
        city: PropTypes.string,
        zip: PropTypes.string,
      }),
      email: PropTypes.string,
    })`},
		{Exact(nil), `PropTypes.exact({})`},
		{Exact(ShapeMap{
			ShapeEntry{"iterator", Symbol},
			ShapeEntry{"match", Symbol},
		}), `PropTypes.exact({
      iterator: PropTypes.symbol,
      match: PropTypes.symbol,
    })`},
		{IsRequired(Node), "PropTypes.node.isRequired"},
	} {
		if got := test.typ.Format(2); got != test.want {
			t.Errorf("(%v).Format(2) = %q; want %q", test.typ, got, test.want)
		}
	}
}

func TestOverrides(t *testing.T) {
	im, in := ImportName, Indent
	ImportName = "PROPTYPES"
	Indent = "    "
	for _, test := range []struct {
		typ  PropType
		want string
	}{
		{String, "PROPTYPES.string"},
		{Shape(ShapeMap{
			ShapeEntry{"salutation", OneOf("'Hello, world!'", "'Hallo Welt!'")},
		}), `PROPTYPES.shape({
        salutation: PROPTYPES.oneOf([
            'Hello, world!',
            'Hallo Welt!',
        ]),
    })`},
	} {
		if got := test.typ.Format(1); got != test.want {
			t.Errorf("(%v).Format(1) = %q; want %q", test.typ, got, test.want)
		}
	}
	ImportName, Indent = im, in
}
