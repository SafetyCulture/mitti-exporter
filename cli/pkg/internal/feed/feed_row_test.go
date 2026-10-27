package feed

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// The columns and primary keys of a rowFeed are declared by hand, so make sure they agree with the row struct.
// A mismatch would only surface when upserting rows, as a SQL error.
func TestRowFeeds_columns_should_match_row_model(t *testing.T) {
	client := NewExporterApp(nil, nil, &ExporterFeedCfg{})

	checked := 0
	for _, f := range client.GetFeeds() {
		if !strings.Contains(fmt.Sprintf("%T", f), "rowFeed") {
			continue
		}
		checked++

		t.Run(f.Name(), func(t *testing.T) {
			modelType := reflect.TypeOf(f.Model())

			var modelColumns []string
			primaryKeys := map[string]bool{}
			for i := range modelType.NumField() {
				field := modelType.Field(i)
				column := field.Tag.Get("csv")
				assert.NotEmpty(t, column, "field %s has no csv tag", field.Name)
				modelColumns = append(modelColumns, column)
				if strings.Contains(field.Tag.Get("gorm"), "primarykey") {
					primaryKeys[column] = true
				}
			}

			declared := append(append([]string{}, f.PrimaryKey()...), f.Columns()...)
			sort.Strings(modelColumns)
			sort.Strings(declared)
			assert.Equal(t, modelColumns, declared, "PrimaryKey() + Columns() should list every column of the row")

			// gorm creates the table from the model while rows are written using the name of the feed
			tabler, ok := f.Model().(interface{ TableName() string })
			assert.True(t, ok, "model should define TableName()")
			if ok {
				assert.Equal(t, f.Name(), tabler.TableName())
			}

			assert.Len(t, primaryKeys, len(f.PrimaryKey()))
			for _, pk := range f.PrimaryKey() {
				assert.True(t, primaryKeys[pk], "%s should be tagged primarykey", pk)
			}
		})
	}

	assert.Equal(t, 17, checked, "expected all the rowFeed feeds to be registered")
}

func TestJSONArray_UnmarshalJSON(t *testing.T) {
	type row struct {
		Values JSONArray `json:"values"`
	}

	tests := map[string]string{
		`{"values":["a", "b"]}`: `["a","b"]`,
		`{"values":[]}`:         `[]`,
		`{"values":null}`:       ``,
		`{}`:                    ``,
	}
	for input, expected := range tests {
		var r row
		assert.NoError(t, jsonUnmarshal(input, &r), input)
		assert.Equal(t, JSONArray(expected), r.Values, input)
	}
}

func jsonUnmarshal(s string, v interface{}) error {
	return json.Unmarshal([]byte(s), v)
}
