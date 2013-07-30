package sqlscan

import (
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// the name of our struct tag
const tagName = "sqlscan"

// Quote is the quote character for table and column names. "`" works for sqlite and mysql, "\"" for postgresql
var Quote = "`"

// Placeholder is the SQL value placeholder. "?" works for sqlite and mysql, "$1" for postgresql
var Placeholder = "?"

// Debug enables debug mode, where unused columns and struct fields will be logged
var Debug = true

type structField struct {
	column     string
	index      int
	primaryKey bool
	meddler    Meddler
}

type structData struct {
	columns []string
	fields  map[string]*structField
	pk      string
}

// cache reflection data
var fieldsCache = make(map[reflect.Type]*structData)

// getFields gathers the list of columns from a struct using reflection.
func getFields(dstType reflect.Type) (*structData, error) {
	if result, present := fieldsCache[dstType]; present {
		return result, nil
	}

	// make sure dst is a non-nil pointer to a struct
	if dstType.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("sqlscan called with non-pointer destination %v", dstType)
	}
	structType := dstType.Elem()
	if structType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("sqlscan called with pointer to non-struct %v", dstType)
	}

	// gather the list of fields in the struct
	data := new(structData)
	data.fields = make(map[string]*structField)

	for i := 0; i < structType.NumField(); i++ {
		f := structType.Field(i)

		// skip non-exported fields
		if f.PkgPath != "" {
			continue
		}

		// examine the tag for metadata
		tag := strings.Split(f.Tag.Get(tagName), ",")

		// was this field marked for skipping?
		if len(tag) > 0 && tag[0] == "-" {
			continue
		}

		// default to the field name
		name := f.Name

		// the tag can override the field name
		if len(tag) > 0 && tag[0] != "" {
			name = tag[0]
		}

		// check for a meddler
		var meddler Meddler = registry["identity"]
		for j := 1; j < len(tag); j++ {
			if tag[j] == "pk" {
				if f.Type.Kind() == reflect.Ptr {
					return nil, fmt.Errorf("sqlscan found field %s which is marked as the primary key but is a pointer", f.Name)
				}

				// make sure it is an int of some kind
				switch f.Type.Kind() {
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				default:
					return nil, fmt.Errorf("sqlscan found field %s which is marked as the primary key, but is not an integer type", f.Name)
				}

				if data.pk != "" {
					return nil, fmt.Errorf("sqlscan found field %s which is marked as the primary key, but a primary key field was already found", f.Name)
				}
				data.pk = name
			} else if m, present := registry[tag[j]]; present {
				meddler = m
			} else {
				return nil, fmt.Errorf("sqlscan found field %s with meddler %s, but that meddler is not registered", f.Name, tag[j])
			}
		}

		if _, present := data.fields[name]; present {
			return nil, fmt.Errorf("sqlscan found multiple fields for column %s", name)
		}
		data.fields[name] = &structField{
			column:     name,
			primaryKey: name == data.pk,
			index:      i,
			meddler:    meddler,
		}
		data.columns = append(data.columns, name)
	}

	fieldsCache[dstType] = data
	return data, nil
}

// Columns returns a list of column names for its input struct.
// Will panic if the struct contains fields that it does not know how to handle.
func Columns(includePk bool, src interface{}) []string {
	data, err := getFields(reflect.TypeOf(src))
	if err != nil {
		panic("sqlscan.Columns: error scanning fields of struct: " + err.Error())
	}

	var names []string
	for _, elt := range data.columns {
		if !includePk && elt == data.pk {
			continue
		}
		names = append(names, elt)
	}

	return names
}

// ColumnsQuoted is similar to Columns, but it return the list of columns in the form:
//   `column1`,`column2`,...
// using Quote as the quote character.
func ColumnsQuoted(includePk bool, src interface{}) string {
	var unquoted []string
	unquoted = Columns(includePk, src)

	var quoted []string
	for _, elt := range unquoted {
		quoted = append(quoted, Quote+elt+Quote)
	}

	return strings.Join(quoted, ",")
}

// PrimaryKey returns the name and value of the primary key field. The name
// is the empty string if there is not primary key field marked.
func PrimaryKey(src interface{}) (name string, pk int) {
	data, err := getFields(reflect.TypeOf(src))
	if err != nil {
		panic("sqlscan.PrimaryKey: error scanning fields of struct: " + err.Error())
	}

	if data.pk == "" {
		return "", 0
	}

	name = data.pk
	pk = int(reflect.ValueOf(src).Elem().Field(data.fields[name].index).Int())

	return name, pk
}

// SetPrimaryKey sets the primary key field to the given int value.
// Will panic if there is not primary key, or if it is not of an integer type.
func SetPrimaryKey(pk int, src interface{}) {
	data, err := getFields(reflect.TypeOf(src))
	if err != nil {
		panic("sqlscan.SetPrimaryKey: error scanning fields of struct: " + err.Error())
	}

	if data.pk == "" {
		panic("sqlscan.SetPrimaryKey: no primary key field found")
	}

	reflect.ValueOf(src).Elem().Field(data.fields[data.pk].index).SetInt(int64(pk))
}

