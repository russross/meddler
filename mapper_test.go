package meddler

import (
	"testing"
)

func TestDefaultMapper(t *testing.T) {
	// default mapper should be no-op
	var tests = map[string]string{
		"":        "",
		"foo":     "foo",
		"foo_bar": "foo_bar",
		"FooBar":  "FooBar",
		"FOOBAR":  "FOOBAR",
	}

	for i, e := range tests {
		if v := Mapper(i); v != e {
			t.Errorf("Mapper(\"%s\"): expected %s, got %s", i, e, v)
		}
	}

}

func TestSnakeCase(t *testing.T) {
	var tests = map[string]string{
		"":            "",
		"ID":          "id",
		"ColumnName":  "column_name",
		"COLUMN_NAME": "column_name",
		"column_name": "column_name",
		"UserID":      "user_id",
		"UserNameRaw": "user_name_raw",
	}

	for i, e := range tests {
		if v := SnakeCase(i); v != e {
			t.Errorf("SnakeCase(\"%s\"): expected %s, got %s", i, e, v)
		}
	}
}

func TestLowerCase(t *testing.T) {
	var tests = map[string]string{
		"":            "",
		"ID":          "id",
		"ColumnName":  "columnname",
		"COLUMN_NAME": "column_name",
		"column_name": "column_name",
		"UserID":      "userid",
		"UserNameRaw": "usernameraw",
	}

	for i, e := range tests {
		if v := LowerCase(i); v != e {
			t.Errorf("LowerCase(\"%s\"): expected %s, got %s", i, e, v)
		}
	}

}