// SaveValues returns a list of PreWrite processed values suitable for
// use in an INSERT or UPDATE query. If includePk is false, the primary
// key field is omitted.
func SaveValues(includePk bool, src interface{}) ([]interface{}, error) {
	data, err := getFields(reflect.TypeOf(src))
	if err != nil {
		return nil, err
	}
	structVal := reflect.ValueOf(src).Elem()

	var values []interface{}
	for _, name := range data.columns {
		if !includePk && name == data.pk {
			continue
		}
		field := data.fields[name]
		saveVal, err := field.meddler.PreWrite(structVal.Field(field.index).Interface())
		if err != nil {
			return nil, fmt.Errorf("sqlscan.SaveValues: PreWrite error on column %s: %v", field.column, err)
		}
		values = append(values, saveVal)
	}

	return values, nil
}

// SavePlaceholders returns a list of placeholders suitable for an INSERT or UPDATE query.
// If includePk is false, the primary key field is omitted.
func SavePlaceholders(includePk bool, src interface{}) []string {
	data, err := getFields(reflect.TypeOf(src))
	if err != nil {
		panic("sqlscan.SavePlaceholdersList: error scanning fields of struct: " + err.Error())
	}

	var placeholders []string
	for _, name := range data.columns {
		if !includePk && name == data.pk {
			continue
		}
		ph := strings.Replace(Placeholder, "1", strconv.FormatInt(int64(len(placeholders)+1), 10), 1)
		placeholders = append(placeholders, ph)
	}

	return placeholders
}

// SavePlaceholdersString returns a list of placeholders suitable for an INSERT
// query in string form, e.g.:
//   ?,?,?,?
// if includePk is false, the primary key field is omitted.
func SavePlaceholdersString(includePk bool, src interface{}) string {
	return strings.Join(SavePlaceholders(includePk, src), ",")
}

// scan a single row of data into a struct.
func scanRow(rows *sql.Rows, data *structData, columns []string, dst interface{}) error {
	// check if there is data waiting
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return err
		}
		return sql.ErrNoRows
	}

	structVal := reflect.ValueOf(dst).Elem()

	// prepare a list of targets
	var targets []interface{}
	for _, column := range columns {
		if field, present := data.fields[column]; present {
			fieldAddr := structVal.Field(field.index).Addr().Interface()
			scanTarget, err := field.meddler.PreRead(fieldAddr)
			if err != nil {
				return fmt.Errorf("sqlscan.scanRow: error on column %s: %v", field.column, err)
			}
			targets = append(targets, scanTarget)
		} else {
			// no destination, so throw this away
			targets = append(targets, &sql.RawBytes{})
		}
	}

	// perform the scan
	if err := rows.Scan(targets...); err != nil {
		return err
	}

	// post-process
	for i, column := range columns {
		if field, present := data.fields[column]; present {
			fieldAddr := structVal.Field(field.index).Addr().Interface()
			err := field.meddler.PostRead(fieldAddr, targets[i])
			if err != nil {
				return fmt.Errorf("sqlscan.scanRow: error on column %s: %v", field.column, err)
			}
		}
	}

	return rows.Err()
}

// Scan scans a single sql result row into a struct.
// It leaves rows ready to be scanned again for the next row.
// Returns sql.ErrNoRows if there is no data to read.
func Scan(rows *sql.Rows, dst interface{}) error {
	// get the list of struct fields
	data, err := getFields(reflect.TypeOf(dst))
	if err != nil {
		return err
	}

	// get the sql columns
	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	return scanRow(rows, data, columns, dst)
}

// ScanRow scans a single sql result row into a struct.
// It reads exactly one result row and closes rows when finished.
// Returns sql.ErrNoRows if there is no result row.
func ScanRow(rows *sql.Rows, dst interface{}) error {
	// make sure we always close rows
	defer rows.Close()

	if err := Scan(rows, dst); err != nil {
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}

	return nil
}

// ScanAll scans all sql result rows into a slice of structs.
// It reads all rows and closes rows when finished.
// dst should be a pointer to a slice of the appropriate type.
// The new results will be appended to any existing data in dst.
func ScanAll(rows *sql.Rows, dst interface{}) error {
	// make sure we always close rows
	defer rows.Close()

	// make sure dst is an appropriate type
	dstVal := reflect.ValueOf(dst)
	if dstVal.Kind() != reflect.Ptr || dstVal.IsNil() {
		return fmt.Errorf("ScanAll called with non-pointer destination: %T", dst)
	}
	sliceVal := dstVal.Elem()
	if sliceVal.Kind() != reflect.Slice {
		return fmt.Errorf("ScanAll called with pointer to non-slice: %T", dst)
	}
	ptrType := sliceVal.Type().Elem()
	if ptrType.Kind() != reflect.Ptr {
		return fmt.Errorf("ScanAll expects element to be pointers, found %T", dst)
	}
	eltType := ptrType.Elem()
	if eltType.Kind() != reflect.Struct {
		return fmt.Errorf("ScanAll expects element to be pointers to structs, found %T", dst)
	}

	// get the list of struct fields
	data, err := getFields(ptrType)
	if err != nil {
		return err
	}

	// get the sql columns
	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	// gather the results
	for {
		// create a new element
		eltVal := reflect.New(eltType)
		elt := eltVal.Interface()

		// scan it
		if err := scanRow(rows, data, columns, elt); err != nil {
			if err == sql.ErrNoRows {
				return nil
			}
			return err
		}

		// add to the result slice
		sliceVal.Set(reflect.Append(sliceVal, eltVal))
	}
}
